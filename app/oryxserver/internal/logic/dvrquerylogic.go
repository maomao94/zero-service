package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type DvrQueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDvrQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DvrQueryLogic {
	return &DvrQueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询 DVR 云录制配置
func (l *DvrQueryLogic) DvrQuery(in *oryxserver.DvrQueryReq) (*oryxserver.DvrQueryRes, error) {
	data, err := l.svcCtx.OryxClient.DvrQuery(l.ctx)
	if err != nil {
		l.Logger.Errorf("调用 Oryx API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")
	}
	return &oryxserver.DvrQueryRes{All: data.All, Secret: data.Secret}, nil
}
