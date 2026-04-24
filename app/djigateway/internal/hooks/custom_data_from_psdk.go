package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

// OnCustomDataFromPsdk PSDK 自定义数据上报钩子。
// Topic: thing/product/{gateway_sn}/events
// Direction: up（设备→云平台）
// Method: custom_data_transmission_from_psdk
// PSDK 负载设备有自定义数据上报时通过 events topic 主动推送。
func OnCustomDataFromPsdk(ctx context.Context, gatewaySn string, data *djisdk.CustomDataFromPsdkEvent) {
	logx.WithContext(ctx).Infof("[dji-gateway] psdk custom data: sn=%s value=%s", gatewaySn, data.Value)
}
