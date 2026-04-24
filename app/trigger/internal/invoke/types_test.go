package invoke

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestRawProtoToJSON(t *testing.T) {
	var buf []byte
	buf = protowire.AppendTag(buf, 1, protowire.BytesType)
	buf = protowire.AppendString(buf, "hello")
	buf = protowire.AppendTag(buf, 2, protowire.VarintType)
	buf = protowire.AppendVarint(buf, 42)

	var nested []byte
	nested = protowire.AppendTag(nested, 1, protowire.BytesType)
	nested = protowire.AppendString(nested, "world")
	nested = protowire.AppendTag(nested, 2, protowire.VarintType)
	nested = protowire.AppendVarint(nested, 100)
	buf = protowire.AppendTag(buf, 3, protowire.BytesType)
	buf = protowire.AppendBytes(buf, nested)

	result := RawProtoToJSON(buf)
	t.Logf("RawProtoToJSON output: %s", result)

	if result == "" || result == "{}" {
		t.Errorf("expected non-empty JSON, got %q", result)
	}
}

func TestRawProtoToJSON_Empty(t *testing.T) {
	result := RawProtoToJSON(nil)
	t.Logf("RawProtoToJSON(nil) output: %s", result)

	if result != "{}" {
		t.Errorf("expected {}, got %q", result)
	}
}
