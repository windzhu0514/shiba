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
	"go.uber.org/zap"

	"github.com/windzhu0514/shiba/hihttp"
	"github.com/windzhu0514/shiba/log"
)

func Start(opts ...Option) {
	defaultServer.Start(opts...)
}

type server struct {
	flags  *flag.FlagSet
	router *mux.Router
	cron   *cron.Cron

	// options
	configFile string
	CertFile   string
	KeyFile    string
	pprof      bool
	openCron   bool
	openMetric bool
}

var defaultServer = server{
	router: mux.NewRouter(),
	flags:  flag.NewFlagSet(os.Args[0], flag.ExitOnError),
}

func (s *server) Start(opts ...Option) {
	for _, opt := range opts {
		opt()
	}

	flagPort := s.flags.String("p", "", "listen port. default:9999")
	flagConfigFile := s.flags.String("f", "conf.yaml", "config file name")
	if err := s.flags.Parse(os.Args[1:]); err != nil {
		fmt.Println("flag parse:" + err.Error())
		return
	}

	if *flagConfigFile != "" {
		s.configFile = *flagConfigFile
	}

	if err := loadConfig(s.configFile); err != nil {
		fmt.Println("load config file:" + err.Error())
		return
	}

	logNode, exist := fileCfg["log"]
	if exist {
		var cfg log.Config
		if err := logNode.Decode(&cfg); err != nil {
			fmt.Println("module [log] decode config:" + err.Error())
			return
		}

		defaultLogger=log.New("shiba", nil, cfg)
	}else{
		defaultLogger=log.New("", nil, log.Config{})
	}

	for _, mod := range modules {
		_, exist := fileCfg[mod.Name]
		if exist {
			if err := rawFileCfg.Decode(mod.Module); err != nil {
				defaultLogger.Errorf("module [%s] decode config:%s", mod.Name, err.Error())
				return
			}
		}

		if err := mod.Module.Init(); err != nil {
			defaultLogger.Errorf("module [%s] init:%s", mod.Name, err.Error())
			return
		}
	}

	if s.openCron {
		cronLogger := cronLogger{logger: defaultLogger}
		defaultServer.cron = cron.New(
			cron.WithLogger(cronLogger),
			cron.WithChain(cron.SkipIfStillRunning(cronLogger)),
			cron.WithParser(cron.NewParser(
				cron.SecondOptional|cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor,
			)))
	}

	for _, mod := range modules {
		if err := mod.Module.Start(); err != nil {
			defaultLogger.Errorf("module [%s] start:%s\n", mod.Name, err.Error())
			return
		}

		defaultLogger.Infof("module [%s] priority:%d start success", mod.Name, mod.Priority)
	}

	if s.openMetric {
		s.router.Handle("/metrics", promhttp.Handler())
	}

	defaultServer.router.HandleFunc("/log_level", defaultLogger.ServeHTTP)

	if s.pprof {
		s.router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
		s.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		s.router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	var port string
	if *flagPort != "" {
		port = *flagPort
	} else {
		if yamlNode, ok := fileCfg["port"]; ok {
			if err := yamlNode.Decode(&port); err != nil {
				defaultLogger.Error("decode port config:" + err.Error())
				return
			}
		}
	}

	certFile := fileCfg["certFile"].Value
	keyFile := fileCfg["keyFile"].Value
	if certFile != "" {
		if defaultServer.CertFile != "" {
			defaultLogger.Infof("certFile is already set by option,use config instead")
		}
		defaultServer.CertFile = certFile
	}

	if keyFile != "" {
		if defaultServer.KeyFile != "" {
			defaultLogger.Infof("keyFile is already set by option,use config instead")
		}
		defaultServer.KeyFile = keyFile
	}

	defaultServer.cron.Start()

	svr:=hihttp.NewServer(":"+port, defaultServer.router)
	svr.RegisterOnShutdown(func() {
		s.Stop()
	})

	if defaultServer.CertFile=="" && defaultServer.KeyFile==""{
		if err:=svr.ListenAndServe();err!=nil&&err!=http.ErrServerClosed{
			defaultLogger.Error("ListenAndServe:" + err.Error())
		}
	}else{
		if err:=svr.ListenAndServeTLS(defaultServer.CertFile,
			defaultServer.KeyFile);err != nil&&err!=http.ErrServerClosed {
			defaultLogger.Error("ListenAndServe:" + err.Error())
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
			defaultLogger.Error("module stop:"+err.Error(), zap.String("moduleName", mod.Name))
			// not return
		} else {
			defaultLogger.Info("stop success",
				zap.String("moduleName", mod.Name), zap.Int("priority", mod.Priority))
		}
	}
}

type Option func()

func WithConfig(filename string) Option {
	return func() {
		defaultServer.configFile = filename
	}
}

func WithHttps(certFile, keyFile string) Option {
	return func() {
		defaultServer.CertFile = certFile
		defaultServer.CertFile = keyFile

	}
}

func WithPprof() Option {
	return func() {
		defaultServer.pprof = true
	}
}

func WithCron() Option {
	return func() {
		defaultServer.openCron = true
	}
}

func WithMetric() Option {
	return func() {
		defaultServer.openMetric = true
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

func Cron() *cron.Cron {
	if defaultServer.cron == nil {
		panic("cron not start")
	}

	return defaultServer.cron
}
