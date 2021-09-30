// +build windows
//go:build windows

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

func newServer(addr string, handler http.Handler)HttpServer{
	return &http.Server{Addr: addr, Handler: handler}
}

func listenAndServe(addr string, handler http.Handler) error {
	srv := graceShutDownServer(addr, handler)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}

func listenAndServeTLS(addr, certFile, keyFile string, handler http.Handler) error {
	srv := graceShutDownServer(addr, handler)
	if err := srv.ListenAndServeTLS(certFile, keyFile); err != http.ErrServerClosed {
		return err
	}

	return nil
}

func graceShutDownServer(addr string, handler http.Handler) *http.Server {
	srv := &http.Server{Addr: addr, Handler: handler}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			fmt.Println("HTTP server Shutdown: " + err.Error())
		}
		close(idleConnsClosed)
	}()

	return srv
}
