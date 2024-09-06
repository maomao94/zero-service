// Code generated by goctl. DO NOT EDIT.
// versions:
//  goctl version: 1.7.1

package genModel

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/stores/builder"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/stringx"
)

var (
	orderTxnFieldNames          = builder.RawFieldNames(&OrderTxn{})
	orderTxnRows                = strings.Join(orderTxnFieldNames, ",")
	orderTxnRowsExpectAutoSet   = strings.Join(stringx.Remove(orderTxnFieldNames, "`id`", "`create_at`", "`create_time`", "`created_at`", "`update_at`", "`update_time`", "`updated_at`"), ",")
	orderTxnRowsWithPlaceHolder = strings.Join(stringx.Remove(orderTxnFieldNames, "`id`", "`create_at`", "`create_time`", "`created_at`", "`update_at`", "`update_time`", "`updated_at`"), "=?,") + "=?"
)

type (
	orderTxnModel interface {
		Insert(ctx context.Context, session sqlx.Session, data *OrderTxn) (sql.Result, error)
		FindOne(ctx context.Context, id int64) (*OrderTxn, error)
		FindOneByMchIdMchOrderNo(ctx context.Context, mchId string, mchOrderNo string) (*OrderTxn, error)
		FindOneByTxnId(ctx context.Context, txnId string) (*OrderTxn, error)
		Update(ctx context.Context, session sqlx.Session, data *OrderTxn) (sql.Result, error)
		UpdateWithVersion(ctx context.Context, session sqlx.Session, data *OrderTxn) error
		Trans(ctx context.Context, fn func(context context.Context, session sqlx.Session) error) error
		SelectBuilder() squirrel.SelectBuilder
		DeleteSoft(ctx context.Context, session sqlx.Session, data *OrderTxn) error
		FindSum(ctx context.Context, sumBuilder squirrel.SelectBuilder, field string) (float64, error)
		FindCount(ctx context.Context, countBuilder squirrel.SelectBuilder, field string) (int64, error)
		FindAll(ctx context.Context, rowBuilder squirrel.SelectBuilder, orderBy string) ([]*OrderTxn, error)
		FindPageListByPage(ctx context.Context, rowBuilder squirrel.SelectBuilder, page, pageSize int64, orderBy string) ([]*OrderTxn, error)
		FindPageListByPageWithTotal(ctx context.Context, rowBuilder squirrel.SelectBuilder, page, pageSize int64, orderBy string) ([]*OrderTxn, int64, error)
		FindPageListByIdDESC(ctx context.Context, rowBuilder squirrel.SelectBuilder, preMinId, pageSize int64) ([]*OrderTxn, error)
		FindPageListByIdASC(ctx context.Context, rowBuilder squirrel.SelectBuilder, preMaxId, pageSize int64) ([]*OrderTxn, error)
		Delete(ctx context.Context, session sqlx.Session, id int64) error
	}

	defaultOrderTxnModel struct {
		conn  sqlx.SqlConn
		table string
	}

	OrderTxn struct {
		Id                int64        `db:"id"`
		CreateTime        time.Time    `db:"create_time"`
		UpdateTime        time.Time    `db:"update_time"`
		DeleteTime        time.Time    `db:"delete_time"`
		DelState          int64        `db:"del_state"`
		Version           int64        `db:"version"`              // 版本号
		TxnId             string       `db:"txn_id"`               // 订单号
		OriTxnId          string       `db:"ori_txn_id"`           // 原订单号
		TxnTime           time.Time    `db:"txn_time"`             // 订单时间
		TxnDate           time.Time    `db:"txn_date"`             // 订单日期
		MchId             string       `db:"mch_id"`               // 商户id
		MchOrderNo        string       `db:"mch_order_no"`         // 商户订单号
		PayType           string       `db:"pay_type"`             // 支付类型 alipay-支付宝,wxpay-微信
		TxnType           int64        `db:"txn_type"`             // 交易类型,1000-消费,2000-退款
		TxnChannel        string       `db:"txn_channel"`          // 交易渠道
		TxnAmt            int64        `db:"txn_amt"`              // 支付金额,单位分
		RealAmt           int64        `db:"real_amt"`             // 实付金额,单位分
		Result            string       `db:"result"`               // 交易结果,U-未处理,P-交易处理中,F-失败,T-超时,C-关闭,S-成功
		Body              string       `db:"body"`                 // 商品描述信息
		Extra             string       `db:"extra"`                // 特定渠道发起时额外参数
		UserId            int64        `db:"user_id"`              // 用户id
		ChannelUser       string       `db:"channel_user"`         // 渠道用户标识,如微信openId,支付宝账号
		ChannelPayTime    sql.NullTime `db:"channel_pay_time"`     // 渠道支付执行成功时间
		ChannelOrderNo    string       `db:"channel_order_no"`     // 渠道订单号
		PayerAcct         string       `db:"payer_acct"`           // 付款款账户
		PayerAcctName     string       `db:"payer_acct_name"`      // 付款账户名称
		PayerAcctBankName string       `db:"payer_acct_bank_name"` // 付款账户银行名称
		PayeeAcct         string       `db:"payee_acct"`           // 收款账户
		PayeeAcctName     string       `db:"payee_acct_name"`      // 收款账户名称
		PayeeAcctBankName string       `db:"payee_acct_bank_name"` // 收款账户银行名称
		QrCode            string       `db:"qr_code"`              // 生成二维码链接
		ExpireTime        int64        `db:"expire_time"`          // 订单失效时间,单位秒
	}
)

