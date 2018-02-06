package main

import (
	"bufio"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/golang/glog"
	"github.com/rackerlabs/go-connect-tunnel"
	uuid "github.com/satori/go.uuid"
)

var (
	cfg   conf
	proto string
)

// DenyServer server deny messages
func DenyServer(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte("Access denied.\n"))
	if err != nil {
		glog.Warning(err)
	}
}

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

// streamcopy copies a stream from one to the other
func streamcopy(id string, dst io.Writer, src io.Reader) {
	b, err := io.Copy(dst, src)
	glog.Infof("[%s] Copied %d bytes", id, b)
	if err != nil {
		if neterr, ok := err.(*net.OpError); ok {
			if (neterr.Op == "read" || neterr.Op == "readfrom") && neterr.Timeout() {
				return
			}
		}
		glog.Warningf("[%s] %s", id, err)
	}
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

//forward connection
func forward(id string, bufferIo *bufio.ReadWriter, dst string, direct bool) {
	// forward
	glog.Infof("[%s] Forwarding to %s", id, dst)
	// get a connection
	f, err := getconn(id, dst, direct)
	if err != nil {
		glog.Errorf("[%s] %s", id, err)
		return
	}
	// close when finished
	defer closeconn(id, f)
	// set deadlines
	err = f.SetDeadline(time.Now().Add(time.Duration(cfg.Timeout) * time.Second))
	if err != nil {
		glog.Warningf("[%s] %s", id, err)
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
	glog.Infof("[%s] Forwarding to %s done", id, dst)
}

func main() {
	// declare flags
	var cfgfile string
	flag.StringVar(&cfgfile, "c", "config.yaml", "config file")
	err := flag.Set("logtostderr", "true")
	if err != nil {
		glog.Fatal(err)
	}
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
	glog.Infof("HTTP catchall: %s", cfg.CatchAll.HTTP)
	glog.Infof("HTTPS catchall: %s", cfg.CatchAll.HTTPS)
	if len(cfg.Proxy) > 0 {
		glog.Infof("HTTP proxy: %s", cfg.Proxy)
	}
	glog.Infof("Autorised domains: %s", strings.Join(cfg.AllowedDomains, ", "))
	proto = "tcp"
	if cfg.ForceIpv4 {
		glog.Info("Forcing IPv4")
		proto = "tcp4"
	}

	// serve deny messages
	if cfg.CatchAll.Serve {
		http.HandleFunc("/", DenyServer)
		go func() {
			err := http.ListenAndServeTLS(cfg.CatchAll.HTTPS, cfg.CatchAll.Cert, cfg.CatchAll.Key, nil)
			if err != nil {
				glog.Fatal(err)
			}
		}()
		go func() {
			err := http.ListenAndServe(cfg.CatchAll.HTTP, nil)
			if err != nil {
				glog.Fatal(err)
			}
		}()
	}

	// listen for requests
	for _, d := range cfg.Listen.HTTP {
		go listen(d, HTTPDestination)
	}
	for _, d := range cfg.Listen.HTTPS {
		go listen(d, HTTPSDestination)
	}

	// wait
	select {}
}
