package netx

import (
	"fmt"
	"io"
	"math"
)

// readLimitedBody 从 reader 读取数据，若超过 limit 字节则返回错误。
// 若 limit <= 0 或 limit == math.MaxInt64 则不限制大小。
func readLimitedBody(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 || limit == math.MaxInt64 {
		return io.ReadAll(r)
	}
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("read body exceeds limit %d bytes", limit)
	}
	return data, nil
}
