package logic

import (
	"context"
	"encoding/json"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
func (l *ConfigUpdateLogic) ConfigUpdate(in *djicloud.ConfigUpdateReq) (*djicloud.CommonRes, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(in.Config), &config); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "config JSON 不合法")
	}

	data := &djisdk.ConfigUpdateData{
		ConfigScope: in.ConfigScope,
		Config:      config,
	}
	tid, err := l.svcCtx.DjiClient.ConfigUpdate(l.ctx, in.DeviceSn, data)
	if err != nil {
		return commandRes(tid, err)
	}
	return okRes(tid), nil
}
