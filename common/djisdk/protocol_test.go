package djisdk

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewServiceRequestFillsTimestampAndFields(t *testing.T) {
	before := time.Now().UnixMilli()
	req := NewServiceRequest("tid-1", "bid-1", MethodFlightTaskPrepare, map[string]any{"flight_id": "f1"})
	after := time.Now().UnixMilli()

	if req.Tid != "tid-1" || req.Bid != "bid-1" || req.Method != MethodFlightTaskPrepare {
		t.Fatalf("request = %+v", req)
	}
	if req.Timestamp < before || req.Timestamp > after {
		t.Fatalf("Timestamp = %d, want in [%d, %d]", req.Timestamp, before, after)
	}
	data, ok := req.Data.(map[string]any)
	if !ok || data["flight_id"] != "f1" {
		t.Fatalf("Data = %+v", req.Data)
	}
}

func TestNewEventReplyFillsTimestampAndResult(t *testing.T) {
	before := time.Now().UnixMilli()
	reply := NewEventReply("tid-2", "bid-2", "some_method", PlatformResultTimeout)
	after := time.Now().UnixMilli()

	if reply.Tid != "tid-2" || reply.Bid != "bid-2" || reply.Method != "some_method" {
		t.Fatalf("reply = %+v", reply)
	}
	if reply.Data.Result != int(PlatformResultTimeout) {
		t.Fatalf("Data.Result = %d, want %d", reply.Data.Result, PlatformResultTimeout)
	}
	if reply.Timestamp < before || reply.Timestamp > after {
		t.Fatalf("Timestamp = %d, want in [%d, %d]", reply.Timestamp, before, after)
	}
}

func TestServiceRequestJSONRoundTrip(t *testing.T) {
	req := NewServiceRequest("tid-1", "bid-1", MethodFlightTaskExecute, map[string]any{"flight_id": "f1"})
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded struct {
		Tid       string          `json:"tid"`
		Bid       string          `json:"bid"`
		Timestamp int64           `json:"timestamp"`
		Method    string          `json:"method"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Tid != "tid-1" || decoded.Bid != "bid-1" || decoded.Method != MethodFlightTaskExecute || decoded.Timestamp == 0 {
		t.Fatalf("decoded = %+v", decoded)
	}
	if string(decoded.Data) == "null" {
		t.Fatal("data must not be null")
	}
}

func TestServiceReplyDataJSONTags(t *testing.T) {
	reply := ServiceReply{Data: ServiceReplyData{Result: 0, Output: map[string]any{"ok": true}}}
	raw, err := json.Marshal(reply)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"result":0`) || !strings.Contains(string(raw), `"output":`) {
		t.Fatalf("raw = %s", raw)
	}
}
