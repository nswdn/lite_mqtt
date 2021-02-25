package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

var conf config

const configFileName = "config.json"

type config struct {
	Database string `json:"database"` // mysql/redis

	DataSource string `json:"data_source"`

	AuthEnable bool   `json:"auth_enable"`
	AuthSQL    string `json:"auth_sql"`

	SaveSQL     string `json:"save_sql"`
	SavePublish bool   `json:"save_publish"`

	TlsMode  bool   `json:"tls_mode"`
	CrtFile  string `json:"crt_file"`
	PrivFile string `json:"priv_file"`

	Addr       string `json:"addr"`
	ListenPort int    `json:"listen_port"`
}

func init() {
	content, err := ioutil.ReadFile(configFileName)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(content, &conf); err != nil {
		panic(err)
	}

	if strings.TrimSpace(conf.Database) == "" {
		if conf.SavePublish || conf.AuthEnable {
			panic("specify a database in config file to save publish message or authorization")
		}
	}
}

func DataSource() string {
	return conf.DataSource
}

func SQL() (string, string) {
	return conf.AuthSQL, conf.SaveSQL
}

func EnableAuth() bool {
	return conf.AuthEnable
}

func DatabaseType() string {
	return conf.Database
}

func SavePublish() bool {
	return conf.SavePublish
}

func TlsMode() bool {
	return conf.TlsMode
}

func Addr() string {
	return fmt.Sprintf("%s:%d", conf.Addr, conf.ListenPort)
}

func TlsFile() (string, string) {
	return conf.CrtFile, conf.PrivFile
}
