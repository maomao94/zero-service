package handler

import (
	"context"
	"zero-service/app/bridgemqtt/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type testHandler struct {
	svcCtx *svc.ServiceContext
}

func NewTestHandler(svcCtx *svc.ServiceContext) *testHandler {
	return &testHandler{
		svcCtx: svcCtx,
	}
}

func (h *testHandler) Handle(ctx context.Context, msg []byte) error {
	logx.Info("testHandler: ", string(msg))
	return nil
}