func newOrderTxnModel(conn sqlx.SqlConn) *defaultOrderTxnModel {
	return &defaultOrderTxnModel{
		conn:  conn,
		table: "`order_txn`",
	}
}

func (m *defaultOrderTxnModel) Delete(ctx context.Context, session sqlx.Session, id int64) error {
	query := fmt.Sprintf("delete from %s where `id` = ?", m.table)
	if session != nil {
		_, err := session.ExecCtx(ctx, query, id)
		return err
	}
	_, err := m.conn.ExecCtx(ctx, query, id)
	return err
}
func (m *defaultOrderTxnModel) FindOne(ctx context.Context, id int64) (*OrderTxn, error) {
	query := fmt.Sprintf("select %s from %s where `id` = ? and del_state = ? limit 1", orderTxnRows, m.table)
	var resp OrderTxn
	err := m.conn.QueryRowCtx(ctx, &resp, query, id, 0)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *defaultOrderTxnModel) FindOneByMchIdMchOrderNo(ctx context.Context, mchId string, mchOrderNo string) (*OrderTxn, error) {
	var resp OrderTxn
	query := fmt.Sprintf("select %s from %s where `mch_id` = ? and `mch_order_no` = ?  and del_state = ? limit 1", orderTxnRows, m.table)
	err := m.conn.QueryRowCtx(ctx, &resp, query, mchId, mchOrderNo, 0)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *defaultOrderTxnModel) FindOneByTxnId(ctx context.Context, txnId string) (*OrderTxn, error) {
	var resp OrderTxn
	query := fmt.Sprintf("select %s from %s where `txn_id` = ?  and del_state = ? limit 1", orderTxnRows, m.table)
	err := m.conn.QueryRowCtx(ctx, &resp, query, txnId, 0)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *defaultOrderTxnModel) Insert(ctx context.Context, session sqlx.Session, data *OrderTxn) (sql.Result, error) {
	data.DeleteTime = time.Unix(0, 0)
	data.DelState = 0

	query := fmt.Sprintf("insert into %s (%s) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", m.table, orderTxnRowsExpectAutoSet)
	if session != nil {
		return session.ExecCtx(ctx, query, data.DeleteTime, data.DelState, data.Version, data.TxnId, data.OriTxnId, data.TxnTime, data.TxnDate, data.MchId, data.MchOrderNo, data.PayType, data.TxnType, data.TxnChannel, data.TxnAmt, data.RealAmt, data.Result, data.Body, data.Extra, data.UserId, data.ChannelUser, data.ChannelPayTime, data.ChannelOrderNo, data.PayerAcct, data.PayerAcctName, data.PayerAcctBankName, data.PayeeAcct, data.PayeeAcctName, data.PayeeAcctBankName, data.QrCode, data.ExpireTime)
	}
	return m.conn.ExecCtx(ctx, query, data.DeleteTime, data.DelState, data.Version, data.TxnId, data.OriTxnId, data.TxnTime, data.TxnDate, data.MchId, data.MchOrderNo, data.PayType, data.TxnType, data.TxnChannel, data.TxnAmt, data.RealAmt, data.Result, data.Body, data.Extra, data.UserId, data.ChannelUser, data.ChannelPayTime, data.ChannelOrderNo, data.PayerAcct, data.PayerAcctName, data.PayerAcctBankName, data.PayeeAcct, data.PayeeAcctName, data.PayeeAcctBankName, data.QrCode, data.ExpireTime)
}

