package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ OrderTxnModel = (*customOrderTxnModel)(nil)

type (
	// OrderTxnModel is an interface to be customized, add more methods here,
	// and implement the added methods in customOrderTxnModel.
	OrderTxnModel interface {
		orderTxnModel
		withSession(session sqlx.Session) OrderTxnModel
	}

	customOrderTxnModel struct {
		*defaultOrderTxnModel
	}
)

// NewOrderTxnModel returns a model for the database table.
func NewOrderTxnModel(conn sqlx.SqlConn, opts ...ModelOption) OrderTxnModel {
	return &customOrderTxnModel{
		defaultOrderTxnModel: newOrderTxnModel(conn, opts...),
	}
}

func (m *customOrderTxnModel) withSession(session sqlx.Session) OrderTxnModel {
	return &customOrderTxnModel{
		defaultOrderTxnModel: newOrderTxnModel(sqlx.NewSqlConnFromSession(session), WithDBType(m.dbType)),
	}
}
