// Code generated by goctl. DO NOT EDIT.
// versions:
//  goctl version: 1.7.1

package model

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
	userFieldNames          = builder.RawFieldNames(&User{})
	userRows                = strings.Join(userFieldNames, ",")
	userRowsExpectAutoSet   = strings.Join(stringx.Remove(userFieldNames, "`id`", "`create_at`", "`create_time`", "`created_at`", "`update_at`", "`update_time`", "`updated_at`"), ",")
	userRowsWithPlaceHolder = strings.Join(stringx.Remove(userFieldNames, "`id`", "`create_at`", "`create_time`", "`created_at`", "`update_at`", "`update_time`", "`updated_at`"), "=?,") + "=?"
)

type (
	userModel interface {
		Insert(ctx context.Context, session sqlx.Session, data *User) (sql.Result, error)
		FindOne(ctx context.Context, id int64) (*User, error)
		FindOneByMobile(ctx context.Context, mobile string) (*User, error)
		Update(ctx context.Context, session sqlx.Session, data *User) (sql.Result, error)
		UpdateWithVersion(ctx context.Context, session sqlx.Session, data *User) error
		Trans(ctx context.Context, fn func(context context.Context, session sqlx.Session) error) error
		SelectBuilder() squirrel.SelectBuilder
		DeleteSoft(ctx context.Context, session sqlx.Session, data *User) error
		FindSum(ctx context.Context, sumBuilder squirrel.SelectBuilder, field string) (float64, error)
		FindCount(ctx context.Context, countBuilder squirrel.SelectBuilder, field string) (int64, error)
		FindAll(ctx context.Context, rowBuilder squirrel.SelectBuilder, orderBy string) ([]*User, error)
		FindPageListByPage(ctx context.Context, rowBuilder squirrel.SelectBuilder, page, pageSize int64, orderBy string) ([]*User, error)
		FindPageListByPageWithTotal(ctx context.Context, rowBuilder squirrel.SelectBuilder, page, pageSize int64, orderBy string) ([]*User, int64, error)
		FindPageListByIdDESC(ctx context.Context, rowBuilder squirrel.SelectBuilder, preMinId, pageSize int64) ([]*User, error)
		FindPageListByIdASC(ctx context.Context, rowBuilder squirrel.SelectBuilder, preMaxId, pageSize int64) ([]*User, error)
		Delete(ctx context.Context, session sqlx.Session, id int64) error
	}

	defaultUserModel struct {
		conn  sqlx.SqlConn
		table string
	}

	User struct {
		Id         int64     `db:"id"`
		CreateTime time.Time `db:"create_time"`
		UpdateTime time.Time `db:"update_time"`
		DeleteTime time.Time `db:"delete_time"`
		DelState   int64     `db:"del_state"`
		Version    int64     `db:"version"` // 版本号
		Mobile     string    `db:"mobile"`
		Password   string    `db:"password"`
		Nickname   string    `db:"nickname"`
		Sex        int64     `db:"sex"` // 性别 0:男 1:女
		Avatar     string    `db:"avatar"`
		Info       string    `db:"info"`
	}
)

func newUserModel(conn sqlx.SqlConn) *defaultUserModel {
	return &defaultUserModel{
		conn:  conn,
		table: "`user`",
	}
}

