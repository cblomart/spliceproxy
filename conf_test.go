package main

import "testing"

func Test_conf_allowed(t *testing.T) {
	type fields struct {
		Timeout        int
		Buffer         int
		Listen         endpointsDef
		ForceIpv4      bool
		Proxy          string
		CatchAll       catchAllDef
		AllowedDomains []string
	}
	type args struct {
		a string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &conf{
				Timeout:        tt.fields.Timeout,
				Buffer:         tt.fields.Buffer,
				Listen:         tt.fields.Listen,
				ForceIpv4:      tt.fields.ForceIpv4,
				Proxy:          tt.fields.Proxy,
				CatchAll:       tt.fields.CatchAll,
				AllowedDomains: tt.fields.AllowedDomains,
			}
			if got := c.allowed(tt.args.a); got != tt.want {
				t.Errorf("conf.allowed() = %v, want %v", got, tt.want)
			}
		})
	}
}
