
func (m *default{{.upperStartCamelObject}}Model) Insert(ctx context.Context, session sqlx.Session, data *{{.upperStartCamelObject}}) (sql.Result, error) {
	data.DeleteTime = sql.NullTime{
		Valid: false,
	}
	data.DelState = 0
	columns, values := generateColumnsAndValues(data, []string{})
	insertBuilder := m.InsertBuilder().Columns(columns...).Values(values...)

	if m.dbType == DatabaseTypePostgres {
		insertBuilder = insertBuilder.Suffix("RETURNING id")
		query, args, err := insertBuilder.ToSql()
		if err != nil {
			return nil, err
		}
		var id int64
		var execErr error
		if session != nil {
			execErr = session.QueryRowCtx(ctx, &id, query, args...)
		} else {
			execErr = m.conn.QueryRowCtx(ctx, &id, query, args...)
		}
		if execErr != nil {
			return nil, execErr
		}
		data.Id = id
		return &postgresResult{id: id}, nil
	} else {
		query, args, err := insertBuilder.ToSql()
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
}
