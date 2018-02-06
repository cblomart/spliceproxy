package main

import (
	"io"
	"net"
	"testing"
)

func Test_sniSniffConn_Read(t *testing.T) {
	type fields struct {
		r    io.Reader
		Conn net.Conn
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := sniSniffConn{
				r:    tt.fields.r,
				Conn: tt.fields.Conn,
			}
			got, err := c.Read(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("sniSniffConn.Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("sniSniffConn.Read() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sniSniffConn_Write(t *testing.T) {
	type fields struct {
		r    io.Reader
		Conn net.Conn
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := sniSniffConn{
				r:    tt.fields.r,
				Conn: tt.fields.Conn,
			}
			got, err := s.Write(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("sniSniffConn.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("sniSniffConn.Write() = %v, want %v", got, tt.want)
			}
		})
	}
}
