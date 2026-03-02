func (m *default{{.upperStartCamelObject}}Model) formatPrimary(primary any) string {
	return fmt.Sprintf("%s%v", {{.primaryKeyLeft}}, primary)
}
func (m *default{{.upperStartCamelObject}}Model) queryPrimary(ctx context.Context, conn sqlx.SqlConn, v, primary any) error {
	selectBuilder := m.SelectBuilder().Columns(m.rows).
		Where(adaptSQLPlaceholders("{{.originalPrimaryField}} = $1", m.dbType), primary).
		Where("del_state = 0").
		Limit(1)
	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return err
	}
	return conn.QueryRowCtx(ctx, v, query, args...)
}
