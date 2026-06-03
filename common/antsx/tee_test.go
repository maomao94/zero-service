package antsx

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
)

type failingWriter struct {
	err error
}

func (w failingWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}

func TestTeeWriter_BasicWriteRead(t *testing.T) {
	data := []byte("hello, world")
	tee := NewTeeWriter()
	done := make(chan struct{})

	go func() {
		defer close(done)
		buf := make([]byte, len(data))
		n, err := io.ReadFull(tee.Reader(), buf)
		if err != nil {
			t.Errorf("Read failed: %v", err)
			return
		}
		if n != len(data) {
			t.Errorf("Read %d bytes, want %d", n, len(data))
		}
		if string(buf) != string(data) {
			t.Errorf("Read %q, want %q", string(buf), string(data))
		}
	}()

	n, err := tee.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}
	if err := tee.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	<-done
}

func TestTeeWriter_WithAdditionalWriter(t *testing.T) {
	data := []byte("test data for hash")
	expected := fmt.Sprintf("%x", md5.Sum(data))

	hash := md5.New()
	tee := NewTeeWriter(hash)

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		n, err := tee.Reader().Read(buf)
		if err != nil && err != io.EOF {
			t.Errorf("Read failed: %v", err)
			return
		}
		if n != len(data) {
			t.Errorf("Read %d bytes, want %d", n, len(data))
		}
		tee.Reader().Read(make([]byte, 1))
	}()

	n, err := tee.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}
	tee.Close()
	<-done

	got := fmt.Sprintf("%x", hash.Sum(nil))
	if got != expected {
		t.Errorf("Hash = %q, want %q", got, expected)
	}
}

func TestTeeWriter_WithTmpFile(t *testing.T) {
	data := []byte("file content for tmp test")
	tmpFile, err := os.CreateTemp("", "teewriter_test_*")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	tee := NewTeeWriter(tmpFile)

	done := make(chan struct{})
	go func() {
		defer close(done)
		io.Copy(io.Discard, tee.Reader())
	}()

	if _, err := tee.Write(data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	tee.Close()
	<-done

	tmpFile.Seek(0, io.SeekStart)
	buf := make([]byte, len(data))
	n, err := io.ReadFull(tmpFile, buf)
	if err != nil {
		t.Fatalf("Read tmp file failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Read %d bytes from tmp file, want %d", n, len(data))
	}
	if string(buf) != string(data) {
		t.Errorf("Tmp file content = %q, want %q", string(buf), string(data))
	}
}

func TestTeeWriter_CloseWithError(t *testing.T) {
	tee := NewTeeWriter()

	errChan := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(tee.Reader())
		errChan <- err
	}()

	expectedErr := fmt.Errorf("test error")
	tee.CloseWithError(expectedErr)

	err := <-errChan
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error %v from Reader, got %v", expectedErr, err)
	}
}

func TestTeeWriter_MultipleAdditionalWriters(t *testing.T) {
	data := []byte("data for multi target")
	hash1 := md5.New()
	hash2 := md5.New()
	expected := fmt.Sprintf("%x", md5.Sum(data))

	tee := NewTeeWriter(hash1, hash2)

	done := make(chan struct{})
	go func() {
		defer close(done)
		io.Copy(io.Discard, tee.Reader())
	}()

	if _, err := tee.Write(data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	tee.Close()
	<-done

	got1 := fmt.Sprintf("%x", hash1.Sum(nil))
	got2 := fmt.Sprintf("%x", hash2.Sum(nil))
	if got1 != expected {
		t.Errorf("Hash1 = %q, want %q", got1, expected)
	}
	if got2 != expected {
		t.Errorf("Hash2 = %q, want %q", got2, expected)
	}
}

func TestTeeWriter_LargeData(t *testing.T) {
	data := make([]byte, 5*1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	expected := fmt.Sprintf("%x", md5.Sum(data))

	hash := md5.New()
	tee := NewTeeWriter(hash)

	done := make(chan struct{})
	go func() {
		defer close(done)
		io.Copy(io.Discard, tee.Reader())
	}()

	n, err := tee.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}
	tee.Close()
	<-done

	got := fmt.Sprintf("%x", hash.Sum(nil))
	if got != expected {
		t.Errorf("Hash = %q, want %q", got, expected)
	}
}

func TestTeeWriter_AdditionalWriterError(t *testing.T) {
	expectedErr := errors.New("writer failed")
	tee := NewTeeWriter(failingWriter{err: expectedErr})
	done := make(chan struct{})

	go func() {
		defer close(done)
		io.Copy(io.Discard, tee.Reader())
	}()

	_, err := tee.Write([]byte("data"))
	tee.Close()
	<-done

	if !errors.Is(err, expectedErr) {
		t.Errorf("Write error = %v, want %v", err, expectedErr)
	}
}

func TestTeeWriter_WriteAfterReaderClosed(t *testing.T) {
	tee := NewTeeWriter()
	if err := tee.Reader().Close(); err != nil {
		t.Fatalf("Close reader failed: %v", err)
	}

	_, err := tee.Write([]byte("data"))
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("Write error = %v, want %v", err, io.ErrClosedPipe)
	}
}

func TestTeeWriter_CloseIdempotent(t *testing.T) {
	tee := NewTeeWriter()

	go func() {
		io.Copy(io.Discard, tee.Reader())
	}()

	tee.Close()
	tee.Close()
	tee.Close()
}
