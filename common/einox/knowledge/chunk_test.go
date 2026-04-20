package knowledge

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestChunkDocumentsToPairs(t *testing.T) {
	docs := []*schema.Document{{Content: "a"}, {Content: "b"}}
	vecs := [][]float32{{1, 2}, {3, 4}}
	pairs, err := chunkDocumentsToPairs(docs, vecs, "pfx")
	if err != nil || len(pairs) != 2 {
		t.Fatalf("pairs=%v err=%v", pairs, err)
	}
	if pairs[0].ChunkID != "pfx-0" || pairs[0].Text != "a" {
		t.Fatalf("first pair %+v", pairs[0])
	}
	_, err = chunkDocumentsToPairs(docs, [][]float32{{1}}, "x")
	if err == nil {
		t.Fatal("expected mismatch len")
	}
}
