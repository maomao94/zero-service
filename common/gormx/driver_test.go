package gormx

import (
	"strings"
	"testing"

	dameng "github.com/godoes/gorm-dameng"
	kingbase "github.com/godoes/gorm-kingbase"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestParseDatabaseTypeDetectsSQLite(t *testing.T) {
	cases := []struct {
		dsn  string
		want DatabaseType
	}{
		{"file:test.db?mode=memory", DatabaseSQLite},
		{"sqlite://test.db", DatabaseSQLite},
		{"sqlite3://test.db", DatabaseSQLite},
		{":memory:", DatabaseSQLite},
	}
	for _, tc := range cases {
		if got := ParseDatabaseType(tc.dsn); got != tc.want {
			t.Fatalf("ParseDatabaseType(%q) = %s, want %s", tc.dsn, got, tc.want)
		}
	}
}

func TestParseDatabaseTypeDetectsPostgres(t *testing.T) {
	cases := []struct {
		dsn  string
		want DatabaseType
	}{
		{"postgres://user:pass@localhost/db", DatabasePostgres},
		{"postgresql://user:pass@localhost/db", DatabasePostgres},
	}
	for _, tc := range cases {
		if got := ParseDatabaseType(tc.dsn); got != tc.want {
			t.Fatalf("ParseDatabaseType(%q) = %s, want %s", tc.dsn, got, tc.want)
		}
	}
}

func TestParseDatabaseTypeDetectsMySQL(t *testing.T) {
	cases := []struct {
		dsn  string
		want DatabaseType
	}{
		{"mysql://user:pass@tcp(localhost:3306)/db", DatabaseMySQL},
		{"", DatabaseMySQL},
	}
	for _, tc := range cases {
		if got := ParseDatabaseType(tc.dsn); got != tc.want {
			t.Fatalf("ParseDatabaseType(%q) = %s, want %s", tc.dsn, got, tc.want)
		}
	}
}

func TestParseDatabaseTypeDetectsDM(t *testing.T) {
	cases := []struct {
		dsn  string
		want DatabaseType
	}{
		{"dm://SYSDBA:SYSDBA@localhost:5236?schema=SYSDBA", DatabaseDM},
		{"DM://SYSDBA:SYSDBA@localhost:5236", DatabaseDM},
	}
	for _, tc := range cases {
		if got := ParseDatabaseType(tc.dsn); got != tc.want {
			t.Fatalf("ParseDatabaseType(%q) = %s, want %s", tc.dsn, got, tc.want)
		}
	}
}

func TestParseDatabaseTypeDetectsKingbase(t *testing.T) {
	cases := []struct {
		dsn  string
		want DatabaseType
	}{
		{"kingbase://SYSTEM:pass@localhost:54321/TEST?sslmode=disable", DatabaseKingbase},
		{"KINGBASE://SYSTEM:pass@localhost:54321/TEST", DatabaseKingbase},
	}
	for _, tc := range cases {
		if got := ParseDatabaseType(tc.dsn); got != tc.want {
			t.Fatalf("ParseDatabaseType(%q) = %s, want %s", tc.dsn, got, tc.want)
		}
	}
}

func TestGetDialectorReturnsDM(t *testing.T) {
	d, err := GetDialector(DatabaseDM, "dm://SYSDBA:SYSDBA@localhost:5236?schema=SYSDBA")
	if err != nil {
		t.Fatalf("get dialector error = %v", err)
	}
	if _, ok := d.(*dameng.Dialector); !ok {
		t.Fatalf("dialector type = %T, want *dameng.Dialector", d)
	}
	if d.Name() != "dm" {
		t.Fatalf("dialector name = %s, want dm", d.Name())
	}
}

func TestGetDialectorReturnsKingbase(t *testing.T) {
	d, err := GetDialector(DatabaseKingbase, "kingbase://SYSTEM:pass@localhost:54321/TEST?sslmode=disable")
	if err != nil {
		t.Fatalf("get dialector error = %v", err)
	}
	kb, ok := d.(*kingbase.Dialector)
	if !ok {
		t.Fatalf("dialector type = %T, want *kingbase.Dialector", d)
	}
	if kb.DriverName != kingbase.DriverName {
		t.Fatalf("dialector driver name = %s, want %s", kb.DriverName, kingbase.DriverName)
	}
	if kb.DSN == "" || strings.HasPrefix(kb.DSN, "kingbase://") {
		t.Fatalf("dialector dsn = %q, want converted key-value dsn", kb.DSN)
	}
	if kb.WithoutQuotingCheck {
		t.Fatalf("dialector without quoting check should default to false")
	}
}

func TestGetDialectorKingbaseWithoutQuotingCheckOption(t *testing.T) {
	d, err := GetDialector(DatabaseKingbase, "kingbase://SYSTEM:pass@localhost:54321/TEST?sslmode=disable&without_quoting_check=true")
	if err != nil {
		t.Fatalf("get dialector error = %v", err)
	}
	kb, ok := d.(*kingbase.Dialector)
	if !ok {
		t.Fatalf("dialector type = %T, want *kingbase.Dialector", d)
	}
	if !kb.WithoutQuotingCheck {
		t.Fatalf("dialector without quoting check should be enabled by dsn option")
	}
	if strings.Contains(kb.DSN, kingbaseOptionWithoutQuotingCheck) {
		t.Fatalf("dialector dsn = %q, gormx option should not leak into driver dsn", kb.DSN)
	}
	if !strings.Contains(kb.DSN, "sslmode=disable") {
		t.Fatalf("dialector dsn = %q, other params should be preserved", kb.DSN)
	}
}

func TestGetDialectorReturnsMySQL(t *testing.T) {
	d, err := GetDialector(DatabaseMySQL, "user:pass@tcp(localhost)/db")
	if err != nil {
		t.Fatalf("get dialector error = %v", err)
	}
	if d == nil {
		t.Fatalf("dialector should not be nil")
	}
}

func TestGetDialectorReturnsPostgres(t *testing.T) {
	d, err := GetDialector(DatabasePostgres, "host=localhost")
	if err != nil {
		t.Fatalf("get dialector error = %v", err)
	}
	if d == nil {
		t.Fatalf("dialector should not be nil")
	}
}

func TestGetDialectorReturnsSQLite(t *testing.T) {
	d, err := GetDialector(DatabaseSQLite, "file:test?mode=memory")
	if err != nil {
		t.Fatalf("get dialector error = %v", err)
	}
	if d == nil {
		t.Fatalf("dialector should not be nil")
	}
}

func TestGetDialectorRejectsUnsupportedType(t *testing.T) {
	_, err := GetDialector(DatabaseType("oracle"), "dsn")
	if err == nil {
		t.Fatalf("expected error for unsupported database type")
	}
}

func TestParseDatabaseTypeRejectsGaussDBScheme(t *testing.T) {
	if got := ParseDatabaseType("gaussdb://user:pass@localhost:8000/db"); got != DatabaseType("gaussdb") {
		t.Fatalf("ParseDatabaseType(gaussdb://...) = %s, want unsupported gaussdb type", got)
	}
}

func TestGetDialectorRejectsGaussDB(t *testing.T) {
	_, err := GetDialector(DatabaseType("gaussdb"), "host=localhost port=8000 user=gorm dbname=gorm sslmode=disable")
	if err == nil {
		t.Fatalf("expected error for disabled gaussdb driver")
	}
}

func TestGaussDBUsesPostgresCompatibleDSN(t *testing.T) {
	d, err := GetDialector(ParseDatabaseType("postgres://user:pass@localhost:8000/db?sslmode=disable&TimeZone=Asia/Shanghai"), "postgres://user:pass@localhost:8000/db?sslmode=disable&TimeZone=Asia/Shanghai")
	if err != nil {
		t.Fatalf("get dialector error = %v", err)
	}
	if _, ok := d.(*postgres.Dialector); !ok {
		t.Fatalf("dialector type = %T, want *postgres.Dialector", d)
	}
}

func TestDatabaseTypeOfUsesDBDialector(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error = %v", err)
	}

	db.Statement = nil
	if got := GetDatabaseTypeFromDialector(db); got != DatabaseSQLite {
		t.Fatalf("database type = %s, want %s", got, DatabaseSQLite)
	}
}
