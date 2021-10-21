package shiba

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"

	"github.com/windzhu0514/shiba/hihttp"
	"github.com/windzhu0514/shiba/log"
)

func Start(opts ...Option) {
	defaultServer.Start(opts...)
}

type serverConfig struct {
	Production            bool   `yaml:"production"`
	Port                  string `yaml:"port"`
	CertFile              string `yaml:"certFile"`
	KeyFile               string `yaml:"keyFile"`
	DisableSignatureCheck bool   `yaml:"disableSignatureCheck"`

	// options
	configFile  string
	pprof       bool
	openCron    bool
	openMetric  bool
	middlewares []mux.MiddlewareFunc
}

type server struct {
	Config serverConfig `yaml:"shiba"`

	flags  *flag.FlagSet
	router *mux.Router
	cron   *cron.Cron
}

var defaultServer = &server{
	router: mux.NewRouter(),
	flags:  flag.NewFlagSet(os.Args[0], flag.ExitOnError),
}

func (s *server) Start(opts ...Option) {
	for _, opt := range opts {
		opt()
	}

	for _, mod := range modules {
		if err := mod.Module.Init(); err != nil {
			defaultLogger.Errorf("module [%s] init:%s", mod.Name, err.Error())
			return
		}
	}

	flagPort := s.flags.String("p", "9999", "listen port")
	flagConfigFile := s.flags.String("f", "shiba.yaml", "config file path")
	if err := s.flags.Parse(os.Args[1:]); err != nil {
		fmt.Println("flag parse:" + err.Error())
		return
	}

	// 命令行覆盖option
	if *flagConfigFile != "" {
		s.Config.configFile = *flagConfigFile
	}

	if err := loadConfig(s.Config.configFile); err != nil {
		fmt.Println("load config file:" + err.Error())
		return
	}

	// 配置覆盖option
	if err := rawFileCfg.Decode(defaultServer); err != nil {
		defaultLogger.Errorf("module [server] decode config:%s", err.Error())
		return
	}

	logNode, _ := fileCfg["log"]
	var cfg log.Config
	if err := logNode.Decode(&cfg); err != nil {
		fmt.Println("module [log] decode config:" + err.Error())
		return
	}

	defaultLogger = log.New("shiba", nil, cfg)

	for _, mod := range modules {
		_, exist := fileCfg[mod.Name]
		if !exist {
			continue
		}

		if err := rawFileCfg.Decode(mod.Module); err != nil {
			defaultLogger.Errorf("module [%s] decode config:%s", mod.Name, err.Error())
			return
		}

		if err := mod.Module.Start(); err != nil {
			defaultLogger.Errorf("module [%s] start:%s\n", mod.Name, err.Error())
			return
		}

		defaultLogger.Infof("module [%s] priority:%d start success", mod.Name, mod.Priority)
	}

	if s.Config.openCron {
		cronLogger := cronLogger{logger: defaultLogger}
		defaultServer.cron = cron.New(
			cron.WithLogger(cronLogger),
			cron.WithChain(cron.SkipIfStillRunning(cronLogger)),
			cron.WithParser(cron.NewParser(
				cron.SecondOptional|cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor,
			)))
		defaultServer.cron.Start()
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
	defaultServer.router.HandleFunc("/log_level", defaultLogger.ServeHTTP)

	port := defaultServer.Config.Port
	if *flagPort != "" {
		port = *flagPort
	}

	svr := hihttp.NewServer(":"+port, defaultServer.router)
	svr.RegisterOnShutdown(func() {
		fmt.Println("Stop")
		s.Stop()
	})

	if defaultServer.Config.CertFile == "" && defaultServer.Config.KeyFile == "" {
		defaultLogger.Info("ListenAndServe")
		if err := svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			defaultLogger.Error("ListenAndServe:" + err.Error())
		}
	} else {
		defaultLogger.Info("ListenAndServeTLS")
		if err := svr.ListenAndServeTLS(defaultServer.Config.CertFile,
			defaultServer.Config.KeyFile); err != nil && err != http.ErrServerClosed {
			defaultLogger.Error("ListenAndServeTLS:" + err.Error())
		}
	}

	err := defaultLogger.Close()
	if err != nil {
		fmt.Println("module log stop failed:" + err.Error())
		return
	}
}

func (s *server) Stop() {
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

type Option func()

func WithConfig(filename string) Option {
	return func() {
		defaultServer.Config.configFile = filename
	}
}

func WithHttps(certFile, keyFile string) Option {
	return func() {
		defaultServer.Config.CertFile = certFile
		defaultServer.Config.KeyFile = keyFile

	}
}

func WithPprof() Option {
	return func() {
		defaultServer.Config.pprof = true
	}
}

func WithCron() Option {
	return func() {
		defaultServer.Config.openCron = true
	}
}

func WithMetric() Option {
	return func() {
		defaultServer.Config.openMetric = true
	}
}

func WithMiddleware(middlewares ...mux.MiddlewareFunc) Option {
	return func() {
		defaultServer.router.Use(middlewares...)
	}
}

func RegisterModule(priority int, mod Module) {
	registerModule(priority, mod)
}

func GetModule(name string) Module {
	return getModule(name)
}

func Logger(name string) log.Logger {
	return defaultLogger.Clone(name)
}

func DBMaster(name string) (*sqlx.DB, error) {
	return db.Master(name)
}

func DBSlave(name string) (*sqlx.DB, error) {
	return db.Slave(name)
}

func Redis(name string) (redis.Cmdable, error) {
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

func Config() serverConfig {
	return defaultServer.Config
}
