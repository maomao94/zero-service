
func (m *default{{.upperStartCamelObject}}Model) FindOneBy{{.upperField}}(ctx context.Context, {{.in}}) (*{{.upperStartCamelObject}}, error) {
{{if .withCache}}{{.cacheKey}}
	var resp {{.upperStartCamelObject}}
	err := m.QueryRowIndexCtx(ctx, &resp, {{.cacheKeyVariable}}, m.formatPrimary, func(ctx context.Context, conn sqlx.SqlConn, v any) (i any, e error) {
		selectBuilder := m.SelectBuilder().Columns(m.rows).
			Where(adaptSQLPlaceholders("{{.originalField}}", m.dbType), {{.lowerStartCamelField}}).
			Where("del_state = 0").
			Limit(1)
		query, args, err := selectBuilder.ToSql()
		if err != nil {
			return nil, err
		}
		if err := conn.QueryRowCtx(ctx, &resp, query, args...); err != nil {
			return nil, err
		}
		return resp.{{.upperStartCamelPrimaryKey}}, nil
	}, m.queryPrimary)
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}{{else}}selectBuilder := m.SelectBuilder().Columns(m.rows).
		Where(adaptSQLPlaceholders("{{.originalField}}", m.dbType), {{.lowerStartCamelField}}).
		Where("del_state = 0").
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
	}
}{{end}}

