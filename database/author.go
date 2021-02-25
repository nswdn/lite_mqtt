package database

import (
	"excel_parser/config"
	"log"
)

type author interface {
	auth(user, pwd string) (bool, error)
	save(clientID string, content []byte) error
	close() error
}

var db author

const (
	MySQL = "mysql"
	Redis = "redis"
)

// if enable auth is true. auth user from databases.
func init() {
	switch config.DatabaseType() {
	case MySQL:
		if !config.SavePublish() && !config.EnableAuth() {
			return
		}
		query, save := config.SQL()
		if err := calMysql(config.DataSource(), query, save); err != nil {
			panic(err)
		}
		log.Println("db with mysql")
	case Redis:
		panic("unsupported db by redis")
	}
}

func Auth(user, pwd string) bool {
	auth, err := db.auth(user, pwd)
	if err != nil {
		log.Println(err)
	}
	return auth
}

func Save(clientID string, content []byte) bool {
	err := db.save(clientID, content)
	if err != nil {
		log.Println(err)
	}
	return err == nil
}

func Close() error {
	return db.close()
}
