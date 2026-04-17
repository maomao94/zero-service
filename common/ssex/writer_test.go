package ssex

import (
	"bytes"
	"net/http"
	"sync"
	"testing"
)

type fakeResponseWriter struct {
	header http.Header
	buf    bytes.Buffer
	mu     sync.Mutex
}

func newFakeResponseWriter() *fakeResponseWriter {
	return &fakeResponseWriter{header: make(http.Header)}
}

func (f *fakeResponseWriter) Header() http.Header {
	return f.header
}

func (f *fakeResponseWriter) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.Write(p)
}

func (f *fakeResponseWriter) WriteHeader(statusCode int) {}

func (f *fakeResponseWriter) Flush() {}

func TestWriter_ConcurrentWrites(t *testing.T) {
	fw := newFakeResponseWriter()
	w, err := NewWriter(fw)
	if err != nil {
		t.Fatalf("new writer failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.WriteKeepAlive()
			w.WriteData("chunk")
		}()
	}
	wg.Wait()

	out := fw.buf.String()
	if out == "" {
		t.Fatal("expected sse output, got empty")
	}
}
