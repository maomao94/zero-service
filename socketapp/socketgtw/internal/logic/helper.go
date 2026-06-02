package logic

import (
	"encoding/json"

	"github.com/zeromicro/go-zero/core/jsonx"
)

func parseJsonPayload(raw string) any {
	b := []byte(raw)
	var js json.RawMessage
	if jsonx.Unmarshal(b, &js) == nil {
		return json.RawMessage(b)
	}
	return raw
}
