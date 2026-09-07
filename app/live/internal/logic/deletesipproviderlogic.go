package logic

import (
	"context"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSipProviderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteSipProviderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSipProviderLogic {
	return &DeleteSipProviderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteSipProviderLogic) DeleteSipProvider(in *live.DeleteSipProviderReq) (*live.DeleteSipProviderRes, error) {
	id := strings.TrimSpace(in.GetId())
	if id == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "供应商 ID 不能为空")
	}
	if err := l.svcCtx.MeetingRepo.DeleteSipProvider(l.ctx, id); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "删除供应商失败")
	}
	l.Logger.Infof("SIP provider deleted: %s", id)
	return &live.DeleteSipProviderRes{}, nil
}
