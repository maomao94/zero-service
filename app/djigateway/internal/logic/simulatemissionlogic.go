package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type SimulateMissionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSimulateMissionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SimulateMissionLogic {
	return &SimulateMissionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SimulateMissionLogic) SimulateMission(in *djigateway.SimulateMissionReq) (*djigateway.CommonRes, error) {
	data := &djisdk.SimulateMission{
		IsEnable:  true,
		Latitude:  in.Latitude,
		Longitude: in.Longitude,
	}
	tid, err := l.svcCtx.DjiClient.SimulateMission(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[drc] simulate mission failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
