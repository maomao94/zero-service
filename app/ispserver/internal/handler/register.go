package handler

import (
	"context"

	"zero-service/common/gnetx"
	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
)

// NewRegisterHandler 返回 251-1 注册指令的 isp.IspHandler，对标 Java SipEndpoint.T2511_2514。
func NewRegisterHandler(conf isp.ServerConfig) isp.IspHandler {
	return func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (*isp.Message, error) {
		clientID := req.SendCode
		if clientID == "" {
			logx.WithContext(ctx).Error("[ispserver] 注册失败: SendCode 为空")
			return isp.NewResponse(req, isp.SessionSourceServer, isp.StatusReject, isp.CommandGenericResponseWithItems, nil), nil
		}

		sc := conn.(gnetx.ServerConn)
		sc.Register(clientID)

		logx.WithContext(ctx).Infof("[ispserver] 设备注册成功: %s", clientID)

		return isp.NewResponse(req, isp.SessionSourceServer, isp.StatusSuccess, isp.CommandGenericResponseWithItems,
			[]isp.Item{{
				"heart_beat_interval":       itoa(conf.HeartbeatInterval),
				"patroldevice_run_interval": itoa(conf.DeviceRunInterval),
				"nest_run_interval":         itoa(conf.NestRunInterval),
				"weather_interval":          itoa(conf.WeatherInterval),
			}}), nil
	}
}
