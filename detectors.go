package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/golang/glog"
)

// HTTPSDestination detect HTTPS destination via SNI
// from https://github.com/google/tcpproxy/blob/de1c7de/sni.go#L156
func HTTPSDestination(id string, br *bufio.ReadWriter, port string) (hostname string, direct bool, err error) {
	// peek into the stream
	buff, err := br.Peek(sslHeaderLen)
	if err != nil {
		glog.Warningf("[%s] %s", id, err)
	}
	if len(buff) == 0 {
		return "", false, errors.New(errNoContent)
	}
	if buff[0] != sslTypeHandshake {
		return "", false, errors.New(errNotTLS)
	}
	recLen := int(buff[3])<<8 | int(buff[4]) // ignoring version in hdr[1:3]
	buff, err = br.Peek(sslHeaderLen + recLen)
	if err != nil {
		return "", false, err
	}
	err = tls.Server(sniSniffConn{r: bytes.NewReader(buff)}, &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			hostname = hello.ServerName
			return nil, nil
		},
	}).Handshake()
	if err != nil {
		// throw other errors than no certificates configured
		// this errors is defined as errors.New(...) in go sources
		if strings.Compare(err.Error(), "tls: no certificates configured") != 0 {
			return "", false, err
		}
	}
	glog.Infof("[%s] Peeked SSL destination: %s", id, hostname)
	if cfg.allowed(hostname) {
		return hostname + port, false, nil
	}
	glog.Warningf("[%s] Destination %s not autorized redirecting to catchall: %s", id, hostname, cfg.CatchAll.HTTPS)
	return cfg.CatchAll.HTTPS, true, nil
}

// HTTPDestination detect HTTP destination in headers
func HTTPDestination(id string, br *bufio.ReadWriter, port string) (hostname string, direct bool, err error) {
	// peek into the stream
	index := 1
	lastindex := index
	var char byte
	for index <= cfg.Buffer {
		buff, err := br.Peek(index)
		// for http browser will open multiple connections
		// ignore any peeking errors that are read related
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if strings.Compare(neterr.Op, "write") == 0 {
					glog.Warningf("[%s] %s", id, err)
				}
			} else {
				glog.Warningf("[%s] %s", id, err)
			}
		}
		if len(buff) < index {
			return "", false, errors.New(errNoContent)
		}
		char = buff[index-1]
		if char == '\r' {
			//begin of new line
			line := string(buff[lastindex : index-1])
			if strings.Compare("", line) == 0 {
				return "", false, fmt.Errorf(errNoHTTPHost, cfg.Buffer)
			}
			if strings.HasPrefix(line, hostHeader) {
				hostname = strings.TrimPrefix(line, hostHeader)
				glog.Infof("[%s] Peeked HTTP destination: %s", id, hostname)
				break
			}
			lastindex = index
		}
		if char == '\n' {
			lastindex = index
		}
		index++
	}
	if len(hostname) == 0 {
		return "", false, fmt.Errorf(errNoHTTPHost, cfg.Buffer)
	}
	if cfg.allowed(hostname) {
		return hostname + port, false, nil
	}
	glog.Warningf("[%s] Destination %s not autorized redirecting to catchall: %s", id, hostname, cfg.CatchAll.HTTP)
	return cfg.CatchAll.HTTP, true, nil
}
