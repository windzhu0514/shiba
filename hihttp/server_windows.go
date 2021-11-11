//go:build windows
// +build windows

package hihttp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type server struct {
	svr *http.Server
}

func newServer(addr string, handler http.Handler) HttpServer {
	return &server{&http.Server{Addr: addr, Handler: handler}}
}

func listenAndServe(addr string, handler http.Handler) error {
	srv := &server{svr: &http.Server{Addr: addr, Handler: handler}}
	return srv.ListenAndServe()
}

func listenAndServeTLS(addr, certFile, keyFile string, handler http.Handler) error {
	srv := &server{svr: &http.Server{Addr: addr, Handler: handler}}
	return srv.ListenAndServeTLS(certFile, keyFile)
}

func (s *server) ListenAndServe() error {
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.svr.Shutdown(ctx); err != nil {
			fmt.Println("HTTP server Shutdown: " + err.Error())
		}
		close(idleConnsClosed)
	}()

	err := s.svr.ListenAndServe()
	<-idleConnsClosed
	return err
}

func (s *server) ListenAndServeTLS(certFile, keyFile string) error {
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.svr.Shutdown(ctx); err != nil {
			fmt.Println("HTTP server Shutdown: " + err.Error())
		}
		close(idleConnsClosed)
	}()

	err := s.svr.ListenAndServeTLS(certFile, keyFile)
	<-idleConnsClosed
	return err
}
