package solo

import (
	"net/http"

	"zero-service/aiapp/aigtw/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// GatewayMetaHandler 返回只读网关元数据（如知识库后端类型），供前端 RagPanel 与排障。
func GatewayMetaHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps := svcCtx.Dependencies()
		httpx.OkJsonCtx(r.Context(), w, map[string]any{
			"ready":            svcCtx.Ready(),
			"dependencies":     deps,
			"knowledgeBackend": deps["knowledge_backend"],
			"knowledge":        deps["knowledge"],
			"knowledge_error":  deps["knowledge_error"],
		})
	}
}
