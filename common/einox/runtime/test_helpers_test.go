package runtime

import (
	"encoding/json"
	"testing"

	"zero-service/common/einox/protocol"
)

func decodeData[T any](t *testing.T, event protocol.Event) (T, bool) {
	t.Helper()
	var out T
	if len(event.Data) == 0 {
		return out, false
	}
	if err := json.Unmarshal(event.Data, &out); err != nil {
		t.Fatalf("decode event data: %v", err)
	}
	return out, true
}
