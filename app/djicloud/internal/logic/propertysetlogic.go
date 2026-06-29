package logic

import (
	"context"
	"encoding/json"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type PropertySetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPropertySetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PropertySetLogic {
	return &PropertySetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PropertySetLogic) PropertySet(in *djicloud.PropertySetReq) (*djicloud.CommonRes, error) {
	var properties djisdk.PropertySetData
	if err := json.Unmarshal([]byte(in.Properties), &properties); err != nil {
		return &djicloud.CommonRes{Code: -1, Message: "invalid properties JSON: " + err.Error()}, nil
	}

	tid, err := l.svcCtx.DjiClient.PropertySet(l.ctx, in.DeviceSn, properties)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
