package logic

import (
	"context"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/zeromicro/go-zero/core/logx"
)

type KickSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewKickSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickSessionLogic {
	return &KickSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 强行踢出关闭指定会话
func (l *KickSessionLogic) KickSession(in *lalproxy.KickSessionReq) (*lalproxy.KickSessionRes, error) {
	// todo: add your logic here and delete this line

	return &lalproxy.KickSessionRes{}, nil
}
