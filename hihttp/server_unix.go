//go:build !windows
// +build !windows

package hihttp

import (
	"net/http"

	"github.com/fvbock/endless"
)

func newServer(addr string, handler http.Handler) HttpServer {
	return endless.NewServer(addr, handler)
}

func listenAndServe(addr string, handler http.Handler) error {
	server := endless.NewServer(addr, handler)
	return server.ListenAndServe()
}

func listenAndServeTLS(addr string, certFile string, keyFile string, handler http.Handler) error {
	server := endless.NewServer(addr, handler)
	return server.ListenAndServeTLS(certFile, keyFile)
}
