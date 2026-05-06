package ossx

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
