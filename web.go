package main

import (
	"net/http"

	"github.com/golang/glog"
)

func serveWeb() {
	http.HandleFunc("/", DenyServer)
	go func() {
		err := http.ListenAndServeTLS(cfg.CatchAll.HTTPS, cfg.CatchAll.Cert, cfg.CatchAll.Key, nil)
		if err != nil {
			glog.Fatal(err)
		}
	}()
	go func() {
		err := http.ListenAndServe(cfg.CatchAll.HTTP, nil)
		if err != nil {
			glog.Fatal(err)
		}
	}()
}
