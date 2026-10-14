package filex

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestUploadWithMD5(t *testing.T) {
	reader := bytes.NewReader([]byte("hello world"))
	var written bytes.Buffer

	md5Hex, err := UploadWithMD5(reader, func(r io.Reader) error {
		_, err := io.Copy(&written, r)
		return err
	})
	if err != nil {
		t.Fatalf("UploadWithMD5() error = %v", err)
	}
	if written.String() != "hello world" {
		t.Fatalf("written = %q, want hello world", written.String())
	}
	if md5Hex != "5eb63bbbe01eeed093cb22bb8f5acdc3" {
		t.Fatalf("md5 = %q, want expected hash", md5Hex)
	}
}

func TestUploadWithMD5ReturnsUploadError(t *testing.T) {
	uploadErr := errors.New("upload failed")
	_, err := UploadWithMD5(bytes.NewReader([]byte("x")), func(io.Reader) error {
		return uploadErr
	})
	if !errors.Is(err, uploadErr) {
		t.Fatalf("err = %v, want %v", err, uploadErr)
	}
}

func TestUploadWithMD5Size(t *testing.T) {
	data := []byte("hello world")
	md5Hex, n, err := UploadWithMD5Size(bytes.NewReader(data), func(r io.Reader) error {
		_, err := io.Copy(io.Discard, r)
		return err
	})
	if err != nil {
		t.Fatalf("UploadWithMD5Size() error = %v", err)
	}
	if n != int64(len(data)) {
		t.Fatalf("size = %d, want %d", n, len(data))
	}
	if md5Hex != "5eb63bbbe01eeed093cb22bb8f5acdc3" {
		t.Fatalf("md5 = %q, want expected hash", md5Hex)
	}
}

func TestCountingReader(t *testing.T) {
	c := NewCountingReader(bytes.NewReader([]byte("hello")))
	buf := make([]byte, 2)
	if _, err := io.ReadFull(c, buf); err != nil {
		t.Fatal(err)
	}
	if c.N() != 2 {
		t.Fatalf("N() = %d after 2 bytes, want 2", c.N())
	}
	if _, err := io.ReadFull(c, buf); err != nil {
		t.Fatal(err)
	}
	if c.N() != 4 {
		t.Fatalf("N() = %d after 4 bytes, want 4", c.N())
	}
}
