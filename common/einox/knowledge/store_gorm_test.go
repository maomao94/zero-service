package knowledge

import (
	"context"
	"testing"
)

func TestGormStoreCRUDSearch(t *testing.T) {
	cfg := Config{
		Backend: "gorm",
		DataDir: t.TempDir(),
	}
	st, err := newGORMStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	const uid, bid = "u1", "b1"
	if err := st.CreateBase(ctx, uid, bid, "kb"); err != nil {
		t.Fatal(err)
	}
	v := []float32{0, 1, 0}
	if err := st.UpsertChunks(ctx, uid, bid, "s1", "f.txt", []chunkVectorPair{
		{ChunkID: "c1", Text: "hello", Vector: v},
	}); err != nil {
		t.Fatal(err)
	}
	hits, err := st.Search(ctx, uid, bid, v, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Text != "hello" {
		t.Fatalf("hits=%+v", hits)
	}
}

func TestGormStoreSQLiteMemoryDSN(t *testing.T) {
	cfg := Config{
		Backend: "gorm",
		DSN:     "file::memory:?cache=shared",
	}
	st, err := newGORMStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	if err := st.CreateBase(ctx, "u-mem", "b-mem", "kb"); err != nil {
		t.Fatal(err)
	}
	v := []float32{1, 1, 0}
	if err := st.UpsertChunks(ctx, "u-mem", "b-mem", "s1", "x.txt", []chunkVectorPair{
		{Text: "mem", Vector: v},
	}); err != nil {
		t.Fatal(err)
	}
	hits, err := st.Search(ctx, "u-mem", "b-mem", v, 3)
	if err != nil || len(hits) != 1 {
		t.Fatalf("err=%v hits=%+v", err, hits)
	}
}
