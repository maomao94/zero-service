func new{{.upperStartCamelObject}}Model(conn sqlx.SqlConn{{if .withCache}}, c cache.CacheConf, opts ...cache.Option{{end}}) *default{{.upperStartCamelObject}}Model {
	return new{{.upperStartCamelObject}}ModelWithDBType(conn, DatabaseTypeMySQL{{if .withCache}}, c, opts...{{end}})
}

func new{{.upperStartCamelObject}}ModelWithDBType(conn sqlx.SqlConn, dbType DatabaseType{{if .withCache}}, c cache.CacheConf, opts ...cache.Option{{end}}) *default{{.upperStartCamelObject}}Model {
	tableName := {{.table}}
	fieldNames := builder.RawFieldNames(&{{.upperStartCamelObject}}{}, true)
	rows := strings.Join(fieldNames, ",")
	return &default{{.upperStartCamelObject}}Model{
		{{if .withCache}}CachedConn: sqlc.NewConn(conn, c, opts...){{else}}conn: conn{{end}},
		table:      tableName,
		dbType:     dbType,
		rows: rows,
	}
}

