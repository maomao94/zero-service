// Code generated by goctl. DO NOT EDIT.
// goctl 1.7.3
// Source: trigger.proto

package server

import (
	"context"

	"zero-service/app/trigger/internal/logic"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
)

type TriggerRpcServer struct {
	svcCtx *svc.ServiceContext
	trigger.UnimplementedTriggerRpcServer
}

func NewTriggerRpcServer(svcCtx *svc.ServiceContext) *TriggerRpcServer {
	return &TriggerRpcServer{
		svcCtx: svcCtx,
	}
}

func (s *TriggerRpcServer) Ping(ctx context.Context, in *trigger.Req) (*trigger.Res, error) {
	l := logic.NewPingLogic(ctx, s.svcCtx)
	return l.Ping(in)
}

func (s *TriggerRpcServer) SendTrigger(ctx context.Context, in *trigger.SendTriggerReq) (*trigger.SendTriggerRes, error) {
	l := logic.NewSendTriggerLogic(ctx, s.svcCtx)
	return l.SendTrigger(in)
}
