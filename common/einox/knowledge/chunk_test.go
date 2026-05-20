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

func TestSplitIntoDocumentsWithOverlap(t *testing.T) {
	docs := SplitIntoDocumentsWithOverlap("abcdefghijklmnopqrstuvwxyz", 10, 3)
	got := documentContents(docs)
	want := []string{"abcdefghij", "hijklmnopq", "opqrstuvwx", "vwxyz"}
	if len(got) != len(want) {
		t.Fatalf("chunks = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chunks = %#v, want %#v", got, want)
		}
	}
}

func TestSplitIntoDocumentsCapsLargeOverlap(t *testing.T) {
	docs := SplitIntoDocumentsWithOverlap("abcdef", 3, 99)
	got := documentContents(docs)
	want := []string{"abc", "bcd", "cde", "def"}
	if len(got) != len(want) {
		t.Fatalf("chunks = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chunks = %#v, want %#v", got, want)
		}
	}
}

func documentContents(docs []*schema.Document) []string {
	out := make([]string, 0, len(docs))
	for _, doc := range docs {
		out = append(out, doc.Content)
	}
	return out
}
