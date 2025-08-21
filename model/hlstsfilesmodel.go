package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ HlsTsFilesModel = (*customHlsTsFilesModel)(nil)

type (
	// HlsTsFilesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customHlsTsFilesModel.
	HlsTsFilesModel interface {
		hlsTsFilesModel
		withSession(session sqlx.Session) HlsTsFilesModel
	}

	customHlsTsFilesModel struct {
		*defaultHlsTsFilesModel
	}
)

// NewHlsTsFilesModel returns a model for the database table.
func NewHlsTsFilesModel(conn sqlx.SqlConn) HlsTsFilesModel {
	return &customHlsTsFilesModel{
		defaultHlsTsFilesModel: newHlsTsFilesModel(conn),
	}
}

func (m *customHlsTsFilesModel) withSession(session sqlx.Session) HlsTsFilesModel {
	return NewHlsTsFilesModel(sqlx.NewSqlConnFromSession(session))
}
