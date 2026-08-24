package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type VersionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewVersionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VersionsLogic {
	return &VersionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询版本信息
func (l *VersionsLogic) Versions(in *oryxserver.VersionsReq) (*oryxserver.VersionsRes, error) {
	version, err := l.svcCtx.OryxClient.Versions(l.ctx)
	if err != nil {
		l.Logger.Errorf("调用 Oryx API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")
	}
	return &oryxserver.VersionsRes{Version: version}, nil
}
