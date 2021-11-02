package shiba

import "github.com/gorilla/mux"

type Option func(s *Server)

func WithConfig(filename string) Option {
	return func(s *Server) {
		s.Config.configFile = filename
	}
}

func WithHttps(certFile, keyFile string) Option {
	return func(s *Server) {
		s.Config.CertFile = certFile
		s.Config.KeyFile = keyFile

	}
}

func WithPprof() Option {
	return func(s *Server) {
		s.Config.pprof = true
	}
}

func WithCron() Option {
	return func(s *Server) {
		s.Config.openCron = true
	}
}

func WithMetric() Option {
	return func(s *Server) {
		s.Config.openMetric = true
	}
}

func WithMiddleware(middlewares ...MiddlewareFunc) Option {
	return func(s *Server) {
		for _, middleware := range middlewares {
			s.Config.middlewares = append(s.Config.middlewares, mux.MiddlewareFunc(middleware))
		}
	}
}

func WithTracingAgentHostPort(addr string) Option {
	return func(s *Server) {
		s.Config.TracingAgentHostPort = addr
	}
}
