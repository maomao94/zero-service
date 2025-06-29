// Code generated by goctl. DO NOT EDIT.
// goctl 1.8.4
// Source: iecstream.proto

package server

import (
	"context"

	"zero-service/facade/iecstream/iecstream"
	"zero-service/facade/iecstream/internal/logic"
	"zero-service/facade/iecstream/internal/svc"
)

type IecStreamRpcServer struct {
	svcCtx *svc.ServiceContext
	iecstream.UnimplementedIecStreamRpcServer
}

func NewIecStreamRpcServer(svcCtx *svc.ServiceContext) *IecStreamRpcServer {
	return &IecStreamRpcServer{
		svcCtx: svcCtx,
	}
}

func (s *IecStreamRpcServer) Ping(ctx context.Context, in *iecstream.Req) (*iecstream.Res, error) {
	l := logic.NewPingLogic(ctx, s.svcCtx)
	return l.Ping(in)
}

// 推送 chunk asdu 消息
func (s *IecStreamRpcServer) PushChunkAsdu(ctx context.Context, in *iecstream.PushChunkAsduReq) (*iecstream.PushChunkAsduRes, error) {
	l := logic.NewPushChunkAsduLogic(ctx, s.svcCtx)
	return l.PushChunkAsdu(in)
}
