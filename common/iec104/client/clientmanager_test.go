package client

import "testing"

func TestClientManagerGetClientOrNilReturnsNilWhenNotFound(t *testing.T) {
	manager := NewClientManager()

	cli, err := manager.GetClientOrNil("127.0.0.1", 2404)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cli != nil {
		t.Fatalf("expected nil client, got %#v", cli)
	}
}
