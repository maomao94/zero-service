package holiday

import "context"

// Source 加载以 yyyy-mm-dd 为 key 的特殊日期记录。
type Source interface {
	Load(ctx context.Context) (map[string]Entry, error)
}
