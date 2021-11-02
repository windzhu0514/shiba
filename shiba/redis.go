package shiba

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-redis/redis/v8"
)

type RedisCmdable interface {
	redis.Cmdable
}

func init() {
	registerModule(-98, redisx)
}

type redisConfig struct {
	Disable      bool     `yaml:"disable"`
	IsCluster    bool     `yaml:"isCluster"` // 是否是集群
	Address      []string `yaml:"address"`   // 大于1个地址为集群
	DBIndex      int      `yaml:"dbIndex"`
	Password     string   `yaml:"password"`
	PoolSize     int      `yaml:"poolSize"`
	MinIdleConns int      `yaml:"minIdleConns"`
}

var redisx = &redisPool{}

type redisPool struct {
	Config  map[string]redisConfig `yaml:"redis"`
	poolsMu sync.Mutex
	pools   map[string]redis.Cmdable
}

func (p *redisPool) Name() string {
	return "redis"
}

func (p *redisPool) Init(ctx *Context) error {
	p.pools = make(map[string]redis.Cmdable)
	return nil
}

func (p *redisPool) Start(ctx *Context) error {
	if err := p.testAll(); err != nil {
		return err
	}

	return nil
}

func (p *redisPool) testAll() error {
	if len(p.Config) == 0 {
		defaultLogger.Clone("redis").Debug("has no redis config,the module will not initialize")
		return nil
	}

	for name, cfg := range p.Config {
		if cfg.Disable {
			continue
		}

		pool, err := p.new(name, cfg)
		if err != nil {
			return fmt.Errorf(name+":%w", err)
		}

		if client, ok := pool.(*redis.ClusterClient); ok {
			if err := client.Close(); err != nil {
				return fmt.Errorf(name+":%w", err)
			}
		} else if client, ok := pool.(*redis.Client); ok {
			if err := client.Close(); err != nil {
				return fmt.Errorf(name+":%w", err)
			}
		}
	}

	return nil
}

func (p *redisPool) Stop(ctx *Context) error {
	for name, pool := range p.pools {
		if pool == nil {
			continue
		}

		if client, ok := pool.(*redis.ClusterClient); ok {
			if err := client.Close(); err != nil {
				return fmt.Errorf(name+":%w", err)
			}
		} else if client, ok := pool.(*redis.Client); ok {
			if err := client.Close(); err != nil {
				return fmt.Errorf(name+":%w", err)
			}
		}
	}

	return nil
}

func (p *redisPool) Get(name string) (RedisCmdable, error) {
	if name == "" {
		name = "default"
	}

	pool, ok := p.pools[name]
	if ok {
		return pool, nil
	}

	p.poolsMu.Lock()
	defer p.poolsMu.Unlock()

	pool, ok = p.pools[name]
	if ok {
		return pool, nil
	}

	cfg, ok := p.Config[name]
	if !ok {
		return nil, errors.New("cant find redis config:" + name)
	}

	if cfg.Disable {
		return nil, errors.New("redis config is disable:" + name)
	}

	pool, err := p.new(name, cfg)
	if err != nil {
		return nil, err
	}

	p.pools[name] = pool

	return pool, nil
}

func (p *redisPool) new(name string, cfg redisConfig) (RedisCmdable, error) {
	if len(cfg.Address) == 0 {
		return nil, errors.New(name + ":address is empty")
	}

	var client RedisCmdable
	if cfg.IsCluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.Address,
			PoolSize:     cfg.PoolSize,
			Password:     cfg.Password,
			MinIdleConns: cfg.MinIdleConns,
			// 避免访问从节点，和MinIdleConns为0结合，间接避免从节点建立连接与访问
			ReadOnly:       false,
			RouteByLatency: false,
			RouteRandomly:  false,
			//Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			//	var d net.Dialer
			//	return d.DialContext(ctx, network, addr)
			//},
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:         cfg.Address[0],
			DB:           cfg.DBIndex,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			Password:     cfg.Password,
		})

	}

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf(name+":%w", err)
	}

	return client, nil
}
