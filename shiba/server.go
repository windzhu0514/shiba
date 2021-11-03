package shiba

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"

	"github.com/windzhu0514/shiba/hihttp"
	"github.com/windzhu0514/shiba/log"
)

var defaultServer *Server

func NewServer(opts ...Option) *Server {
	defaultServer = &Server{
		router: mux.NewRouter(),
		flags:  flag.NewFlagSet(os.Args[0], flag.ExitOnError),
	}

	for _, opt := range opts {
		opt(defaultServer)
	}

	return defaultServer
}

type ServerConfig struct {
	ServiceName           string `yaml:"serviceName"`
	Production            bool   `yaml:"production"`
	Port                  string `yaml:"port"`
	CertFile              string `yaml:"certFile"`
	KeyFile               string `yaml:"keyFile"`
	DisableSignatureCheck bool   `yaml:"disableSignatureCheck"`
	TracingAgentHostPort  string `yaml:"tracingAgentHostPort"`

	// options
	configFile  string
	pprof       bool
	openCron    bool
	openMetric  bool
	middlewares []mux.MiddlewareFunc
}

type Server struct {
	Config ServerConfig `yaml:"shiba"`
	flags  *flag.FlagSet
	router *mux.Router
	cron   *cron.Cron
}

func (s *Server) Start() error {
	for _, mod := range modules {
		if err := mod.Module.Init(); err != nil {
			return fmt.Errorf("module [%s] init:%s", mod.Name, err.Error())
		}
	}

	flagPort := s.flags.String("p", "9999", "listen port")
	flagConfigFile := s.flags.String("f", "conf.yaml", "redisConfig file path")
	if err := s.flags.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("flag parse:" + err.Error())
	}

	// 命令行覆盖option
	if *flagConfigFile != "" {
		s.Config.configFile = *flagConfigFile
	}

	if err := loadConfig(s.Config.configFile); err != nil {
		return fmt.Errorf("load redisConfig file:" + err.Error())
	}

	// 配置覆盖option
	if err := rawFileCfg.Decode(s); err != nil {
		return fmt.Errorf("module [server] decode redisConfig:%s", err.Error())
	}

	logNode := fileCfg["log"]
	var cfg log.Config
	if err := logNode.Decode(&cfg); err != nil {
		return fmt.Errorf("module [log] decode redisConfig:" + err.Error())
	}

	defaultLogger = log.New("shiba", nil, cfg)

	for _, mod := range modules {
		_, exist := fileCfg[mod.Name]
		if !exist {
			continue
		}

		if err := rawFileCfg.Decode(mod.Module); err != nil {
			errMsg := fmt.Sprintf("module [%s] decode redisConfig:%s", mod.Name, err.Error())
			defaultLogger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		if err := mod.Module.Start(); err != nil {
			errMsg := fmt.Sprintf("module [%s] start:%s\n", mod.Name, err.Error())
			defaultLogger.Errorf(errMsg)
			return errors.New(errMsg)
		}

		defaultLogger.Infof("module [%s] priority:%d start success", mod.Name, mod.Priority)
	}

	if s.Config.openCron {
		cronLogger := cronLogger{logger: defaultLogger}
		s.cron = cron.New(
			cron.WithLogger(cronLogger),
			cron.WithChain(cron.SkipIfStillRunning(cronLogger)),
			cron.WithParser(cron.NewParser(
				cron.SecondOptional|cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor,
			)))
		s.cron.Start()
	}

	if s.Config.openMetric {
		s.router.Handle("/metrics", promhttp.Handler())
	}

	if s.Config.pprof {
		s.router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
		s.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		s.router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	//  curl -X PUT localhost:8080/log_level?level=debug -H "Content-Type: application/x-www-form-urlencoded"
	//  curl -X PUT localhost:8080/log_level -H "Content-Type: application/json" -d '{"level":"debug"}'
	s.router.HandleFunc("/log_level", defaultLogger.ServeHTTP)

	port := s.Config.Port
	if *flagPort != "" {
		port = *flagPort
	}

	svr := hihttp.NewServer(":"+port, s.router)
	svr.RegisterOnShutdown(func() {
		s.stop()
	})

	s.router.Use(s.Config.middlewares...)

	if len(s.Config.TracingAgentHostPort) > 0 {
		closer, err := newJaegerTracer(s.Config.ServiceName, s.Config.TracingAgentHostPort)
		if err != nil {
			errMsg := fmt.Sprintf("server new Tracer:%s", err.Error())
			defaultLogger.Error(errMsg)
			return errors.New(errMsg)
		}

		svr.RegisterOnShutdown(func() {
			if err := closer.Close(); err != nil {
				defaultLogger.Errorf("module [shiba] Tracer Close:%s\n", err.Error())
			}
		})
		s.router.Use(MiddlewareTracing)
	}

	if s.Config.CertFile == "" && s.Config.KeyFile == "" {
		defaultLogger.Info("start ListenAndServe")
		if err := svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			defaultLogger.Error("ListenAndServe:" + err.Error())
		}
	} else {
		defaultLogger.Info("start ListenAndServeTLS")
		if err := svr.ListenAndServeTLS(s.Config.CertFile,
			s.Config.KeyFile); err != nil && err != http.ErrServerClosed {
			defaultLogger.Error("ListenAndServeTLS:" + err.Error())
		}
	}

	err := defaultLogger.Close()
	if err != nil {
		errMsg := fmt.Sprintf("module log stop failed:" + err.Error())
		defaultLogger.Error(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func (s *Server) stop() {
	for i := len(modules) - 1; i >= 0; i-- {
		mod := modules[i]

		_, exist := fileCfg[mod.Name]
		if !exist {
			continue
		}

		if err := mod.Module.Stop(); err != nil {
			defaultLogger.Infof("module [%s] stop:%s", mod.Name, err.Error())
			// not return
		} else {
			defaultLogger.Infof("module [%s] priority:%d stop success", mod.Name, mod.Priority)
		}
	}
}

func (s *Server) RegisterModule(priority int, mod Module) {
	registerModule(priority, mod)
}

// func GetModule(name string) Module {
// 	return getModule(name)
// }

func Logger(name string) log.Logger {
	return defaultLogger.Clone(name)
}

func DBMaster(name string) (*sqlx.DB, error) {
	return db.Master(name)
}

func DBSlave(name string) (*sqlx.DB, error) {
	return db.Slave(name)
}

func Redis(name string) (RedisCmdable, error) {
	return redisx.Get(name)
}

func Router() *mux.Router {
	return defaultServer.router
}

func FlagSet() *flag.FlagSet {
	return defaultServer.flags
}

func Cron() *cron.Cron {
	if defaultServer.cron == nil {
		panic("cron not start")
	}

	return defaultServer.cron
}

func Config() ServerConfig {
	return defaultServer.Config
}
