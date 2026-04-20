package solo

import (
	"net/http"

	"zero-service/aiapp/aigtw/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// GatewayMetaHandler 返回只读网关元数据（如知识库后端类型），供前端 RagPanel 与排障。
func GatewayMetaHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := svcCtx.Config
		kb := ""
		if c.Knowledge.Enabled {
			kb = c.Knowledge.EffectiveBackend()
		}
		kstatus := "disabled"
		kerr := ""
		if svcCtx.Knowledge != nil {
			kstatus = "ok"
		} else if c.Knowledge.Enabled {
			kstatus = "misconfigured"
			kerr = svcCtx.KnowledgeInitErr
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{
			"knowledgeBackend": kb,
			"knowledge":        kstatus,
			"knowledge_error":  kerr,
		})
	}
}
