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

// from https://github.com/google/tcpproxy/blob/de1c7de/sni.go#L156
func HTTPSDestination(br *bufio.ReadWriter) (hostname string, buff []byte, err error) {
	// peek into the stream
	buff, err = br.Peek(sslHeaderLen)
	if err != nil {
		glog.Warning(err)
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
	return hostname, buff, nil
}

// HTTPDestination
func HTTPDestination(br *bufio.ReadWriter) (hostname string, buff []byte, err error) {
	glog.Info("Peeking for destination in http")
	// peek into the stream
	index := 1
	lastindex := index
	var char byte
	for index <= cfg.Buffer {
		buff, err := br.Peek(index)
		if err != nil {
			glog.Warning(err)
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
				glog.Infof("Peeked a host: %s", hostname)
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
func listen(port string, detectdest func(*bufio.ReadWriter) (string, []byte, error)) {
	glog.Infof("Listening on port %s", port)
	l, err := net.Listen("tcp", port)
	if err != nil {
		glog.Fatal(err)
	}
	defer l.Close()
	for {
		c, err := l.Accept()
		if err != nil {
			glog.Warning(err)
			c.Close()
			continue
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(time.Duration(cfg.Timeout) * time.Second))
		glog.Infof("Connection: %s->%s", c.RemoteAddr().String(), port)
		bufferReader := bufio.NewReader(c)
		bufferWriter := bufio.NewWriter(c)
		bufferIo := bufio.NewReadWriter(bufferReader, bufferWriter)
		dest, _, err := detectdest(bufferIo)
		if err != nil {
			glog.Warning(err)
			c.Close()
			continue
		}
		glog.Infof("Connection: %s->%s->%s", c.RemoteAddr().String(), port, dest)
		go forward(bufferIo, dest+port)
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
func forward(bufferIo *bufio.ReadWriter, dst string) {
	// get hostname and port
	if isLoopback(dst) {
		glog.Warningf("not forwarding to loopback")
		return
	}
	// forward
	glog.Infof("Forwarding to %s", dst)
	f, err := net.Dial("tcp", dst)
	if err != nil {
		glog.Error(err)
		return
	}

	// set deadlines
	f.SetWriteDeadline(time.Now().Add(time.Duration(cfg.Timeout*2) * time.Second))
	f.SetReadDeadline(time.Now().Add(time.Duration(cfg.Timeout*2) * time.Second))

	// close when finished
	defer f.Close()

	glog.Info("Copying the rest of IOs")

	// coordonate read writes
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		b, err := io.Copy(f, bufferIo)
		glog.Infof("Copied %d bytes to %s", b, f.RemoteAddr().String())
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if strings.Compare(neterr.Op, "write") == 0 {
					glog.Warning(err)
				}
			} else {
				glog.Warning(err)
			}
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		b, err := io.Copy(bufferIo, f)
		glog.Infof("Copied %d bytes from %s", b, f.RemoteAddr().String())
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if strings.Compare(neterr.Op, "write") == 0 {
					glog.Warning(err)
				}
			} else {
				glog.Warning(err)
			}
		}
		wg.Done()
	}()
	// wait for intput and output copy
	wg.Wait()
	// close the connection
	f.Close()
	// notify end of transfer
	glog.Infof("Forwarding to %s done", dst)
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
