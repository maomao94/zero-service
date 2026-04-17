package builtin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// =============================================================================
// now —— 返回服务器当前时间
// =============================================================================

type nowParam struct {
	Format string `json:"format,omitempty" jsonschema:"description=时间格式，遵循 Go time.Format 规则; 为空则返回 RFC3339"`
}

type nowResult struct {
	Time string `json:"time"`
	Unix int64  `json:"unix"`
}

// NewNow 返回一个获取当前时间的工具。
func NewNow() tool.InvokableTool {
	t, err := utils.InferTool("now", "Now: 返回服务器当前时间 (RFC3339 或指定 Go 格式)。",
		func(_ context.Context, in *nowParam) (*nowResult, error) {
			t := time.Now()
			layout := in.Format
			if layout == "" {
				layout = time.RFC3339
			}
			return &nowResult{Time: t.Format(layout), Unix: t.Unix()}, nil
		})
	if err != nil {
		panic(err)
	}
	return t
}

// =============================================================================
// random_id —— 返回一段随机 hex ID
// =============================================================================

type randIDParam struct {
	Bytes int `json:"bytes,omitempty" jsonschema:"description=随机字节数, 默认 8 (=16 位 hex)"`
}

type randIDResult struct {
	ID string `json:"id"`
}

// NewRandomID 返回一个产生随机 ID 的工具。
func NewRandomID() tool.InvokableTool {
	t, err := utils.InferTool("random_id", "RandomID: 生成随机 hex ID, 用于临时命名。",
		func(_ context.Context, in *randIDParam) (*randIDResult, error) {
			n := in.Bytes
			if n <= 0 {
				n = 8
			}
			buf := make([]byte, n)
			if _, err := rand.Read(buf); err != nil {
				return nil, err
			}
			return &randIDResult{ID: hex.EncodeToString(buf)}, nil
		})
	if err != nil {
		panic(err)
	}
	return t
}
