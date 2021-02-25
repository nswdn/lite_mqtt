package config

import (
	"encoding/json"
	"io/ioutil"
)

var conf config

const configFileName = "config.json"

type config struct {
	AuthEnable bool   `json:"auth_enable"`
	AuthType   string `json:"auth_type"` // mysql/redis

	QuerySQL   string `json:"query_sql"`
	DataSource string `json:"data_source"`
}

func init() {
	content, err := ioutil.ReadFile(configFileName)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(content, &conf); err != nil {
		panic(err)
	}
}

func DataSourceAndQuerySQL() (string, string) {
	return conf.DataSource, conf.QuerySQL
}

func EnableAuth() bool {
	return conf.AuthEnable
}

func AuthType() string {
	return conf.AuthType
}
