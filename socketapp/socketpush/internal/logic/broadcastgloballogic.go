package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
)

type BroadcastGlobalLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBroadcastGlobalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BroadcastGlobalLogic {
	return &BroadcastGlobalLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向所有在线前端广播消息
func (l *BroadcastGlobalLogic) BroadcastGlobal(in *socketpush.BroadcastGlobalReq) (*socketpush.BroadcastGlobalRes, error) {
	// 广播类接口保持原语义：单个网关失败只记录日志，不返回错误，
	// 避免调用方重试导致重复广播；失败节点已由 fanoutPush 逐个记录。
	err := fanoutPush(l.ctx, l.Logger, l.svcCtx.SocketContainer.GetClients(), l.svcCtx.Config.PushTimeout,
		func(ctx context.Context, cli socketgtw.SocketGtwClient) error {
			_, err := cli.BroadcastGlobal(ctx, &socketgtw.BroadcastGlobalReq{
				ReqId:   in.ReqId,
				Event:   in.Event,
				Payload: in.Payload,
			})
			return err
		})
	if err != nil {
		l.Errorf("broadcast global partially failed: reqId=%s err=%v", in.ReqId, err)
	}
	return &socketpush.BroadcastGlobalRes{
		ReqId: in.ReqId,
	}, nil
}
