func new{{.upperStartCamelObject}}Model(conn sqlx.SqlConn{{if .withCache}}, c cache.CacheConf, opts ...cache.Option{{end}}, mopts ...ModelOption) *default{{.upperStartCamelObject}}Model {
	o := applyModelOptions(mopts)
	tableName := {{.table}}
	fieldNames := builder.RawFieldNames(&{{.upperStartCamelObject}}{}, true)
	rows := strings.Join(fieldNames, ",")
	return &default{{.upperStartCamelObject}}Model{
		{{if .withCache}}CachedConn: sqlc.NewConn(conn, c, opts...){{else}}conn: conn{{end}},
		table:      tableName,
		dbType:     o.dbType,
		rows: rows,
	}
}

