package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/golang/glog"
	uuid "github.com/satori/go.uuid"
)

const (
	errNotImp     = "Not Implemented."
	errNoHTTPHost = "No Host header found in buffered HTTP header (%d bytes)"
	errNotTLS     = "Communication is not TLS"
	errNoContent  = "Nothing recieved"

	hostHeader = "Host: "

	sslHeaderLen     = 5
	sslTypeHandshake = 0x16
)

var cfg conf

// HTTPSDestination detect HTTPS destination via SNI
// from https://github.com/google/tcpproxy/blob/de1c7de/sni.go#L156
func HTTPSDestination(id string, br *bufio.ReadWriter) (hostname string, buff []byte, err error) {
	// peek into the stream
	buff, err = br.Peek(sslHeaderLen)
	if err != nil {
		glog.Warningf("[%s] %s", id, err)
	}
	if len(buff) == 0 {
		return "", buff, errors.New(errNoContent)
	}
	if buff[0] != sslTypeHandshake {
		return "", buff, errors.New(errNotTLS)
	}
	recLen := int(buff[3])<<8 | int(buff[4]) // ignoring version in hdr[1:3]
	buff, err = br.Peek(sslHeaderLen + recLen)
	if err != nil {
		return "", buff, err
	}
	tls.Server(sniSniffConn{r: bytes.NewReader(buff)}, &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			hostname = hello.ServerName
			return nil, nil
		},
	}).Handshake()
	glog.Infof("[%s] Peeked a host: %s", id, hostname)
	return hostname, buff, nil
}

// HTTPDestination detect HTTP destination in headers
func HTTPDestination(id string, br *bufio.ReadWriter) (hostname string, buff []byte, err error) {
	glog.Info("Peeking for destination in http")
	// peek into the stream
	index := 1
	lastindex := index
	var char byte
	for index <= cfg.Buffer {
		buff, err := br.Peek(index)
		if err != nil {
			glog.Warningf("[%s] %s", id, err)
		}
		if len(buff) < index {
			return "", buff, errors.New(errNoContent)
		}
		char = buff[index-1]
		if char == '\r' {
			//begin of new line
			line := string(buff[lastindex : index-1])
			if strings.Compare("", line) == 0 {
				return "", buff, fmt.Errorf(errNoHTTPHost, cfg.Buffer)
			}
			if strings.HasPrefix(line, hostHeader) {
				hostname = strings.TrimPrefix(line, hostHeader)
				glog.Infof("[%s] Peeked a host: %s", id, hostname)
				return hostname, buff, nil
			}
			lastindex = index
		}
		if char == '\n' {
			lastindex = index
		}
		index++
	}
	return "", buff, fmt.Errorf(errNoHTTPHost, cfg.Buffer)
}

//listen on defined port an forward to detected host by detectdest function
func listen(port string, detectdest func(string, *bufio.ReadWriter) (string, []byte, error)) {
	glog.Infof("Listening on port %s", port)
	l, err := net.Listen("tcp", port)
	if err != nil {
		glog.Fatal(err)
	}
	defer l.Close()
	for {
		id := uuid.Must(uuid.NewV4())
		c, err := l.Accept()
		if err != nil {
			glog.Warningf("[%s] %s", id, err)
			c.Close()
			continue
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(time.Duration(cfg.Timeout) * time.Second))
		glog.Infof("[%s] Request: %s->%s", id, c.RemoteAddr().String(), port)
		bufferReader := bufio.NewReader(c)
		bufferWriter := bufio.NewWriter(c)
		bufferIo := bufio.NewReadWriter(bufferReader, bufferWriter)
		dest, _, err := detectdest(id.String(), bufferIo)
		if err != nil {
			glog.Warningf("[%s] %s", id, err)
			c.Close()
			continue
		}
		glog.Infof("[%s] Routing: %s->%s->%s", id, c.RemoteAddr().String(), port, dest)
		go forward(id.String(), bufferIo, dest+port)
	}
}

// IsLoopback returns true if the name only resolves to loopback addresses.
func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	// Check for loopback network.
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if !ip.IsLoopback() {
			return false
		}
	}
	return true
}

//forward connection
func forward(id string, bufferIo *bufio.ReadWriter, dst string) {
	// get hostname and port
	if isLoopback(dst) {
		glog.Warningf("[%s] not forwarding to loopback", id)
		return
	}
	// forward
	glog.Infof("[%s] Forwarding to %s", id, dst)
	f, err := net.Dial("tcp", dst)
	if err != nil {
		glog.Errorf("[%s] %s", id, err)
		return
	}

	// set deadlines
	f.SetWriteDeadline(time.Now().Add(time.Duration(cfg.Timeout*2) * time.Second))
	f.SetReadDeadline(time.Now().Add(time.Duration(cfg.Timeout*2) * time.Second))

	// close when finished
	defer f.Close()

	glog.Infof("[%s] Copying the rest of IOs", id)

	// coordonate read writes
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		b, err := io.Copy(f, bufferIo)
		glog.Infof("[%s] Copied %d bytes to %s", id, b, f.RemoteAddr().String())
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if strings.Compare(neterr.Op, "write") == 0 {
					glog.Warningf("[%s] %s", id, err)
				}
			} else {
				glog.Warningf("[%s] %s", id, err)
			}
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		b, err := io.Copy(bufferIo, f)
		glog.Infof("[%s] Copied %d bytes from %s", id, b, f.RemoteAddr().String())
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if strings.Compare(neterr.Op, "write") == 0 {
					glog.Warningf("[%s] %s", id, err)
				}
			} else {
				glog.Warningf("[%s] %s", id, err)
			}
		}
		wg.Done()
	}()
	// wait for intput and output copy
	wg.Wait()
	// close the connection
	f.Close()
	// notify end of transfer
	glog.Infof("[%s] Forwarding to %s done", id, dst)
}

func main() {
	// declare flags
	var cfgfile string
	flag.StringVar(&cfgfile, "c", "config.yaml", "config file")
	flag.Set("logtostderr", "true")
	flag.Parse()
	// read config file
	glog.Info("Reading config file: ", cfgfile)
	data, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		glog.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		glog.Fatal(err)
	}

	for _, d := range cfg.Listen.Http {
		go listen(":"+d, HTTPDestination)
	}
	for _, d := range cfg.Listen.Https {
		go listen(":"+d, HTTPSDestination)
	}

	// wait
	select {}
}