func (m *defaultUserModel) Delete(ctx context.Context, session sqlx.Session, id int64) error {
	query := fmt.Sprintf("delete from %s where `id` = ?", m.table)
	if session != nil {
		_, err := session.ExecCtx(ctx, query, id)
		return err
	}
	_, err := m.conn.ExecCtx(ctx, query, id)
	return err
}
func (m *defaultUserModel) FindOne(ctx context.Context, id int64) (*User, error) {
	query := fmt.Sprintf("select %s from %s where `id` = ? and del_state = ? limit 1", userRows, m.table)
	var resp User
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

func (m *defaultUserModel) FindOneByMobile(ctx context.Context, mobile string) (*User, error) {
	var resp User
	query := fmt.Sprintf("select %s from %s where `mobile` = ?  and del_state = ? limit 1", userRows, m.table)
	err := m.conn.QueryRowCtx(ctx, &resp, query, mobile, 0)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *defaultUserModel) Insert(ctx context.Context, session sqlx.Session, data *User) (sql.Result, error) {
	data.DeleteTime = time.Unix(0, 0)
	data.DelState = 0

	query := fmt.Sprintf("insert into %s (%s) values (?, ?, ?, ?, ?, ?, ?, ?, ?)", m.table, userRowsExpectAutoSet)
	if session != nil {
		return session.ExecCtx(ctx, query, data.DeleteTime, data.DelState, data.Version, data.Mobile, data.Password, data.Nickname, data.Sex, data.Avatar, data.Info)
	}
	return m.conn.ExecCtx(ctx, query, data.DeleteTime, data.DelState, data.Version, data.Mobile, data.Password, data.Nickname, data.Sex, data.Avatar, data.Info)
}

func (m *defaultUserModel) Update(ctx context.Context, session sqlx.Session, newData *User) (sql.Result, error) {
	newData.DeleteTime = time.Unix(0, 0)
	newData.DelState = 0
	query := fmt.Sprintf("update %s set %s where `id` = ?", m.table, userRowsWithPlaceHolder)
	if session != nil {
		return session.ExecCtx(ctx, query, newData.DeleteTime, newData.DelState, newData.Version, newData.Mobile, newData.Password, newData.Nickname, newData.Sex, newData.Avatar, newData.Info, newData.Id)
	}
	return m.conn.ExecCtx(ctx, query, newData.DeleteTime, newData.DelState, newData.Version, newData.Mobile, newData.Password, newData.Nickname, newData.Sex, newData.Avatar, newData.Info, newData.Id)
}

func (m *defaultUserModel) UpdateWithVersion(ctx context.Context, session sqlx.Session, newData *User) error {

	oldVersion := newData.Version
	newData.Version += 1

	var sqlResult sql.Result
	var err error

	query := fmt.Sprintf("update %s set %s where `id` = ? and version = ? ", m.table, userRowsWithPlaceHolder)
	if session != nil {
		sqlResult, err = session.ExecCtx(ctx, query, newData.DeleteTime, newData.DelState, newData.Version, newData.Mobile, newData.Password, newData.Nickname, newData.Sex, newData.Avatar, newData.Info, newData.Id, oldVersion)
	} else {
		sqlResult, err = m.conn.ExecCtx(ctx, query, newData.DeleteTime, newData.DelState, newData.Version, newData.Mobile, newData.Password, newData.Nickname, newData.Sex, newData.Avatar, newData.Info, newData.Id, oldVersion)
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

func (m *defaultUserModel) DeleteSoft(ctx context.Context, session sqlx.Session, data *User) error {
	data.DelState = 1
	data.DeleteTime = time.Now()
	if err := m.UpdateWithVersion(ctx, session, data); err != nil {
		return errors.Wrapf(errors.New("delete soft failed "), "UserModel delete err : %+v", err)
	}
	return nil
}

func (m *defaultUserModel) FindSum(ctx context.Context, builder squirrel.SelectBuilder, field string) (float64, error) {

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

func (m *defaultUserModel) FindCount(ctx context.Context, builder squirrel.SelectBuilder, field string) (int64, error) {

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

func (m *defaultUserModel) FindAll(ctx context.Context, builder squirrel.SelectBuilder, orderBy string) ([]*User, error) {

	builder = builder.Columns(userRows)

	if orderBy == "" {
		builder = builder.OrderBy("id DESC")
	} else {
		builder = builder.OrderBy(orderBy)
	}

	query, values, err := builder.Where("del_state = ?", 0).ToSql()
	if err != nil {
		return nil, err
	}

	var resp []*User

	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *defaultUserModel) FindPageListByPage(ctx context.Context, builder squirrel.SelectBuilder, page, pageSize int64, orderBy string) ([]*User, error) {

	builder = builder.Columns(userRows)

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

	var resp []*User

	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *defaultUserModel) FindPageListByPageWithTotal(ctx context.Context, builder squirrel.SelectBuilder, page, pageSize int64, orderBy string) ([]*User, int64, error) {

	total, err := m.FindCount(ctx, builder, "id")
	if err != nil {
		return nil, 0, err
	}

	builder = builder.Columns(userRows)

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

	var resp []*User

	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, total, nil
	default:
		return nil, total, err
	}
}

func (m *defaultUserModel) FindPageListByIdDESC(ctx context.Context, builder squirrel.SelectBuilder, preMinId, pageSize int64) ([]*User, error) {

	builder = builder.Columns(userRows)

	if preMinId > 0 {
		builder = builder.Where(" id < ? ", preMinId)
	}

	query, values, err := builder.Where("del_state = ?", 0).OrderBy("id DESC").Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, err
	}

	var resp []*User

	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *defaultUserModel) FindPageListByIdASC(ctx context.Context, builder squirrel.SelectBuilder, preMaxId, pageSize int64) ([]*User, error) {

	builder = builder.Columns(userRows)

	if preMaxId > 0 {
		builder = builder.Where(" id > ? ", preMaxId)
	}

	query, values, err := builder.Where("del_state = ?", 0).OrderBy("id ASC").Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, err
	}

	var resp []*User

	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)

	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *defaultUserModel) Trans(ctx context.Context, fn func(ctx context.Context, session sqlx.Session) error) error {

	return m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return fn(ctx, session)
	})

}

func (m *defaultUserModel) SelectBuilder() squirrel.SelectBuilder {
	return squirrel.Select().From(m.table)
}

func (m *defaultUserModel) tableName() string {
	return m.table
}
