package main

const (
	errNotImp     = "Not Implemented."
	errNoHTTPHost = "No Host header found in buffered HTTP header (%d bytes)"
	errNotTLS     = "Communication is not TLS"
	errNoContent  = "Nothing received"

	hostHeader = "Host: "

	sslHeaderLen     = 5
	sslTypeHandshake = 0x16
)
