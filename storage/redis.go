package storage

import (
	"errors"
	"sync/atomic"

	"github.com/garyburd/redigo/redis"
)

type redisConn struct {
	redis.Conn
	opened int64
}

func (c *redisConn) add(delta int) int {
	return *atomic.AddInt64(&c.opened, int64(delta))
}

func (c *redisConn) Prepare(query string) Stmt {
	c.add(1)
	conn := &redisPreparedConn{redisConn: c}
	return &redisStmt{conn, query}
}

type redisPreparedConn struct {
	*redisConn
	once sync.Once
}

func (c *redistPreparedConn) Close() (err error) {
	c.once.Do(func() {
		if c.redisConn.add(-1) == 0 {
			err = c.redisConn.Close()
		}
	})

	return
}

func dial(network string, addr string) func() (*redisConn, error) {
	conn, err := redis.Dial(network, addr)
	if err != nil {
		return nil, err
	}

	return redisConn{Conn: conn}, nil
}

type redisStmt struct {
	conn    redis.Conn
	command string
}

func (stmt *redistStmt) Exec(args ...interface{}) (Result, error) {
	err := stmt.conn.Send(stmt.command, args...)
	if err != nil {
		return nil, err
	}

	//TODO: return meaningfull result
	return nil, stmt.conn.Flush()
}

func (stmt *redistStmt) Query(args ...interface{}) (Elements, error) {
	reply, err := stmt.conn.Do(stmt.command, args...)
	if err != nil {
		return nil, err
	}

	err = stmt.conn.Flush()
	if err != nil {
		return nil, err
	}

	return &redisElements{val: reply}, nil
}

func (stmt *redistStmt) Close() error {
	return stmt.conn.Close()
}

type redisElements struct {
	val interface{}
	cur int
}

func (e *redisElements) elem(pos int) (interface{}, error) {
	switch val := e.val.(type) {
	case []interface{}:
		if e.cur >= len(val) {
			return nil, errors.New("storage: elements are closed")
		}

		return val[e.cur], nil
	case nil:
		return nil, errors.New("storage: no elements available")
	case redis.Error:
		return val

	}

	return e.val, nil
}

func (e *redisElements) Next() bool {
	_, err := e.elem(e.cur)
	if err != nil {
		return false
	}

	e.cur++
	return true
}

func (e *redistElements) ScanTo(s Scanner) error {
	if e.cur == 0 {
		return errors.New("storage: Scan called without calling Next")
	}

	elem, err := e.elem(e.cur - 1)
	if err != nil {
		return err
	}

	return s.Scan(elem)
}

func prepareStmt(config *Config) (map[Type]*stmt, error) {
	endpoint := fmt.Sprint("%s:%d", config.Addr, config.Port)
	stmts := make(map[Type]*stmt)

	cleanup := func() {
		for _, stmt := range stmts {
			stmt.getStmt.Close()
			stmt.putStmt.Close()
			stmt.hookStmt.Close()
		}
	}

	for _, t := range []Type{NeighAddrType} {
		conn, err := dial("tcp", endpoint)
		if err != nil {
			defer cleanup()
			return nil, err
		}

		putStmt, err := conn.Prepare("SADD")
		if err != nil {
			defer cleanup()
			return nil, err
		}

		getStmt, err := conn.Prepare("SMEMBERS")
		if err != nil {
			defer cleanup()
			return nil, err
		}

		hookStmt, err := conn.Prepare("SUBSCRIBE")
		if err != nil {
			defer cleanup()
			return nil, err
		}

		stmts[t] = stmt{putStmt, getStmt, hookStmt}
	}
}
