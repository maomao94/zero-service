
func (m *default{{.upperStartCamelObject}}Model) Update(ctx context.Context, session sqlx.Session, {{if .containsIndexCache}}newData{{else}}data{{end}} *{{.upperStartCamelObject}}) (sql.Result, error) {
	{{if .containsIndexCache}}data := newData{{end}}
	data.DeleteTime = sql.NullTime{
		Valid: false,
	}
	data.DelState = 0
	columns, values := generateColumnsAndValues(data, []string{})
	updateBuilder := m.UpdateBuilder()
	for i, column := range columns {
		updateBuilder = updateBuilder.Set(column, values[i])
	}
	updateBuilder = updateBuilder.Where("id = ?", data.Id)
	query, args, err := updateBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	var result sql.Result
	var execErr error
	if session != nil {
		result, execErr = session.ExecCtx(ctx, query, args...)
	} else {
		result, execErr = m.conn.ExecCtx(ctx, query, args...)
	}
	return result, execErr
}

func (m *default{{.upperStartCamelObject}}Model) UpdateWithVersion(ctx context.Context, session sqlx.Session, {{if .containsIndexCache}}newData{{else}}data{{end}} *{{.upperStartCamelObject}}) error {
	{{if .containsIndexCache}}data := newData{{end}}
	oldVersion := data.Version
	data.Version += 1
	data.DeleteTime = sql.NullTime{
		Valid: false,
	}
	data.DelState = 0
	columns, values := generateColumnsAndValues(data, []string{})
	updateBuilder := m.UpdateBuilder()
	for i, column := range columns {
		updateBuilder = updateBuilder.Set(column, values[i])
	}
	updateBuilder = updateBuilder.Where("id = ?", data.Id).Where("version = ?", oldVersion)
	query, args, err := updateBuilder.ToSql()
	if err != nil {
		return err
	}
	var sqlResult sql.Result
	var execErr error
	if session != nil {
		sqlResult, execErr = session.ExecCtx(ctx, query, args...)
	} else {
		sqlResult, execErr = m.conn.ExecCtx(ctx, query, args...)
	}
	if execErr != nil {
		return execErr
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

func (m *default{{.upperStartCamelObject}}Model) DeleteSoft(ctx context.Context, session sqlx.Session, id int64) error {
	data, err := m.FindOne(ctx, id)
	if err != nil {
		return err
	}
	data.DelState = 1
	data.DeleteTime = sql.NullTime{
		Time: time.Now(),
		Valid: true,
	}
	if err := m.UpdateWithVersion(ctx, session, data); err != nil {
		return errors.Wrapf(errors.New("delete soft failed "), "{{.upperStartCamelObject}}Model delete err : %+v", err)
	}
	return nil
}

func (m *default{{.upperStartCamelObject}}Model) FindSum(ctx context.Context, builder squirrel.SelectBuilder, field string) (float64, error) {
	if len(field) == 0 {
		return 0, errors.Wrapf(errors.New("FindSum Least One Field"), "FindSum Least One Field")
	}
	sumFunction := "COALESCE(SUM(" + field + "),0)"
	builder = builder.Columns(sumFunction)
	query, values, err := builder.Where("del_state = ?", 0).ToSql()
	if err != nil {
		return 0, err
	}
	var resp float64
	{{if .withCache}}err = m.QueryRowNoCacheCtx(ctx, &resp, query, values...){{else}}
	err = m.conn.QueryRowCtx(ctx, &resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return 0, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) FindCount(ctx context.Context, builder squirrel.SelectBuilder, field string) (int64, error) {
	if len(field) == 0 {
		return 0, errors.Wrapf(errors.New("FindCount Least One Field"), "FindCount Least One Field")
	}
	builder = builder.Columns("COUNT(" + field + ")")
	query, values, err := builder.Where("del_state = ?", 0).ToSql()
	if err != nil {
		return 0, err
	}
	var resp int64
	{{if .withCache}}err = m.QueryRowNoCacheCtx(ctx, &resp, query, values...){{else}}
	err = m.conn.QueryRowCtx(ctx, &resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return 0, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) FindAll(ctx context.Context, builder squirrel.SelectBuilder, orderBy ...string) ([]*{{.upperStartCamelObject}}, error) {
	builder = builder.Columns(m.rows)
	if len(orderBy) == 0 {
		builder = builder.OrderBy("id DESC")
	} else {
		builder = builder.OrderBy(orderBy...)
	}
	query, values, err := builder.Where("del_state = ?", 0).ToSql()
	if err != nil {
		return nil, err
	}
	var resp []*{{.upperStartCamelObject}}
	{{if .withCache}}err = m.QueryRowsNoCacheCtx(ctx, &resp, query, values...){{else}}
	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) FindPageListByPage(ctx context.Context, builder squirrel.SelectBuilder, page, pageSize int64, orderBy ...string) ([]*{{.upperStartCamelObject}}, error) {
	builder = builder.Columns(m.rows)
	if len(orderBy) == 0 {
		builder = builder.OrderBy("id DESC")
	} else {
		builder = builder.OrderBy(orderBy...)
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	query, values, err := builder.Where("del_state = ?", 0).Offset(uint64(offset)).Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, err
	}
	var resp []*{{.upperStartCamelObject}}
	{{if .withCache}}err = m.QueryRowsNoCacheCtx(ctx, &resp, query, values...){{else}}
	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) FindPageListByPageWithTotal(ctx context.Context, builder squirrel.SelectBuilder, page, pageSize int64, orderBy ...string) ([]*{{.upperStartCamelObject}}, int64, error) {
	total, err := m.FindCount(ctx, builder, "id")
	if err != nil {
		return nil, 0, err
	}
	builder = builder.Columns(m.rows)
	if len(orderBy) == 0 {
		builder = builder.OrderBy("id DESC")
	} else {
		builder = builder.OrderBy(orderBy...)
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	query, values, err := builder.Where("del_state = ?", 0).Offset(uint64(offset)).Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, total, err
	}
	var resp []*{{.upperStartCamelObject}}
	{{if .withCache}}err = m.QueryRowsNoCacheCtx(ctx, &resp, query, values...){{else}}
	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, total, nil
	default:
		return nil, total, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) FindPageListByIdDESC(ctx context.Context, builder squirrel.SelectBuilder, preMinId, pageSize int64) ([]*{{.upperStartCamelObject}}, error) {
	builder = builder.Columns(m.rows)
	if preMinId > 0 {
		builder = builder.Where("id < ?", preMinId)
	}
	query, values, err := builder.Where("del_state = ?", 0).OrderBy("id DESC").Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, err
	}
	var resp []*{{.upperStartCamelObject}}
	{{if .withCache}}err = m.QueryRowsNoCacheCtx(ctx, &resp, query, values...){{else}}
	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) FindPageListByIdASC(ctx context.Context, builder squirrel.SelectBuilder, preMaxId, pageSize int64) ([]*{{.upperStartCamelObject}}, error) {
	builder = builder.Columns(m.rows)
	if preMaxId > 0 {
		builder = builder.Where("id > ?", preMaxId)
	}
	query, values, err := builder.Where("del_state = ?", 0).OrderBy("id ASC").Limit(uint64(pageSize)).ToSql()
	if err != nil {
		return nil, err
	}
	var resp []*{{.upperStartCamelObject}}
	{{if .withCache}}err = m.QueryRowsNoCacheCtx(ctx, &resp, query, values...){{else}}
	err = m.conn.QueryRowsCtx(ctx, &resp, query, values...)
	{{end}}
	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) Trans(ctx context.Context, fn func(ctx context.Context, session sqlx.Session) error) error {
	{{if .withCache}}
	return m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return fn(ctx, session)
	})
	{{else}}
	return m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return fn(ctx, session)
	})
	{{end}}
}

