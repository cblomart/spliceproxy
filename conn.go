package main

import (
	"bufio"
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/golang/glog"
	tunnel "github.com/rackerlabs/go-connect-tunnel"
	uuid "github.com/satori/go.uuid"
)

func handleconn(id string, c net.Conn, detectdest func(string, *bufio.ReadWriter, string) (string, bool, error)) {
	_, port, err := net.SplitHostPort(c.LocalAddr().String())
	if err != nil {
		glog.Warningf("[%s] %s", id, err)
		return
	}
	err = c.SetDeadline(time.Now().Add(time.Duration(cfg.Timeout) * time.Second))
	if err != nil {
		glog.Warningf("[%s] %s", id, err)
	}
	glog.Infof("[%s] Request: %s->%s", id, c.RemoteAddr().String(), ":"+port)
	bufferIo := bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))
	dest, direct, err := detectdest(id, bufferIo, ":"+port)
	if err != nil {
		glog.Warningf("[%s] %s", id, err)
		closeconn(id, c)
		return
	}
	go func() {
		forward(id, bufferIo, dest, direct)
		closeconn(id, c)
	}()
}

//listen on defined port an forward to detected host by detectdest function
func listen(addr string, detectdest func(string, *bufio.ReadWriter, string) (string, bool, error)) {
	glog.Infof("Listening on address %s", addr)
	l, err := net.Listen(proto, addr)
	if err != nil {
		glog.Fatal(err)
	}
	defer l.Close()
	// check port
	for {
		id := uuid.Must(uuid.NewV4()).String()
		c, err := l.Accept()
		if err != nil {
			glog.Warningf("[%s] %s", id, err)
			continue
		}
		go handleconn(id, c, detectdest)
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
	return false
}

// get connection
func getconn(id string, dst string, direct bool) (net.Conn, error) {
	if direct {
		glog.Infof("[%s] Direct connection forced", id)
	}
	// get hostname and port
	if isLoopback(dst) && !direct {
		return nil, errors.New(errNoLoopback)
	}
	if len(cfg.Proxy) == 0 || direct {
		return net.Dial(proto, dst)
	}
	glog.Infof("[%s] Proxying via: %s", id, cfg.Proxy)
	proxyURL, err := url.Parse(cfg.Proxy)
	if err != nil {
		return nil, err
	}
	return tunnel.DialViaProxy(proxyURL, dst)
}

// close connection
func closeconn(id string, c net.Conn) {
	err := c.Close()
	if err != nil {
		neterr, ok := err.(*net.OpError)
		if ok && neterr.Err.Error() == "use of closed network connection" {
			return
		}
		glog.Warningf("[%s] %s", id, err)
	}
}
