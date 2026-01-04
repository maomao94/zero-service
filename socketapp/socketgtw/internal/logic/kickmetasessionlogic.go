package logic

import (
	"context"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
)

type KickMetaSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewKickMetaSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickMetaSessionLogic {
	return &KickMetaSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 指定元数据剔除 session
func (l *KickMetaSessionLogic) KickMetaSession(in *socketgtw.KickMetaSessionReq) (*socketgtw.KickMetaSessionRes, error) {
	sessions, ok := l.svcCtx.SocketServer.GetSessionByKey(in.Key, in.Value)
	if ok {
		for _, session := range sessions {
			session.Close()
		}
	}
	return &socketgtw.KickMetaSessionRes{
		ReqId: in.ReqId,
	}, nil
}
