// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ai

import (
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/aiapp/aigtw/internal/logic/ai"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
)

// 对话补全
func ChatCompletionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChatCompletionRequest
		if err := httpx.Parse(r, &req); err != nil {
			writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "invalid_request", err.Error())
			return
		}

		if req.Stream {
			// 流式：接管 ResponseWriter，直接写 SSE
			l := ai.NewChatCompletionsLogic(r.Context(), svcCtx, w, r)
			if err := l.ChatCompletionsStream(&req); err != nil {
				// 流式中出错，如果还没写过 header 则返回 OpenAI 错误格式
				code, errType := classifyError(err)
				writeOpenAIError(w, code, errType, errType, err.Error())
			}
		} else {
			// 非流式：标准 JSON 响应
			l := ai.NewChatCompletionsLogic(r.Context(), svcCtx, w, r)
			resp, err := l.ChatCompletions(&req)
			if err != nil {
				code, errType := classifyError(err)
				writeOpenAIError(w, code, errType, errType, err.Error())
			} else {
				httpx.OkJsonCtx(r.Context(), w, resp)
			}
		}
	}
}

// classifyError 根据错误内容分类 HTTP 状态码和错误类型
func classifyError(err error) (int, string) {
	msg := err.Error()
	if strings.Contains(msg, "not found") {
		return http.StatusNotFound, "model_not_found"
	}
	return http.StatusInternalServerError, "internal_error"
}
