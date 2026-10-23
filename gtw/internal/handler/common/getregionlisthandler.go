package common

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zero-service/common/gtwx"
	"zero-service/gtw/internal/logic/common"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"
)

// 获取区域列表
func GetRegionListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetRegionListRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		l := common.NewGetRegionListLogic(r.Context(), svcCtx)
		resp, err := l.GetRegionList(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
		} else {
			gtwx.WriteOK(r.Context(), w, resp)
		}
	}
}
