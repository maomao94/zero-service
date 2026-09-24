package gormx

import (
	"strings"
	"testing"

	"gorm.io/driver/postgres"
)

func TestParseDatabaseTypeDetectsH3(t *testing.T) {
	cases := []struct {
		dsn  string
		want DatabaseType
	}{
		{"h3://user:pass@localhost:5432/db", DatabaseH3},
		{"H3://user:pass@localhost:5432/db?sslmode=disable", DatabaseH3},
	}
	for _, tc := range cases {
		if got := ParseDatabaseType(tc.dsn); got != tc.want {
			t.Fatalf("ParseDatabaseType(%q) = %s, want %s", tc.dsn, got, tc.want)
		}
	}
}

func TestGetDialectorReturnsH3(t *testing.T) {
	d, err := GetDialector(DatabaseH3, "h3://user:pass@localhost:5432/db?sslmode=disable")
	if err != nil {
		t.Fatalf("get dialector error = %v", err)
	}
	pg, ok := d.(*postgres.Dialector)
	if !ok {
		t.Fatalf("dialector type = %T, want *postgres.Dialector", d)
	}
	if pg.DSN == "" || strings.HasPrefix(pg.DSN, "h3://") {
		t.Fatalf("dialector dsn = %q, want converted key-value dsn", pg.DSN)
	}
}

func TestH3URLToKVDSN(t *testing.T) {
	cases := []struct {
		dsn  string
		want string
	}{
		{
			"h3://user:pass@localhost:5432/db?sslmode=disable",
			"user=user password=pass host=localhost port=5432 dbname=db sslmode=disable",
		},
		{
			"h3://user@host/db",
			"user=user host=host dbname=db",
		},
	}
	for _, tc := range cases {
		got, err := pgCompatURLToKVDSN(tc.dsn, DatabaseH3)
		if err != nil {
			t.Fatalf("pgCompatURLToKVDSN(%q, h3) error = %v", tc.dsn, err)
		}
		if got != tc.want {
			t.Fatalf("pgCompatURLToKVDSN(%q, h3) = %q, want %q", tc.dsn, got, tc.want)
		}
	}
}

func TestH3URLToKVDSNPassesThroughKVDSN(t *testing.T) {
	dsn := "host=localhost user=user port=5432 dbname=db sslmode=disable"
	got, err := pgCompatURLToKVDSN(dsn, DatabaseH3)
	if err != nil {
		t.Fatalf("pgCompatURLToKVDSN error = %v", err)
	}
	if got != dsn {
		t.Fatalf("pgCompatURLToKVDSN(%q, h3) = %q, want passthrough", dsn, got)
	}
}
