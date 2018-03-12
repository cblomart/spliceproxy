package main

import (
	"bufio"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/golang/glog"
)

// streamcopy copies a stream from one to the other
func streamcopy(id string, dst io.Writer, src io.Reader) {
	b, err := io.Copy(dst, src)
	log.Infof("[%s] Copied %d bytes", id, b)
	if err != nil {
		if neterr, ok := err.(*net.OpError); ok {
			if (neterr.Op == "read" || neterr.Op == "readfrom") && neterr.Timeout() {
				return
			}
		}
		log.Warningf("[%s] %s", id, err)
	}
}

//forward connection
func forward(id string, bufferIo *bufio.ReadWriter, dst string, direct bool) {
	// forward
	log.Infof("[%s] Forwarding to %s", id, dst)
	// get a connection
	f, err := getconn(id, dst, direct)
	if err != nil {
		log.Errorf("[%s] %s", id, err)
		return
	}
	// close when finished
	defer closeconn(id, f)
	// set deadlines
	err = f.SetDeadline(time.Now().Add(time.Duration(cfg.Timeout) * time.Second))
	if err != nil {
		log.Warningf("[%s] %s", id, err)
	}
	// coordinate read writes
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		streamcopy(id, f, bufferIo)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		streamcopy(id, bufferIo, f)
		wg.Done()
	}()
	// wait for intput and output copy
	wg.Wait()
	// notify end of transfer
	log.Infof("[%s] Forwarding to %s done", id, dst)
}
