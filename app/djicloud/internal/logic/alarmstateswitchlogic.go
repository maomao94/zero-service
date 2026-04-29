package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AlarmStateSwitchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAlarmStateSwitchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlarmStateSwitchLogic {
	return &AlarmStateSwitchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// AlarmStateSwitch 切换机巢声光报警状态。
func (l *AlarmStateSwitchLogic) AlarmStateSwitch(in *djicloud.AlarmStateSwitchReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.AlarmStateSwitch(l.ctx, in.DeviceSn, int(in.Action))
	if err != nil {
		l.Errorf("[remote-debug] alarm state switch failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
