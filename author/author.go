package author

import (
	"excel_parser/config"
	"log"
)

type author interface {
	auth(user, pwd string) (bool, error)
}

var authorization author

const (
	MySQL = "mysql"
	Redis = "redis"
)

// if enable auth is true. auth user from databases.
func init() {
	if config.EnableAuth() {
		switch config.AuthType() {
		case MySQL:
			if err := calMysql(config.DataSourceAndQuerySQL()); err != nil {
				panic(err)
			}
			log.Println("authorization with mysql")
		case Redis:

		}
	}
}

func Auth(user, pwd string) bool {
	auth, err := authorization.auth(user, pwd)
	if err != nil {
		log.Println(err)
	}
	return auth
}
