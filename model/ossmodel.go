package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ OssModel = (*customOssModel)(nil)

type (
	// OssModel is an interface to be customized, add more methods here,
	// and implement the added methods in customOssModel.
	OssModel interface {
		ossModel
		withSession(session sqlx.Session) OssModel
	}

	customOssModel struct {
		*defaultOssModel
	}
)

// NewOssModel returns a model for the database table.
func NewOssModel(conn sqlx.SqlConn) OssModel {
	return &customOssModel{
		defaultOssModel: newOssModel(conn),
	}
}

func (m *customOssModel) withSession(session sqlx.Session) OssModel {
	return NewOssModel(sqlx.NewSqlConnFromSession(session))
}
