package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type SrsVersionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSrsVersionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SrsVersionsLogic {
	return &SrsVersionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询 SRS 版本信息（GET /api/v1/versions，免鉴权）
func (l *SrsVersionsLogic) SrsVersions(in *oryxserver.SrsVersionsReq) (*oryxserver.SrsVersionsRes, error) {
	data, err := l.svcCtx.OryxClient.SrsVersions(l.ctx)
	if err != nil {
		l.Logger.Errorf("调用 SRS API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 SRS API 失败")
	}
	return &oryxserver.SrsVersionsRes{
		Major:    data.Major,
		Minor:    data.Minor,
		Revision: data.Revision,
		Version:  data.Version,
	}, nil
}
