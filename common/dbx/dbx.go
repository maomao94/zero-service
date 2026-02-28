package dbx

import (
	"context"
	"database/sql"
	"strings"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlserver"
	_ "github.com/go-sql-driver/mysql" // MySQL驱动
	_ "github.com/lib/pq"              // PostgreSQL驱动
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/postgres"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type DatabaseType string

const (
	DatabaseTypeMySQL    DatabaseType = "mysql"
	DatabaseTypePostgres DatabaseType = "postgres"
	DatabaseTypeSQLite   DatabaseType = "sqlite"
	DatabaseTypeTAOS     DatabaseType = "taos"
)

func ParseDatabaseType(datasource string) DatabaseType {
	datasource = strings.TrimSpace(datasource)
	if strings.HasPrefix(datasource, "file:") || strings.Contains(datasource, ".db") {
		return DatabaseTypeSQLite
	} else if strings.Contains(datasource, "http") || strings.Contains(datasource, "https") {
		return DatabaseTypeTAOS
	} else if strings.Contains(datasource, "@tcp(") {
		return DatabaseTypeMySQL
	} else if strings.HasPrefix(strings.ToLower(datasource), "postgres") {
		return DatabaseTypePostgres
	} else {
		return DatabaseTypeMySQL
	}
}

// New 根据数据源URL自动判断数据库类型并创建连接
// 支持的数据库类型：
// - SQLite: 数据源以"file:"开头或包含".db"后缀
// - TAOS: 数据源包含"http"或"https"
// - MySQL: 数据源包含":"和"@tcp("
// - PostgreSQL: 数据源以"postgres://"开头
func New(datasource string, opts ...sqlx.SqlOption) sqlx.SqlConn {
	dbType := ParseDatabaseType(datasource)
	switch dbType {
	case DatabaseTypeSQLite:
		return NewSqlite(datasource, opts...)
	case DatabaseTypeTAOS:
		return NewTaos(datasource, opts...)
	case DatabaseTypePostgres:
		return postgres.New(datasource, opts...)
	default: // DatabaseTypeMySQL and others
		return sqlx.NewMysql(datasource, opts...)
	}
}

type SqlConnAdapter struct {
	conn sqlx.SqlConn
	db   *sql.DB
}

func NewSqlConnAdapter(conn sqlx.SqlConn) (*SqlConnAdapter, error) {
	db, err := conn.RawDB()
	if err != nil {
		return nil, err
	}
	return &SqlConnAdapter{
		conn: conn,
		db:   db,
	}, nil
}

func (a *SqlConnAdapter) Begin() (*sql.Tx, error) {
	return a.db.Begin()
}

func (a *SqlConnAdapter) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return a.db.BeginTx(ctx, opts)
}

func (a *SqlConnAdapter) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return a.conn.ExecCtx(ctx, query, args...)
}

func (a *SqlConnAdapter) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return a.db.PrepareContext(ctx, query)
}

func (a *SqlConnAdapter) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return a.db.QueryContext(ctx, query, args...)
}

func (a *SqlConnAdapter) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return a.db.QueryRowContext(ctx, query, args...)
}

func NewQoqu(datasource string, opts ...sqlx.SqlOption) *goqu.Database {
	var conn sqlx.SqlConn
	dbType := ParseDatabaseType(datasource)
	switch dbType {
	case DatabaseTypeSQLite:
		conn = NewSqlite(datasource, opts...)
	case DatabaseTypeTAOS:
		conn = NewTaos(datasource, opts...)
	case DatabaseTypePostgres:
		conn = postgres.New(datasource, opts...)
	default: // DatabaseTypeMySQL and others
		conn = sqlx.NewMysql(datasource, opts...)
	}
	adapter, err := NewSqlConnAdapter(conn)
	if err != nil {
		logx.Errorf("Failed to create SqlConnAdapter: %v", err)
		db, _ := conn.RawDB()
		database := goqu.New(string(dbType), db)
		database.Logger(&QoquLog{})
		return database
	}
	database := goqu.New(string(dbType), adapter)
	database.Logger(&QoquLog{})
	return database
}

type QoquLog struct {
}

func (log *QoquLog) Printf(format string, v ...any) {
	logx.Infof(format, v...)
}
