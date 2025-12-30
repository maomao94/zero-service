package logic

import (
	"context"
	"time"
	"zero-service/common/tool"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
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

// 剔除 session
func (l *KickSessionLogic) KickSession(in *socketpush.KickSessionReq) (*socketpush.KickSessionRes, error) {
	baseCtx := context.WithoutCancel(l.ctx)
	for _, cli := range l.svcCtx.SocketContainer.GetClients() {
		threading.GoSafe(func() {
			reqId, _ := tool.SimpleUUID()
			socktCTx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
			defer cancel()
			cli.KickSession(socktCTx, &socketgtw.KickSessionReq{
				ReqId: reqId,
				SId:   in.SId,
			})
		})
	}
	return &socketpush.KickSessionRes{}, nil
}
