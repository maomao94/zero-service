package knowledge

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func TestRedisStoreCRUDSearch(t *testing.T) {
	srv := miniredis.RunT(t)
	cfg := Config{
		Backend: "redis",
		Redis:   RedisConfig{Addr: srv.Addr()},
	}
	st, err := newRedisStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	const uid, bid = "u1", "b1"
	if err := st.CreateBase(ctx, uid, bid, "kb"); err != nil {
		t.Fatal(err)
	}
	v := []float32{1, 0, 0}
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

func TestRedisStoreMaxChunksPerBase(t *testing.T) {
	srv := miniredis.RunT(t)
	cfg := Config{
		Backend:          "redis",
		Redis:            RedisConfig{Addr: srv.Addr()},
		MaxChunksPerBase: 2,
	}
	st, err := newRedisStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	const uid, bid = "u1", "b1"
	if err := st.CreateBase(ctx, uid, bid, "kb"); err != nil {
		t.Fatal(err)
	}
	v := []float32{1, 0, 0}
	if err := st.UpsertChunks(ctx, uid, bid, "s1", "a", []chunkVectorPair{
		{Text: "a1", Vector: v},
		{Text: "a2", Vector: v},
	}); err != nil {
		t.Fatal(err)
	}
	err = st.UpsertChunks(ctx, uid, bid, "s2", "b", []chunkVectorPair{
		{Text: "b1", Vector: v},
	})
	if err == nil {
		t.Fatal("expected over max chunks")
	}
}
