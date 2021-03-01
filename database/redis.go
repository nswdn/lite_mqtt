package database

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
)

type redisDB struct {
	author
	conn *redis.Client
}

func calRedis() {
	options := &redis.Options{
		Network:      "tcp",
		Addr:         "127.0.0.1:6379",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 1,
	}

	client := redis.NewClient(options)

	db = &redisDB{
		conn: client,
	}
}

func (re *redisDB) auth(user, pwd string) (bool, error) {
	result := re.conn.Get(user)
	if result.Err() != nil {
		return false, result.Err()
	}
	if result.String() != pwd {
		return false, errors.New(fmt.Sprintf("auth failed, user:[%s] pwd: [%s]", user, pwd))
	}

	return true, nil
}

func (re *redisDB) save(clientID string, content []byte) error {
	result := re.conn.LPush(clientID, content)
	return result.Err()
}

func (re *redisDB) Close() error {
	return re.conn.Close()
}
