package author

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type mysqlAuthor struct {
	author
	conn     *sql.DB
	querySQL string
}

// returns error when open data source failed
func calMysql(dataSource, querySQL string) error {
	open, err := sql.Open("mysql", dataSource)
	if err != nil {
		return err
	}
	authorization = &mysqlAuthor{conn: open, querySQL: querySQL}
	return nil
}

func (m *mysqlAuthor) auth(user, pwd string) (bool, error) {
	query, err := m.conn.Query(m.querySQL, user, pwd)
	if err != nil {
		return false, err
	}

	defer query.Close()
	return query.Next(), nil
}
