package invoke

import (
	"context"
	"encoding/json"
	"fmt"
	"zero-service/app/trigger/internal/svc"

	"google.golang.org/protobuf/encoding/protowire"
)

type Task struct {
	ID         string
	Protocol   string
	Timeout    int64
	URL        string
	HTTPMethod string
	Headers    map[string]string
	Body       []byte
	GrpcServer string
	Method     string
	Payload    []byte
}

type Result struct {
	ID            string
	Success       bool
	StatusCode    int32
	Error         string
	Data          []byte
	CostMs        int64
	CostFormatted string
}

func FormatCostMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

type Invoker interface {
	Execute(ctx context.Context, svcCtx *svc.ServiceContext, task *Task) *Result
}

func RawProtoToJSON(data []byte) string {
	fields := rawProtoDecode(data)
	b, err := json.Marshal(fields)
	if err != nil {
		return fmt.Sprintf("<proto decode error: %v>", err)
	}
	return string(b)
}

func rawProtoDecode(data []byte) map[string]any {
	fields := make(map[string]any)
	for len(data) > 0 {
		num, wtype, n := protowire.ConsumeTag(data)
		if n < 0 {
			break
		}
		data = data[n:]
		key := fmt.Sprintf("f%d", num)

		switch wtype {
		case protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fields
			}
			data = data[n:]
			appendField(fields, key, v)

		case protowire.Fixed32Type:
			v, n := protowire.ConsumeFixed32(data)
			if n < 0 {
				return fields
			}
			data = data[n:]
			appendField(fields, key, v)

		case protowire.Fixed64Type:
			v, n := protowire.ConsumeFixed64(data)
			if n < 0 {
				return fields
			}
			data = data[n:]
			appendField(fields, key, v)

		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fields
			}
			data = data[n:]
			if nested := rawProtoDecode(v); len(nested) > 0 && isValidProto(v) {
				appendField(fields, key, nested)
			} else {
				appendField(fields, key, string(v))
			}

		default:
			return fields
		}
	}
	return fields
}

func isValidProto(data []byte) bool {
	for len(data) > 0 {
		_, wtype, n := protowire.ConsumeTag(data)
		if n < 0 {
			return false
		}
		data = data[n:]
		switch wtype {
		case protowire.VarintType:
			_, n = protowire.ConsumeVarint(data)
		case protowire.Fixed32Type:
			_, n = protowire.ConsumeFixed32(data)
		case protowire.Fixed64Type:
			_, n = protowire.ConsumeFixed64(data)
		case protowire.BytesType:
			_, n = protowire.ConsumeBytes(data)
		default:
			return false
		}
		if n < 0 {
			return false
		}
		data = data[n:]
	}
	return true
}

func appendField(fields map[string]any, key string, val any) {
	if existing, ok := fields[key]; ok {
		if arr, ok := existing.([]any); ok {
			fields[key] = append(arr, val)
		} else {
			fields[key] = []any{existing, val}
		}
	} else {
		fields[key] = val
	}
}
