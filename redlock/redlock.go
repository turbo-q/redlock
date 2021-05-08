package redlock

import (
	"fmt"
	"log"
	"time"

	"github.com/garyburd/redigo/redis"
)

var rss = []string{
	"127.0.0.1:6379",
	"127.0.0.1:6380",
	"127.0.0.1:6381",
	"127.0.0.1:6382",
	"127.0.0.1:6383",
}

const (
	connTimeout = 30 * time.Millisecond
	// 释放锁lua脚本
	releaseScript = `if redis.call("get",KEYS[1]) == ARGV[1] then
						return redis.call("del",KEYS[1])
					else
						return 0
					end`
	// 看门狗续期脚本
	renewScript = `if redis.call("get",KEYS[1]) == ARGV[1] then
						return redis.call("expire",KEYS[1],30000)
					else
						return 0
					end`
)

var pools []*redis.Pool

func init() {
	for _, rs := range rss {
		// timeout 远小于valid timeout，避免长时间与不可通信节点连接
		pool, err := createPool(rs, redis.DialConnectTimeout(connTimeout))
		if err != nil {
			log.Println(rs, "create pool err", err)
			return
		}
		pools = append(pools, pool)
		log.Printf("%s create pool successed", rs)
	}

}

func createPool(addr string, opts ...redis.DialOption) (*redis.Pool, error) {
	return &redis.Pool{
		MaxIdle:     100,
		MaxActive:   4000,
		IdleTimeout: time.Minute,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr, opts...)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}, nil
}

type RedLock struct {
	K   string
	V   string // 值使用unique random value
	Cfg Config
}

type Config struct {
	Timeout int // 有效时间
}

// 获取锁
func (rl *RedLock) Lock() bool {
	start := time.Now()

	var successNum int // 设置成功的节点数
	for _, pool := range pools {
		conn := pool.Get()
		state, err := redis.String(conn.Do("SET", rl.K, rl.V, "NX", "PX", rl.Cfg.Timeout))
		if err != nil && err != redis.ErrNil {
			fmt.Println(err)
		}
		if state == "OK" {
			successNum++
		}
	}

	threshold := len(pools)/2 + 1 // 设置成功的阈值
	// 成功数大于一半，并且总的消耗小于valid time
	fmt.Println(successNum, threshold, time.Since(start) <= time.Duration(rl.Cfg.Timeout))
	if successNum >= threshold && time.Since(start).Milliseconds() <= int64(rl.Cfg.Timeout) {
		return true
	}

	return false
}

// 释放锁
func (rl *RedLock) Release() {
	for _, pool := range pools {
		conn := pool.Get()
		script := redis.NewScript(1, releaseScript)
		_, err := script.Do(conn, rl.K, rl.V)
		if err != nil {
			log.Println("释放锁出错", err)
		}
	}
}