func (m *default{{.upperStartCamelObject}}Model) ExecCtx(ctx context.Context, session sqlx.Session, query string, args ...any) (sql.Result, error) {
	if session != nil {
		return session.ExecCtx(ctx, query, args...)
	}
	return m.conn.ExecCtx(ctx, query, args...)
}

func (m *default{{.upperStartCamelObject}}Model) SelectWithBuilder(ctx context.Context, builder squirrel.SelectBuilder) ([]*{{.upperStartCamelObject}}, error) {
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}
	var resp []*{{.upperStartCamelObject}}
	err = m.conn.QueryRowsPartialCtx(ctx, &resp, query, args...)
	switch err {
	case nil:
		return resp, nil
	default:
		return nil, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) SelectOneWithBuilder(ctx context.Context, builder squirrel.SelectBuilder) (*{{.upperStartCamelObject}}, error) {
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}
	var resp {{.upperStartCamelObject}}
	err = m.conn.QueryRowPartialCtx(ctx, &resp, query, args...)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) InsertWithBuilder(ctx context.Context, session sqlx.Session, builder squirrel.InsertBuilder) (sql.Result, error) {
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}
	return m.ExecCtx(ctx, session, query, args...)
}

func (m *default{{.upperStartCamelObject}}Model) UpdateWithBuilder(ctx context.Context, session sqlx.Session, builder squirrel.UpdateBuilder) (sql.Result, error) {
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}
	return m.ExecCtx(ctx, session, query, args...)
}

func (m *default{{.upperStartCamelObject}}Model) DeleteWithBuilder(ctx context.Context, session sqlx.Session, builder squirrel.DeleteBuilder) (sql.Result, error) {
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}
	return m.ExecCtx(ctx, session, query, args...)
}

func (m *default{{.upperStartCamelObject}}Model) SelectBuilder() squirrel.SelectBuilder {
	builder := squirrel.Select().From(m.table)
	if m.dbType == DatabaseTypePostgres {
		builder = builder.PlaceholderFormat(squirrel.Dollar)
	}
	return builder
}

func (m *default{{.upperStartCamelObject}}Model) UpdateBuilder() squirrel.UpdateBuilder {
	builder := squirrel.Update(m.table)
	if m.dbType == DatabaseTypePostgres {
		builder = builder.PlaceholderFormat(squirrel.Dollar)
	}
	return builder
}

func (m *default{{.upperStartCamelObject}}Model) DeleteBuilder() squirrel.DeleteBuilder {
	builder := squirrel.Delete(m.table)
	if m.dbType == DatabaseTypePostgres {
		builder = builder.PlaceholderFormat(squirrel.Dollar)
	}
	return builder
}

func (m *default{{.upperStartCamelObject}}Model) InsertBuilder() squirrel.InsertBuilder {
	builder := squirrel.Insert(m.table)
	if m.dbType == DatabaseTypePostgres {
		builder = builder.PlaceholderFormat(squirrel.Dollar)
	}
	return builder
}