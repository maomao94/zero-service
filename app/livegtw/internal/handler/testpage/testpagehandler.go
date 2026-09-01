package testpage

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var meetingPage []byte

// MeetingTestPageHandler 返回 meeting 测试页 HTML（livekit-client 全功能验证）。
func MeetingTestPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(meetingPage)
	}
}
