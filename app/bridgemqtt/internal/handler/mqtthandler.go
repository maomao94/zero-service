package handler

import (
	"context"
	"errors"
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

func (h *testHandler) Consume(ctx context.Context, topic string, payload []byte) error {
	logx.Info("testHandler: ", string(payload))
	return errors.New("test error")
}
