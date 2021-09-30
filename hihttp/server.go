package hihttp

import (
	"net/http"
)

// TODO:LimitListener golang.org/x/net/netutil
// endless不支持自定义listen

type HttpServer interface {
	ListenAndServe() error
	ListenAndServeTLS(certFile, keyFile string) error
	RegisterOnShutdown(f func())
}

func NewServer(addr string, handler http.Handler)HttpServer{
	return newServer(addr,handler)
}

func ListenAndServe(addr string, handler http.Handler) error {
	if err := listenAndServe(addr, handler); err != nil {
		return err
	}

	return nil
}

func ListenAndServeTLS(addr, certFile, keyFile string, handler http.Handler) error {
	if err := listenAndServeTLS(addr, certFile, keyFile, handler); err != nil {
		return err
	}

	return nil
}
