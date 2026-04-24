package logic

import (
	"context"
	"encoding/json"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetPropertyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetPropertyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetPropertyLogic {
	return &SetPropertyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SetPropertyLogic) SetProperty(in *djigateway.SetPropertyReq) (*djigateway.CommonRes, error) {
	var properties djisdk.PropertySetData
	if err := json.Unmarshal([]byte(in.Properties), &properties); err != nil {
		l.Errorf("[property] unmarshal properties failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: "invalid properties JSON: " + err.Error()}, nil
	}

	tid, err := l.svcCtx.DjiClient.SetProperty(l.ctx, in.DeviceSn, properties)
	if err != nil {
		l.Errorf("[property] set property failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
