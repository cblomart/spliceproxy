package main

type catchAllDef struct {
	Http	string
	Https   string
}


type endpointsDef struct {
	Http	[]string
	Https   []string
}

type conf struct {
	Timeout        int
	Buffer		   int
	Listen		   endpointsDef
	CatchAll       catchAllDef
	AllowedDomains []string
}
