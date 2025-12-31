package logic

import (
	"context"

	"zero-service/facade/streamevent/internal/svc"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpSocketMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpSocketMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpSocketMessageLogic {
	return &UpSocketMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 上行socket标准消息, 可以用于__up__和自定义up事件
func (l *UpSocketMessageLogic) UpSocketMessage(in *streamevent.UpSocketMessageReq) (*streamevent.UpSocketMessageReq, error) {
	return &streamevent.UpSocketMessageReq{}, nil
}
