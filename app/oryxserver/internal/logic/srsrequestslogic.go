package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type SrsRequestsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSrsRequestsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SrsRequestsLogic {
	return &SrsRequestsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询 SRS 最近的请求（GET /api/v1/tests/requests，调试用）
func (l *SrsRequestsLogic) SrsRequests(in *oryxserver.SrsRequestsReq) (*oryxserver.SrsRequestsRes, error) {
	data, err := l.svcCtx.OryxClient.SrsRequests(l.ctx)
	if err != nil {
		l.Logger.Errorf("调用 SRS API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 SRS API 失败")
	}
	return &oryxserver.SrsRequestsRes{
		Uri:    data.URI,
		Path:   data.Path,
		Method: data.Method,
	}, nil
}
