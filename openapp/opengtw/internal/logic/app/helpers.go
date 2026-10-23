package app

import (
	"time"

	"zero-service/common/carbonx"
)

const (
	defaultPageNum  = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

func normalizePage(pageNum, pageSize int64) (int64, int64) {
	if pageNum <= 0 {
		pageNum = defaultPageNum
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return pageNum, pageSize
}

// formatTime 统一使用 carbonx 的 yyyy-MM-dd HH:mm:ss，零值返回空串。
func formatTime(t time.Time) string {
	return carbonx.FormatDateTimeOrEmpty(t)
}
