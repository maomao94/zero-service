package logic

import (
	"context"
	"encoding/json"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfigUpdateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConfigUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfigUpdateLogic {
	return &ConfigUpdateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ConfigUpdate 下发设备配置更新。
func (l *ConfigUpdateLogic) ConfigUpdate(in *djigateway.ConfigUpdateReq) (*djigateway.CommonRes, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(in.Config), &config); err != nil {
		l.Errorf("[config] unmarshal config failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: "invalid config JSON: " + err.Error()}, nil
	}

	data := &djisdk.ConfigUpdateData{
		ConfigScope: in.ConfigScope,
		Config:      config,
	}
	tid, err := l.svcCtx.DjiClient.ConfigUpdate(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[config] config update failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
