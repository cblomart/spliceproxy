# SpliceProxy

Proxy to splice http and https traffic.

All http and https traffic sent to the proxy will be treated as a transparent proxy for the indicated hosts.

All traffic not intended to known host list will be transfered to a catch all host.

This is to allow forwarding of request trough corporate network altought the host is not.
The host needs to resolve targeted hostname too the proxy (i.e. DirectAccess nrpt).