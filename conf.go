package main

import (
	"strings"

	"github.com/golang/glog"
)

type catchAllDef struct {
	HTTP  string
	HTTPS string
}

type endpointsDef struct {
	HTTP  []string
	HTTPS []string
}

type conf struct {
	Timeout        int
	Buffer         int
	Listen         endpointsDef
	Proxy          string
	CatchAll       catchAllDef
	AllowedDomains []string
}

func (c *conf) allowed(a string) bool {
	for _, b := range c.AllowedDomains {
		if strings.HasSuffix(a, b) {
			glog.Info("Checking if %s has suffix %s", a, b)
			return true
		}
	}
	return false
}
