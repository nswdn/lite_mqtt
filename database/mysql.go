package database

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type mysqlDB struct {
	author
	conn     *sql.DB
	querySQL string
	saveSQL  string
}

// returns error when open data source failed
func calMysql(dataSource, querySQL, saveSQL string) error {
	open, err := sql.Open("mysql", dataSource)
	if err != nil {
		return err
	}
	db = &mysqlDB{conn: open, querySQL: querySQL, saveSQL: saveSQL}
	return nil
}

func (m *mysqlDB) auth(user, pwd string) (bool, error) {
	query, err := m.conn.Query(m.querySQL, user, pwd)
	if err != nil {
		return false, err
	}

	defer query.Close()
	return query.Next(), nil
}

func (m *mysqlDB) save(clientID string, content []byte) error {
	_, err := m.conn.Exec(m.saveSQL, clientID, string(content))
	return err
}
