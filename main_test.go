package main

import (
	"bufio"
	"net/http"
	"testing"
)

func TestDenyServer(t *testing.T) {
	type args struct {
		w   http.ResponseWriter
		req *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DenyServer(tt.args.w, tt.args.req)
		})
	}
}

func TestHTTPSDestination(t *testing.T) {
	type args struct {
		id   string
		br   *bufio.ReadWriter
		port string
	}
	tests := []struct {
		name         string
		args         args
		wantHostname string
		wantDirect   bool
		wantErr      bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHostname, gotDirect, err := HTTPSDestination(tt.args.id, tt.args.br, tt.args.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPSDestination() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotHostname != tt.wantHostname {
				t.Errorf("HTTPSDestination() gotHostname = %v, want %v", gotHostname, tt.wantHostname)
			}
			if gotDirect != tt.wantDirect {
				t.Errorf("HTTPSDestination() gotDirect = %v, want %v", gotDirect, tt.wantDirect)
			}
		})
	}
}

func TestHTTPDestination(t *testing.T) {
	type args struct {
		id   string
		br   *bufio.ReadWriter
		port string
	}
	tests := []struct {
		name         string
		args         args
		wantHostname string
		wantDirect   bool
		wantErr      bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHostname, gotDirect, err := HTTPDestination(tt.args.id, tt.args.br, tt.args.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPDestination() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotHostname != tt.wantHostname {
				t.Errorf("HTTPDestination() gotHostname = %v, want %v", gotHostname, tt.wantHostname)
			}
			if gotDirect != tt.wantDirect {
				t.Errorf("HTTPDestination() gotDirect = %v, want %v", gotDirect, tt.wantDirect)
			}
		})
	}
}

func Test_listen(t *testing.T) {
	type args struct {
		addr       string
		detectdest func(string, *bufio.ReadWriter, string) (string, bool, error)
	}
	tests := []struct {
		name string
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listen(tt.args.addr, tt.args.detectdest)
		})
	}
}

func Test_isLoopback(t *testing.T) {
	type args struct {
		addr string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLoopback(tt.args.addr); got != tt.want {
				t.Errorf("isLoopback() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_forward(t *testing.T) {
	type args struct {
		id       string
		bufferIo *bufio.ReadWriter
		dst      string
		direct   bool
	}
	tests := []struct {
		name string
		args args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forward(tt.args.id, tt.args.bufferIo, tt.args.dst, tt.args.direct)
		})
	}
}

func Test_main(t *testing.T) {
	tests := []struct {
		name string
	}{
	// TODO: Add test cases.
	}
	for range tests {
		t.Run(tt.name, func(t *testing.T) {
			main()
		})
	}
}
