package main

import (
	"flag"
	"io/ioutil"
	"strings"

	"github.com/go-yaml/yaml"
	"github.com/golang/glog"
)

var (
	cfg   conf
	proto = "tcp"
)

// Init initialize go program
func init() {
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
	if cfg.ForceIpv4 {
		glog.Info("Forcing IPv4")
		proto = "tcp4"
	}
}

func main() {
	// serve deny messages
	if cfg.CatchAll.Serve {
		serveWeb()
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
