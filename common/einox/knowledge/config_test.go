package knowledge

import "testing"

func TestEffectiveBackend(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "memory"},
		{"MEMORY", "memory"},
		{"sqlite", "gorm"},
		{"gorm", "gorm"},
		{"redis", "redis"},
		{"milvus", "milvus"},
		{"unknown", "memory"},
	}
	for _, tc := range cases {
		c := Config{Backend: tc.in}
		if g := c.EffectiveBackend(); g != tc.want {
			t.Errorf("Backend %q: got %q want %q", tc.in, g, tc.want)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := (Config{Enabled: false, Backend: "redis"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Config{Enabled: true, Backend: "redis", Redis: RedisConfig{Addr: "127.0.0.1:6379"}}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Config{Enabled: true, Backend: "redis"}).Validate(); err == nil {
		t.Fatal("expected error without Redis.addr")
	}
	if err := (Config{Enabled: true, Backend: "milvus", Milvus: MilvusConfig{Addr: "h:19530", VectorDim: 128}}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (Config{Enabled: true, Backend: "milvus", Milvus: MilvusConfig{Addr: "h:19530"}}).Validate(); err == nil {
		t.Fatal("expected error without vectorDim")
	}
	mismatch := Config{Enabled: true, Backend: "milvus", Milvus: MilvusConfig{Addr: "h:19530", VectorDim: 128}}
	mismatch.Embedding.ExpectedDim = 256
	if err := mismatch.Validate(); err == nil {
		t.Fatal("expected error when expectedDim != vectorDim")
	}
}
