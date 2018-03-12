package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"strings"

	log "github.com/golang/glog"
)

// HTTPSValidateHello validates hello tls message
func HTTPSValidateHello(br *bufio.Reader) ([]byte, error) {
	buff, err := br.Peek(sslHeaderLen)
	if err != nil {
		return nil, err
	}
	// check ssl parameters
	if len(buff) == 0 {
		return nil, errors.New(errNoContent)
	}
	if buff[0] != sslTypeHandshake {
		return nil, errors.New(errNotTLS)
	}
	recLen := int(buff[3])<<8 | int(buff[4]) // ignoring version in hdr[1:3]
	return br.Peek(sslHeaderLen + recLen)
}

// HTTPSDestination detect HTTPS destination via SNI
// from https://github.com/google/tcpproxy/blob/de1c7de/sni.go#L156
func HTTPSDestination(id string, br *bufio.ReadWriter, port string) (hostname string, direct bool, err error) {
	// peek into the stream
	buff, err := HTTPSValidateHello(br.Reader)
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
	log.Infof("[%s] Peeked SSL destination: %s", id, hostname)
	dest, direct := cfg.route(id, hostname, port, true)
	return dest, direct, nil
}

// return the next line as a string
func peekline(br *bufio.Reader, start *int, index *int) (string, error) {
	if *index >= cfg.Buffer {
		return "", errors.New(errExceedBuffer)
	}
	buff, err := br.Peek(*index)
	if err != nil {
		return "", err
	}
	if buff[*index-1] == '\r' || buff[*index-1] == '\n' {
		tmp := buff[*start : *index-1]
		skip := 1
		if buff[*index-1] == '\n' {
			skip = 0
		}
		*start = *index + skip
		*index = *start + 1
		return string(tmp), nil
	}
	*index++
	return peekline(br, start, index)
}

// peek host for each lines read
func peekhost(br *bufio.Reader, start *int, index *int) (string, error) {
	line, err := peekline(br, start, index)
	for {
		if len(line) == 0 {
			return "", errors.New(errNoHTTPHost)
		}
		if err != nil {
			return "", errors.New(errNoContent)
		}
		if strings.HasPrefix(line, hostHeader) {
			hostname := strings.TrimPrefix(line, hostHeader)
			return hostname, nil
		}
		line, err = peekline(br, start, index)
	}
}

// HTTPDestination detect HTTP destination in headers
func HTTPDestination(id string, br *bufio.ReadWriter, port string) (string, bool, error) {
	// peek into the stream
	start := 0
	index := 1
	hostname, err := peekhost(br.Reader, &start, &index)
	if err != nil {
		return "", false, err
	}
	log.Infof("[%s] Peeked HTTP destination: %s", id, hostname)
	dest, direct := cfg.route(id, hostname, port, true)
	return dest, direct, nil
}
