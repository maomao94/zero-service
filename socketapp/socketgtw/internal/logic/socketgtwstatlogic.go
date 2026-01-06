package logic

import (
	"context"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
)

type SocketGtwStatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSocketGtwStatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SocketGtwStatLogic {
	return &SocketGtwStatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取网关统计信息
func (l *SocketGtwStatLogic) SocketGtwStat(in *socketgtw.SocketGtwStatReq) (*socketgtw.SocketGtwStatRes, error) {
	sessionCount := l.svcCtx.SocketServer.SessionCount()
	return &socketgtw.SocketGtwStatRes{
		Sessions: int64(sessionCount),
	}, nil
}
