package model

import (
	"errors"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var ErrNotFound = sqlx.ErrNotFound
var ErrNoRowsUpdate = errors.New("update db no rows change")

type CacheEntry[T any] struct {
	Data  T
	Valid bool
}

type DatabaseType string

const (
	DatabaseTypeMySQL      DatabaseType = "mysql"
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
	DatabaseTypeSQLite     DatabaseType = "sqlite"
	DatabaseTypeTAOS       DatabaseType = "taos"
)