func (m *defaultOrderTxnModel) Update(ctx context.Context, session sqlx.Session, newData *OrderTxn) (sql.Result, error) {
	newData.DeleteTime = time.Unix(0, 0)
	newData.DelState = 0
	query := fmt.Sprintf("update %s set %s where `id` = ?", m.table, orderTxnRowsWithPlaceHolder)
	if session != nil {
		return session.ExecCtx(ctx, query, newData.DeleteTime, newData.DelState, newData.Version, newData.TxnId, newData.OriTxnId, newData.TxnTime, newData.TxnDate, newData.MchId, newData.MchOrderNo, newData.PayType, newData.TxnType, newData.TxnChannel, newData.TxnAmt, newData.RealAmt, newData.Result, newData.Body, newData.Extra, newData.UserId, newData.ChannelUser, newData.ChannelPayTime, newData.ChannelOrderNo, newData.PayerAcct, newData.PayerAcctName, newData.PayerAcctBankName, newData.PayeeAcct, newData.PayeeAcctName, newData.PayeeAcctBankName, newData.QrCode, newData.ExpireTime, newData.Id)
	}
	return m.conn.ExecCtx(ctx, query, newData.DeleteTime, newData.DelState, newData.Version, newData.TxnId, newData.OriTxnId, newData.TxnTime, newData.TxnDate, newData.MchId, newData.MchOrderNo, newData.PayType, newData.TxnType, newData.TxnChannel, newData.TxnAmt, newData.RealAmt, newData.Result, newData.Body, newData.Extra, newData.UserId, newData.ChannelUser, newData.ChannelPayTime, newData.ChannelOrderNo, newData.PayerAcct, newData.PayerAcctName, newData.PayerAcctBankName, newData.PayeeAcct, newData.PayeeAcctName, newData.PayeeAcctBankName, newData.QrCode, newData.ExpireTime, newData.Id)
}

func (m *defaultOrderTxnModel) UpdateWithVersion(ctx context.Context, session sqlx.Session, newData *OrderTxn) error {

	oldVersion := newData.Version
	newData.Version += 1

	var sqlResult sql.Result
	var err error

	query := fmt.Sprintf("update %s set %s where `id` = ? and version = ? ", m.table, orderTxnRowsWithPlaceHolder)
	if session != nil {
		sqlResult, err = session.ExecCtx(ctx, query, newData.DeleteTime, newData.DelState, newData.Version, newData.TxnId, newData.OriTxnId, newData.TxnTime, newData.TxnDate, newData.MchId, newData.MchOrderNo, newData.PayType, newData.TxnType, newData.TxnChannel, newData.TxnAmt, newData.RealAmt, newData.Result, newData.Body, newData.Extra, newData.UserId, newData.ChannelUser, newData.ChannelPayTime, newData.ChannelOrderNo, newData.PayerAcct, newData.PayerAcctName, newData.PayerAcctBankName, newData.PayeeAcct, newData.PayeeAcctName, newData.PayeeAcctBankName, newData.QrCode, newData.ExpireTime, newData.Id, oldVersion)
	} else {
		sqlResult, err = m.conn.ExecCtx(ctx, query, newData.DeleteTime, newData.DelState, newData.Version, newData.TxnId, newData.OriTxnId, newData.TxnTime, newData.TxnDate, newData.MchId, newData.MchOrderNo, newData.PayType, newData.TxnType, newData.TxnChannel, newData.TxnAmt, newData.RealAmt, newData.Result, newData.Body, newData.Extra, newData.UserId, newData.ChannelUser, newData.ChannelPayTime, newData.ChannelOrderNo, newData.PayerAcct, newData.PayerAcctName, newData.PayerAcctBankName, newData.PayeeAcct, newData.PayeeAcctName, newData.PayeeAcctBankName, newData.QrCode, newData.ExpireTime, newData.Id, oldVersion)
	}

	if err != nil {
		return err
	}
	updateCount, err := sqlResult.RowsAffected()
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrNoRowsUpdate
	}

	return nil
}

func (m *defaultOrderTxnModel) DeleteSoft(ctx context.Context, session sqlx.Session, data *OrderTxn) error {
	data.DelState = 1
	data.DeleteTime = time.Now()
	if err := m.UpdateWithVersion(ctx, session, data); err != nil {
		return errors.Wrapf(errors.New("delete soft failed "), "OrderTxnModel delete err : %+v", err)
	}
	return nil
}

