package gormx

import "testing"

func TestURLToKVDSNKeepsQueryParamOrder(t *testing.T) {
	got, err := urlToKVDSN("kingbase://u:p@h:54321/db?TimeZone=Asia/Shanghai&sslmode=disable", "kingbase")
	if err != nil {
		t.Fatalf("urlToKVDSN error = %v", err)
	}
	want := "user=u password=p host=h port=54321 dbname=db TimeZone=Asia/Shanghai sslmode=disable"
	if got != want {
		t.Fatalf("urlToKVDSN = %q, want %q", got, want)
	}
}

func TestURLToKVDSNFirstParamWinsOnDuplicateKey(t *testing.T) {
	got, err := urlToKVDSN("kingbase://u:p@h:54321/db?sslmode=disable&sslmode=require", "kingbase")
	if err != nil {
		t.Fatalf("urlToKVDSN error = %v", err)
	}
	if want := "user=u password=p host=h port=54321 dbname=db sslmode=disable"; got != want {
		t.Fatalf("urlToKVDSN = %q, want %q", got, want)
	}
}

func TestURLToKVDSNEscapesValues(t *testing.T) {
	got, err := urlToKVDSN("kingbase://u:p%40s%20s@h:54321/db?application_name=my%20app", "kingbase")
	if err != nil {
		t.Fatalf("urlToKVDSN error = %v", err)
	}
	// 密码 URL 解码后为 "p@s s"，应用名解码后为 "my app"，空格需按 libpq 规则转义。
	want := `user=u password=p@s\ s host=h port=54321 dbname=db application_name=my\ app`
	if got != want {
		t.Fatalf("urlToKVDSN = %q, want %q", got, want)
	}
}

func TestURLToKVDSNPassesThroughOtherSchemes(t *testing.T) {
	cases := []string{
		"postgres://u:p@h:5432/db?sslmode=disable",
		"host=h user=u port=5432 dbname=db",
	}
	for _, dsn := range cases {
		got, err := urlToKVDSN(dsn, "kingbase")
		if err != nil {
			t.Fatalf("urlToKVDSN(%q) error = %v", dsn, err)
		}
		if got != dsn {
			t.Fatalf("urlToKVDSN(%q) = %q, want passthrough", dsn, got)
		}
	}
}

func TestURLToKVDSNInvalidURL(t *testing.T) {
	if _, err := urlToKVDSN("kingbase://u:p@h:54321/db?\x7f", "kingbase"); err == nil {
		t.Fatal("urlToKVDSN error = nil, want parse error")
	}
}
