package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	resolver   *net.Resolver
	httpClient *http.Client
)

type endpoint struct {
	IP             net.IPAddr
	HTTPSAvailable bool
	HTTPAvailable  bool
	LastUsed       bool
	Connections    int
}

type site struct {
	Name      string
	Endpoints []*endpoint
}

func check() {
	for _, s := range cfg.AllowedDomains {
		s.Resolve()
		for _, ep := range s.Endpoints {
			go ep.Check(s.Name)
		}
	}
}

func (s *site) Resolve() {
	// initialize resolver if not done
	if resolver == nil {
		resolver = &net.Resolver{PreferGo: true}
	}
	// resolve ip addresses
	ips, err := resolver.LookupIPAddr(context.Background(), s.Name)
	if err != nil {
		log.Errorf("Could not resolve %s: %s", s.Name, err)
		return
	}
	// remove endpoints
	for i, ep := range s.Endpoints {
		oldep := true
		// search resolved addresses
		for _, ip := range ips {
			if ip.String() == ep.IP.String() {
				oldep = false
			}
		}
		if oldep {
			log.Infof("Removing entry for %s: %s", s.Name, ep.IP.String())
			s.Endpoints = append(s.Endpoints[:i], s.Endpoints[i+1:]...)
		}
	}
	// create enpoints
	for _, ip := range ips {
		newip := true
		// search current known enpoints
		for _, ep := range s.Endpoints {
			if ep.IP.String() == ip.String() {
				newip = false
				break
			}
		}
		// check if new enty
		if !newip {
			continue
		}
		// add the new ip entry
		log.Infof("Adding new entry for %s: %s", s.Name, ip.String())
		ep := endpoint{
			IP:             ip,
			HTTPAvailable:  false,
			HTTPSAvailable: false,
			LastUsed:       false,
			Connections:    0,
		}
		s.Endpoints = append(s.Endpoints, &ep)
	}
}

func (ep *endpoint) Check(site string) {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	// create a http request
	go func() {
		req, err := http.NewRequest("HEAD", fmt.Sprintf("http://%s/", ep.IP.String()), nil)
		if err != nil {
			log.Infof("Can't create http request for %s with address %s: %s", site, ep.IP.String(), err)
			ep.HTTPAvailable = false
		}
		req.Host = site
		_, err = httpClient.Do(req)
		if err != nil {
			log.Infof("Http check failed for %s with address %s: %s", site, ep.IP.String(), err)
			ep.HTTPAvailable = false
		} else {
			ep.HTTPAvailable = true
		}
	}()
	// create a https request
	go func() {
		req, err := http.NewRequest("HEAD", fmt.Sprintf("https://%s/", ep.IP.String()), nil)
		if err != nil {
			log.Infof("Can't create https request for %s with address %s: %s", site, ep.IP.String(), err)
			ep.HTTPSAvailable = false
		}
		req.Host = site
		_, err = httpClient.Do(req)
		if err != nil {
			log.Infof("Https check failed for %s with address %s: %s", site, ep.IP.String(), err)
			ep.HTTPSAvailable = false
		} else {
			ep.HTTPSAvailable = true
		}
	}()
}
