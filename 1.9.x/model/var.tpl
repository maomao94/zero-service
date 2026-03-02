var (
{{if .withCache}}{{.cacheKeys}}{{end}}
	_ {{.lowerStartCamelObject}}Model = (*default{{.upperStartCamelObject}}Model)(nil)
)
