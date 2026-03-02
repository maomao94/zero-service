package {{.pkg}}

import (
    "errors"
    "github.com/zeromicro/go-zero/core/stores/sqlx"
    "regexp"
)

var ErrNotFound = sqlx.ErrNotFound
var ErrNoRowsUpdate = errors.New("update db no rows change")

type postgresResult struct {
	id int64
}

func (r *postgresResult) LastInsertId() (int64, error) {
	return r.id, nil
}

func (r *postgresResult) RowsAffected() (int64, error) {
	// Assuming one row is inserted
	return 1, nil
}

type DatabaseType string

const (
	dbTag                             = "db"
	DatabaseTypeMySQL    DatabaseType = "mysql"
	DatabaseTypePostgres DatabaseType = "postgres"
	DatabaseTypeSQLite   DatabaseType = "sqlite"
	DatabaseTypeTAOS     DatabaseType = "taos"
)

func adaptSQLPlaceholders(sqlTemplate, dbType string) string {
	if !strings.EqualFold(dbType, string(DatabaseTypePostgres)) {
		re := regexp.MustCompile(`\$\d+`)
		sqlTemplate = re.ReplaceAllString(sqlTemplate, "?")
	}
	return sqlTemplate
}