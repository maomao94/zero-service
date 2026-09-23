package gormx

import "testing"

func TestKingbaseURLToKVDSN(t *testing.T) {
	cases := []struct {
		dsn  string
		want string
	}{
		{
			"kingbase://SYSTEM:pass@localhost:54321/TEST?sslmode=disable",
			"user=SYSTEM password=pass host=localhost port=54321 dbname=TEST sslmode=disable",
		},
		{
			"kingbase://SYSTEM:pass@localhost:54321/TEST",
			"user=SYSTEM password=pass host=localhost port=54321 dbname=TEST",
		},
		{
			"kingbase://user@host/db",
			"user=user host=host dbname=db",
		},
		{
			"kingbase://u:p@h1:54321,h2:54321/db?sslmode=disable",
			"user=u password=p host=h1,h2 port=54321,54321 dbname=db sslmode=disable",
		},
		{
			"kingbase://u:p@[::1]:54321/db",
			"user=u password=p host=::1 port=54321 dbname=db",
		},
	}
	for _, tc := range cases {
		got, err := kingbaseURLToKVDSN(tc.dsn)
		if err != nil {
			t.Fatalf("kingbaseURLToKVDSN(%q) error = %v", tc.dsn, err)
		}
		if got != tc.want {
			t.Fatalf("kingbaseURLToKVDSN(%q) = %q, want %q", tc.dsn, got, tc.want)
		}
	}
}

func TestKingbaseURLToKVDSNPassesThroughKVDSN(t *testing.T) {
	dsn := "host=localhost user=SYSTEM port=54321 dbname=TEST sslmode=disable"
	got, err := kingbaseURLToKVDSN(dsn)
	if err != nil {
		t.Fatalf("kingbaseURLToKVDSN error = %v", err)
	}
	if got != dsn {
		t.Fatalf("kingbaseURLToKVDSN(%q) = %q, want passthrough", dsn, got)
	}
}
