// Package server
// 数据库
// 支持主从
// 从未配置，默认使用主配置
package shiba

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	 _ "github.com/go-sql-driver/mysql"
)

func init() {
	registerModule(-99, db)
}

var db = &database{}

type connectConfig struct {
	DataSourceName  string        `yaml:"dataSourceName"` // 连接字符串
	MaxOpenConns    int           `yaml:"maxOpenConns"`
	MaxIdleConns    int           `yaml:"maxIdleConns"`
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime"`
}
type databaseConfig struct {
	Disable    bool          `yaml:"disable"`
	DriverName string        `yaml:"driverName"`
	Master     connectConfig `yaml:"master"`
	Slave      connectConfig `yaml:"slave"`
}

type database struct {
	Config    map[string]databaseConfig `yaml:"database"`
	dbsMu     sync.RWMutex
	dbSlaves  map[string]*sqlx.DB
	dbMasters map[string]*sqlx.DB
}

func (p *database) Name() string {
	return "database"
}

func (p *database) Init() error {
	db.dbMasters = make(map[string]*sqlx.DB)
	db.dbSlaves = make(map[string]*sqlx.DB)
	return nil
}

func (db *database) Start() error {
	if err := db.testAll(); err != nil {
		return err
	}

	return nil
}

func (db *database) testAll() error {
	for name, cfg := range db.Config {
		if cfg.Disable {
			continue
		}

		if cfg.DriverName == "" {
			return errors.New(name + " driverName is empty")
		}

		if cfg.Master.DataSourceName == "" && cfg.Slave.DataSourceName == "" {
			return errors.New(name + " master and slave dataSourceName is both empty")
		}

		if cfg.Master.DataSourceName != "" {
			xdb, err := db.new(cfg.DriverName, cfg.Master)
			if err != nil {
				return fmt.Errorf(name+" master:%w", err)
			}

			if err = xdb.Close(); err != nil {
				return fmt.Errorf(name+" master:%w", err)
			}
		}

		if cfg.Slave.DataSourceName != "" {
			xdb, err := db.new(cfg.DriverName, cfg.Slave)
			if err != nil {
				return fmt.Errorf(name+" slave:%w", err)
			}

			if err = xdb.Close(); err != nil {
				return fmt.Errorf(name+" slave:%w", err)
			}
		}
	}

	return nil
}

func (db *database) Stop() error {
	for name, db := range db.dbMasters {
		if db == nil {
			continue
		}

		if err := db.Close(); err != nil {
			return fmt.Errorf(name+" master:%w", err)
		}
	}

	for name, db := range db.dbSlaves {
		if db == nil {
			continue
		}

		if err := db.Close(); err != nil {
			return fmt.Errorf(name+" slave:%w", err)
		}
	}

	return nil
}

func (db *database) Master(name string) (*sqlx.DB, error) {
	if name == "" {
		name = "default"
	}

	dbMaster, ok := db.dbMasters[name]
	if ok {
		return dbMaster, nil
	}

	db.dbsMu.Lock()
	defer db.dbsMu.Unlock()

	dbMaster, ok = db.dbMasters[name]
	if ok {
		return dbMaster, nil
	}

	cfg, ok := db.Config[name]
	if !ok {
		return nil, errors.New("cant find sql config:" + name)
	}

	if cfg.Disable {
		return nil, errors.New("sql config is disable:" + name)
	}

	xdb, err := db.new(cfg.DriverName, cfg.Master)
	if err != nil {
		return nil, fmt.Errorf(name+" master:%w", err)
	}

	db.dbMasters[name] = xdb

	return xdb, nil
}

func (db *database) Slave(name string) (*sqlx.DB, error) {
	if name == "" {
		name = "default"
	}

	dbSlave, ok := db.dbSlaves[name]
	if ok {
		return dbSlave, nil
	}

	db.dbsMu.Lock()
	defer db.dbsMu.Unlock()

	dbSlave, ok = db.dbSlaves[name]
	if ok {
		return dbSlave, nil
	}

	dbMaster, ok := db.dbMasters[name]
	if ok {
		return dbMaster, nil
	}

	cfg, ok := db.Config[name]
	if !ok {
		return nil, errors.New("cant find sql config:" + name)
	}

	if cfg.Disable {
		return nil, errors.New("sql config is disable:" + name)
	}

	if cfg.Slave.DataSourceName == "" {
		xdb, err := db.new(cfg.DriverName, cfg.Master)
		if err != nil {
			return nil, fmt.Errorf(name+" master:%w", err)
		}

		db.dbMasters[name] = xdb
		return xdb, nil
	}

	xdb, err := db.new(cfg.DriverName, cfg.Slave)
	if err != nil {
		return nil, fmt.Errorf(name+" slave:%w", err)
	}

	db.dbSlaves[name] = xdb
	return xdb, nil
}

func (db *database) new(driverName string, connCfg connectConfig) (*sqlx.DB, error) {
	if driverName == "" || connCfg.DataSourceName == "" {
		return nil, errors.New("driverName or dataSourceName is empty")
	}

	xdb, err := sqlx.Connect(driverName, connCfg.DataSourceName)
	if err != nil {
		return nil, err
	}

	// database/sql/sql.go
	// const defaultMaxIdleConns = 2
	maxIdleConns := connCfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 2
	}

	xdb.SetMaxOpenConns(connCfg.MaxOpenConns)
	xdb.SetMaxIdleConns(connCfg.MaxIdleConns)
	xdb.SetConnMaxIdleTime(connCfg.ConnMaxIdleTime)
	xdb.SetConnMaxLifetime(connCfg.ConnMaxLifetime)

	if err = xdb.Ping(); err != nil {
		return nil, err
	}

	return xdb, nil
}
