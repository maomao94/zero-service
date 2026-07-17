func (m *default{{.upperStartCamelObject}}Model) FindOne(ctx context.Context, {{.lowerStartCamelPrimaryKey}} {{.dataType}}) (*{{.upperStartCamelObject}}, error) {
	{{if .withCache}}{{.cacheKey}}
	var resp {{.upperStartCamelObject}}
	err := m.QueryRowCtx(ctx, &resp, {{.cacheKeyVariable}}, func(ctx context.Context, conn sqlx.SqlConn, v any) error {
		selectBuilder := m.SelectBuilder().Columns(m.rows).
			Where("id = ?", {{.lowerStartCamelPrimaryKey}}).
			Where("is_deleted = ?", 0).
			Limit(1)
		query, args, err := selectBuilder.ToSql()
		if err != nil {
			return err
		}
		return conn.QueryRowCtx(ctx, v, query, args...)
	})
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}{{else}}selectBuilder := m.SelectBuilder().Columns(m.rows).
		Where("id = ?", {{.lowerStartCamelPrimaryKey}}).
		Where("is_deleted = ?", 0).
		Limit(1)
	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	var resp {{.upperStartCamelObject}}
	err = m.conn.QueryRowCtx(ctx, &resp, query, args...)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}{{end}}
}