func (m *defaultOrderTxnModel) FindSum(ctx context.Context, builder squirrel.SelectBuilder, field string) (float64, error) {

	if len(field) == 0 {
		return 0, errors.Wrapf(errors.New("FindSum Least One Field"), "FindSum Least One Field")
	}

	builder = builder.Columns("IFNULL(SUM(" + field + "),0)")

	query, values, err := builder.Where("del_state = ?", 0).ToSql()
	if err != nil {
		return 0, err
	}

	var resp float64

	err = m.conn.QueryRowCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, nil
	default:
		return 0, err
	}
}

func (m *defaultOrderTxnModel) FindCount(ctx context.Context, builder squirrel.SelectBuilder, field string) (int64, error) {

	if len(field) == 0 {
		return 0, errors.Wrapf(errors.New("FindCount Least One Field"), "FindCount Least One Field")
	}

	builder = builder.Columns("COUNT(" + field + ")")

	query, values, err := builder.Where("del_state = ?", 0).ToSql()
	if err != nil {
		return 0, err
	}

	var resp int64

	err = m.conn.QueryRowCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, nil
	default:
		return 0, err
	}
}

func (m *defaultOrderTxnModel) FindAll(ctx context.Context, builder squirrel.SelectBuilder, orderBy string) ([]*OrderTxn, error) {

	builder = builder.Columns(orderTxnRows)

	if orderBy == "" {
		builder = builder.OrderBy("id DESC")
	} else {
		builder = builder.OrderBy(orderBy)
	}

	query, values, err := builder.Where("del_state = ?", 0).ToSql()
	if err != nil {
		return nil, err
	}

	var resp []*OrderTxn

	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *defaultOrderTxnModel) FindPageListByPage(ctx context.Context, builder squirrel.SelectBuilder, page, pageSize int64, orderBy string) ([]*OrderTxn, error) {

	builder = builder.Columns(orderTxnRows)

	if orderBy == "" {
		builder = builder.OrderBy("id DESC")
	} else {
		builder = builder.OrderBy(orderBy)
	}

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	query, values, err := builder.Where("del_state = ?", 0).Offset(uint64(offset)).Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, err
	}

	var resp []*OrderTxn

	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *defaultOrderTxnModel) FindPageListByPageWithTotal(ctx context.Context, builder squirrel.SelectBuilder, page, pageSize int64, orderBy string) ([]*OrderTxn, int64, error) {

	total, err := m.FindCount(ctx, builder, "id")
	if err != nil {
		return nil, 0, err
	}

	builder = builder.Columns(orderTxnRows)

	if orderBy == "" {
		builder = builder.OrderBy("id DESC")
	} else {
		builder = builder.OrderBy(orderBy)
	}

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	query, values, err := builder.Where("del_state = ?", 0).Offset(uint64(offset)).Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, total, err
	}

	var resp []*OrderTxn

	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, total, nil
	default:
		return nil, total, err
	}
}

func (m *defaultOrderTxnModel) FindPageListByIdDESC(ctx context.Context, builder squirrel.SelectBuilder, preMinId, pageSize int64) ([]*OrderTxn, error) {

	builder = builder.Columns(orderTxnRows)

	if preMinId > 0 {
		builder = builder.Where(" id < ? ", preMinId)
	}

	query, values, err := builder.Where("del_state = ?", 0).OrderBy("id DESC").Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, err
	}

	var resp []*OrderTxn

	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *defaultOrderTxnModel) FindPageListByIdASC(ctx context.Context, builder squirrel.SelectBuilder, preMaxId, pageSize int64) ([]*OrderTxn, error) {

	builder = builder.Columns(orderTxnRows)

	if preMaxId > 0 {
		builder = builder.Where(" id > ? ", preMaxId)
	}

	query, values, err := builder.Where("del_state = ?", 0).OrderBy("id ASC").Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, err
	}

	var resp []*OrderTxn

	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *defaultOrderTxnModel) Trans(ctx context.Context, fn func(ctx context.Context, session sqlx.Session) error) error {

	return m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return fn(ctx, session)
	})

}

func (m *defaultOrderTxnModel) SelectBuilder() squirrel.SelectBuilder {
	return squirrel.Select().From(m.table)
}

func (m *defaultOrderTxnModel) tableName() string {
	return m.table
}