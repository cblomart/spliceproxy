package main

import (
	"flag"
	"io/ioutil"
	"time"

	"github.com/go-yaml/yaml"
	log "github.com/sirupsen/logrus"
)

var (
	cfg   *conf
	proto = "tcp"
)

// Init initialize go program
func init() {
	// declare flags
	var cfgfile string
	flag.StringVar(&cfgfile, "c", "config.yaml", "config file")
	flag.Parse()
	// read config file
	log.Info("Reading config file: ", cfgfile)
	data, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		log.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatal(err)
	}
	log.Infof("HTTP catchall: %s", cfg.CatchAll.HTTP)
	log.Infof("HTTPS catchall: %s", cfg.CatchAll.HTTPS)
	if len(cfg.Proxy) > 0 {
		log.Infof("HTTP proxy: %s", cfg.Proxy)
	}
	for _, s := range cfg.AllowedDomains {
		log.Infof("Autorised domain: %s", s.Name)
	}
	if cfg.ForceIpv4 {
		log.Info("Forcing IPv4")
		proto = "tcp4"
	}
}

func main() {
	// serve deny messages
	if cfg.CatchAll.Serve {
		serveWeb()
	}
	// check endpoints
	if cfg.Check > 0 {
		checkTicker := time.NewTicker(5 * time.Second)
		checkQuit := make(chan struct{})
		defer close(checkQuit)
		// initial check
		check()
		go func() {
			for {
				select {
				case <-checkTicker.C:
					check()
				case <-checkQuit:
					checkTicker.Stop()
					return
				}
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
