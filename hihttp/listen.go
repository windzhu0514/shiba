// +build !windows

package hihttp

import (
	"net/http"

	"github.com/fvbock/endless"
)

func newServer(addr string, handler http.Handler)HttpServer{
	return endless.NewServer(addr, handler)
}

func listenAndServe(addr string, handler http.Handler,f func()) error {
	server := endless.NewServer(addr, handler)
	if err := server.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func listenAndServeTLS(addr string, certFile string, keyFile string, handler http.Handler,f func()) error {
	server := endless.NewServer(addr, handler)
	server.RegisterOnShutdown()
	if err := server.ListenAndServeTLS( certFile, keyFile); err != nil {
		return err
	}

	return nil
}
