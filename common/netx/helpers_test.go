package netx

import (
	"context"
	"io"
	"testing"
)

func ctx(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}

type blockingReader struct {
	data  []byte
	reads int
}

func (r *blockingReader) Read(p []byte) (int, error) {
	r.reads++
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

type errorReader struct {
	reads int
}

func (e *errorReader) Read(_ []byte) (int, error) {
	e.reads++
	return 0, io.ErrUnexpectedEOF
}

type errorAfterReader struct {
	sent bool
}

func (r *errorAfterReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		return copy(p, "partial"), nil
	}
	return 0, io.ErrUnexpectedEOF
}
