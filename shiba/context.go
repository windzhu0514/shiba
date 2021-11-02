package shiba

import (
	"flag"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/robfig/cron/v3"
	"github.com/windzhu0514/shiba/log"
)

type Context struct {
	s *Server
}

func (c *Context) Logger(name string) log.Logger {
	return defaultLogger.Clone(name)
}

func (c *Context) DBMaster(name string) (*sqlx.DB, error) {
	return db.Master(name)
}

func (c *Context) DBSlave(name string) (*sqlx.DB, error) {
	return db.Slave(name)
}

func (c *Context) Redis(name string) (RedisCmdable, error) {
	return redisx.Get(name)
}

func (c *Context) Router() *mux.Router {
	return c.s.router
}

func (c *Context) FlagSet() *flag.FlagSet {
	return c.s.flags
}

func (c *Context) Cron() *cron.Cron {
	if c.s.cron == nil {
		panic("cron not start")
	}

	return c.s.cron
}

func (c *Context) Config() ServerConfig {
	return c.s.Config
}
