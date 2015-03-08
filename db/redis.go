package db

import (
	"fmt"
	"github.com/netrack/netrack/db/record"

	"github.com/garyburd/redigo/redis"
)

const (
	SetAddCommand    = "SADD"
	SetMemersCommand = "SMEMBERS"
	FlushAllCommand  = "FLUSHALL"
)

func dial(addr string) func() (redis.Conn, error) {
	//TODO: add support of unix sockets
	return func() (redis.Conn, error) {
		return redis.Dial("tcp", addr)
	}
}

type redisStmt struct {
	conn redis.Conn
}

type redisDB struct {
	pool *redis.Pool
}

func (db *redisDB) Add(r record.Type, v interface{}) error {
	conn := db.pool.Get()
	defer conn.Close()

	return conn.Send(SetAddCommand, r, v)
}

func (db *redisDB) Get(r record.Type, v interface{}) error {
	conn := db.pool.Get()
	defer conn.Close()

	reply, err := conn.Do(SetMemersCommand, r)
	fmt.Println(reply)
	return err
}

func (db *redisDb) Prepare(query string) (Stmt, error) {
}

func (db *redisDB) Close() error {
	return db.pool.Close()
}

func Open(config *Config) *redisDB {
	endpoint := fmt.Sprintf("%s:%d", config.Addr, config.Port)
	pool := &redis.Pool{Dial: dial(endpoint)}
	return &redisDB{pool}
}
