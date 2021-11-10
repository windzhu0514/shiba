//go:build !windows
// +build !windows

package hihttp

import (
	"net/http"
	"strings"

	"github.com/fvbock/endless"
)

func newServer(addr string, handler http.Handler) HttpServer {
	return endless.NewServer(addr, handler)
}

func listenAndServe(addr string, handler http.Handler) error {
	server := endless.NewServer(addr, handler)
	err := server.ListenAndServe()
	if err != nil && strings.Contains(err.Error(), "use of closed network connection") {
		return nil
	}

	return err
}

func listenAndServeTLS(addr string, certFile string, keyFile string, handler http.Handler) error {
	server := endless.NewServer(addr, handler)
	err := server.ListenAndServeTLS(certFile, keyFile)
	if err != nil && strings.Contains(err.Error(), "use of closed network connection") {
		return nil
	}
	return err
}
