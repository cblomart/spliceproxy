package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
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
	Changed        bool
	Connections    int
	Protocol       string
}

type site struct {
	Name      string
	Endpoints []endpoint
	Protocol  string
	Index     int
}

func check() {
	for i := range cfg.AllowedDomains {
		s := &cfg.AllowedDomains[i]
		s.Info()
		s.Resolve()
		for j := range s.Endpoints {
			s.Endpoints[j].Check(s.Name)
		}
	}
}

func (s *site) Stats() (int, int, int, int) {
	total := 0
	httplive := 0
	httpslive := 0
	connections := 0
	for _, ep := range s.Endpoints {
		total++
		if ep.HTTPAvailable {
			httplive++
		}
		if ep.HTTPSAvailable {
			httpslive++
		}
		connections += ep.Connections
	}
	return httplive, httpslive, connections, total
}

func (s *site) Available(protocol string) bool {
	httplive, httpslive, _, _ := s.Stats()
	if protocol == "both" {
		return httplive > 0 || httpslive > 0
	}
	if protocol == "http" {
		return httplive > 0
	}
	if protocol == "https" {
		return httpslive > 0
	}
	return false
}

func (s *site) Changed() bool {
	if len(s.Endpoints) == 0 {
		return false
	}
	for i := range s.Endpoints {
		if s.Endpoints[i].Changed {
			return true
		}
	}
	return false
}

func (s *site) Info() {
	httplive, httpslive, connections, total := s.Stats()
	log.Infof("Stats for %s: http=%d https=%d total=%d connection=%d", s.Name, httplive, httpslive, total, connections)
}

func (s *site) Resolve() {
	// initialize resolver if not done
	if resolver == nil {
		resolver = &net.Resolver{}
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
			Changed:        false,
			Connections:    0,
			Protocol:       s.Protocol,
		}
		s.Endpoints = append(s.Endpoints, ep)
	}
}

func (s *site) GetEndpoint(protocol string) *endpoint {
	if !s.Available(protocol) {
		return nil
	}
	i := s.Index
	for {
		ep := s.Endpoints[i]
		i++
		if i >= len(s.Endpoints) {
			i = 0
		}
		if ep.GetAvailability(protocol) {
			return &ep
		}
	}
}

func (ep *endpoint) Check(site string) {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec
			},
		}
	}
	// create a http request
	if ep.Protocol == "both" {
		go ep.CheckProto("http", site)
		go ep.CheckProto("https", site)
	}
	if ep.Protocol == "http" {
		go ep.CheckProto("http", site)
	}
	if ep.Protocol == "https" {
		go ep.CheckProto("https", site)
	}
}

func (ep *endpoint) SetAvailability(protocol string, availability bool) {
	if protocol == "http" {
		ep.HTTPAvailable = availability
		return
	}
	if protocol == "https" {
		ep.HTTPSAvailable = availability
	}
}

func (ep *endpoint) GetAvailability(protocol string) bool {
	if protocol == "http" {
		return ep.HTTPAvailable
	}
	if protocol == "https" {
		return ep.HTTPSAvailable
	}
	return false
}

func (ep *endpoint) CheckProto(protocol, site string) {
	waslive := ep.GetAvailability(protocol)
	address := ep.IP.String()
	if strings.Count(address, ":") >= 2 {
		address = "[" + address + "]"
	}
	req, err := http.NewRequest("HEAD", fmt.Sprintf("%s://%s/", protocol, address), nil)
	if err != nil {
		log.Errorf("Can't create %s request for %s with address %s: %s", protocol, site, ep.IP.String(), err)
		ep.SetAvailability(protocol, false)
	}
	req.Host = site
	_, err = httpClient.Do(req)
	if err != nil {
		log.Warningf("%s check failed for %s with address %s: %s", protocol, site, ep.IP.String(), err)
		ep.SetAvailability(protocol, false)
	} else {
		ep.SetAvailability(protocol, true)
	}
	if waslive != ep.GetAvailability(protocol) {
		ep.Changed = true
	} else {
		ep.Changed = false
	}
}
