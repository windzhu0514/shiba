//go:build !windows
// +build !windows

package hihttp

import (
	"net/http"
	"strings"

	"github.com/fvbock/endless"
)

func newServer(addr string, handler http.Handler) HttpServer {
	return &server{
		svr: endless.NewServer(addr, handler),
	}
}

func listenAndServe(addr string, handler http.Handler) error {
	svr := &server{
		svr: endless.NewServer(addr, handler),
	}

	return svr.ListenAndServe()
}

func listenAndServeTLS(addr, certFile, keyFile string, handler http.Handler) error {
	svr := &server{
		svr: endless.NewServer(addr, handler),
	}

	return svr.ListenAndServeTLS(certFile, keyFile)
}

type server struct {
	svr HttpServer
}

func (s *server) ListenAndServe() error {
	err := s.svr.ListenAndServe()
	if err != nil && strings.Contains(err.Error(), "use of closed network connection") {
		return nil
	}

	return err
}

func (s *server) ListenAndServeTLS(certFile, keyFile string) error {
	err := s.svr.ListenAndServeTLS(certFile, keyFile)
	if err != nil && strings.Contains(err.Error(), "use of closed network connection") {
		return nil
	}

	return err
}
