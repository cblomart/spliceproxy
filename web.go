package main

import (
	"net/http"

	"github.com/golang/glog"
)

// denyServer server deny messages
func denyServer(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte("Access denied.\n"))
	if err != nil {
		glog.Warning(err)
	}
}

func serveWeb() {
	http.HandleFunc("/", denyServer)
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
