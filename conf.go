package main

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

type catchAllDef struct {
	Serve bool
	Key   string
	Cert  string
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
	ForceIpv4      bool
	Proxy          string
	CatchAll       catchAllDef
	AllowedDomains []site
	Check          int
}

func (c *conf) allowed(a string) bool {
	for _, b := range c.AllowedDomains {
		if strings.HasSuffix(a, b.Name) {
			return true
		}
	}
	return false
}

func (c *conf) route(id, hostname string, port string, ssl bool) (string, bool) {
	if len(hostname) > 0 && c.allowed(hostname) {
		return hostname + port, false
	}
	log.Infof("[%s] Unautorised or unknown destination '%s', redirecting to catchall", id, hostname)
	if !ssl {
		return cfg.CatchAll.HTTP, true
	}
	return cfg.CatchAll.HTTPS, true
}
