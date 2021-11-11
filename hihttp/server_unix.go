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
		srv: endless.NewServer(addr, handler),
	}
}

func listenAndServe(addr string, handler http.Handler) error {
	srv := &server{srv: endless.NewServer(addr, handler)}
	return svr.ListenAndServe()
}

func listenAndServeTLS(addr, certFile, keyFile string, handler http.Handler) error {
	srv := &server{srv: endless.NewServer(addr, handler)}
	return srv.ListenAndServeTLS(certFile, keyFile)
}

type server struct {
	srv HttpServer
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

func (s *server) RegisterOnShutdown(f func()) {
	s.srv.RegisterOnShutdown(f)
}
