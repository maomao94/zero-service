package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type HooksQueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHooksQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HooksQueryLogic {
	return &HooksQueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询回调配置
func (l *HooksQueryLogic) HooksQuery(in *oryxserver.HooksQueryReq) (*oryxserver.HooksQueryRes, error) {
	data, err := l.svcCtx.OryxClient.HooksQuery(l.ctx)
	if err != nil {
		l.Logger.Errorf("调用 Oryx API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")
	}
	return &oryxserver.HooksQueryRes{
		Req:    data.Req,
		Res:    data.Res,
		Target: data.Target,
		Opaque: data.Opaque,
		All:    data.All,
		Host:   data.Host,
	}, nil
}
