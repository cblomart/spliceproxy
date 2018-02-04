package main

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
		if b == a {
			return true
		}
	}
	return false
}
