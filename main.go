package main

import (
	"flag"
	"github.com/go-yaml/yaml"
	"github.com/golang/glog"
	"io/ioutil"
	"net"
	"bufio"
	"errors"
	"io"
	"time"
	"bytes"
	"strings"
	"fmt"
	"crypto/tls"
)

const (
	errNotImp = "Not Implemented."
	errNoHttpHost = "No Host header found in buffered HTTP header (%d bytes)"
	errNotTLS = "Communication is not TLS"
	errNoContent = "Nothing recieved"

	hostHeader = "Host: "

	sslHeaderLen = 5
	sslTypeHandshake = 0x16
)

var cfg conf

// from https://github.com/google/tcpproxy/blob/de1c7de/sni.go#L156
func HttpsDestination(br *bufio.Reader) (hostname string,err error) {
	// peek into the stream
	buff, err := br.Peek(sslHeaderLen)
    if err != nil  {
		glog.Warning(err)
	}
	if len(buff)==0 {
		return "",errors.New(errNoContent)
	}
	if buff[0] != sslTypeHandshake {
		return "",errors.New(errNotTLS)
	}
	recLen := int(buff[3])<<8 | int(buff[4]) // ignoring version in hdr[1:3]
	helloBytes, err := br.Peek(sslHeaderLen + recLen)
	if err != nil {
		return "",err
	}
	tls.Server(sniSniffConn{r: bytes.NewReader(helloBytes)}, &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			hostname = hello.ServerName
			return nil, nil
		},
    }).Handshake()
	return hostname,nil
}

func HttpDestination(br *bufio.Reader) (hostname string, err error) {
	glog.Info("Peeking for destination in http")
	// create the buffer
	buff := bytes.NewBuffer(nil)
	// peek into the stream
	index := 1
	var char byte
	for (index <= cfg.Buffer) {
		tmp, err := br.Peek(index)
	    if err != nil  {
			glog.Warning(err)
		}
		if len(tmp)==index-1 {
			return "",errors.New(errNoContent)
		}
		char = tmp[index-1]
		index += 1
		if char == '\r' || char == '\n' {
			line := buff.String()
			if (strings.Compare("",line) == 0 && char == '\r') {
				return "",errors.New(fmt.Sprintf(errNoHttpHost,cfg.Buffer))
			}
			if (strings.HasPrefix(line,hostHeader)) {
				hostname = strings.TrimPrefix(line, hostHeader)
				glog.Infof("Peeked a host: %s",hostname)
				return hostname, nil
			}
			buff.Reset()
			continue
		}
		buff.WriteByte(char)
	}
	return "",errors.New(fmt.Sprintf(errNoHttpHost,cfg.Buffer))
}

//listen on defined port an forward to detected host by detectdest function
func listen(port string, detectdest func(*bufio.Reader) (string, error)) {
	glog.Infof("Listening on port %s", port)
	l, err := net.Listen("tcp", port)
	if err != nil {
		glog.Fatal(err)
	}
	defer l.Close()
	for{
		c, err := l.Accept()
		c.SetDeadline(time.Now().Add(time.Duration(cfg.Timeout) * time.Second))
		if err != nil {
			glog.Warning(err)
			continue
		}
		defer c.Close()
		glog.Infof("Connection: %s->%s", c.RemoteAddr().String(), port)
		dest, err := detectdest(bufio.NewReader(c))
		if err != nil {
			glog.Warning(err)
			c.Close()
			continue
		}
		glog.Infof("Connection: %s->%s->%s", c.RemoteAddr().String(), port, dest)
		go forward(c,dest + port)
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
func forward(c net.Conn, dst string) {
	// get hostname and port
	if (isLoopback(dst)) {
		glog.Warningf("not forwarding to loopback")
		c.Close()
		return
	}
	// forward
	glog.Infof("Forwarding: %s->%s", c.RemoteAddr().String(), dst)
	f, err := net.Dial("tcp", dst)
	if err != nil {
		glog.Error(err)
		return
	}
	defer f.Close()

	ch := make(chan struct{}, 2)

	go func() {
		io.Copy(f, c)
		ch <- struct{}{}
	}()

	go func() {
		io.Copy(c, f)
		ch <- struct{}{}
	}()

	<-ch
}

func main() {
	// declare flags
	var cfgfile string
	flag.StringVar(&cfgfile, "c", "config.yaml", "config file")
	flag.Set("logtostderr", "true")
	flag.Parse()
	// read config file
	glog.Info("Reading config file: ",cfgfile) 
	data, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		glog.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		glog.Fatal(err)
	}

	for _, d := range cfg.Listen.Http {
		go listen(":"+d,HttpDestination)
	}
	for _, d := range cfg.Listen.Https {
		go listen(":"+d,HttpsDestination)
	}

	// wait 
	select{}
}