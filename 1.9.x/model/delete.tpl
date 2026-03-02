func (m *default{{.upperStartCamelObject}}Model) Delete(ctx context.Context, session sqlx.Session, {{.lowerStartCamelPrimaryKey}} {{.dataType}}) error {
	{{if .withCache}}{{if .containsIndexCache}}data, err := m.FindOne(ctx, {{.lowerStartCamelPrimaryKey}})
	if err != nil {
		return err
	}

	{{end}}{{.keys}}
	_, err {{if .containsIndexCache}}={{else}}:={{end}} m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		deleteBuilder := m.DeleteBuilder().Where("id = ?", {{.lowerStartCamelPrimaryKey}})
		query, args, err := deleteBuilder.ToSql()
		if err != nil {
			return nil, err
		}
		if session != nil {
			return session.ExecCtx(ctx, query, args...)
		}
		return conn.ExecCtx(ctx, query, args...)
	}, {{.keyValues}}){{else}}deleteBuilder := m.DeleteBuilder().Where("id = ?", {{.lowerStartCamelPrimaryKey}})
	query, args, err := deleteBuilder.ToSql()
	if err != nil {
		return err
	}
	var execErr error
	if session != nil {
		_, execErr = session.ExecCtx(ctx, query, args...)
	} else {
		_, execErr = m.conn.ExecCtx(ctx, query, args...)
	}
	return execErr{{end}}
}