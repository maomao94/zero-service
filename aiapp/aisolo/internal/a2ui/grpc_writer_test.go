package a2ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGRPCStreamWriter(t *testing.T) {
	t.Run("create writer", func(t *testing.T) {
		writer := NewGRPCStreamWriter(nil, "test-session")
		assert.NotNil(t, writer)
		assert.Equal(t, "test-session", writer.sessionID)
	})
}

func TestGRPCStreamWriter_Write(t *testing.T) {
	t.Run("write to nil stream", func(t *testing.T) {
		writer := NewGRPCStreamWriter(nil, "test-session")
		n, err := writer.Write([]byte("test data"))
		assert.Error(t, err)
		assert.Equal(t, 0, n)
	})
}

func TestGRPCStreamWriter_SessionID(t *testing.T) {
	writer := NewGRPCStreamWriter(nil, "my-session")
	assert.Equal(t, "my-session", writer.SessionID())
}
