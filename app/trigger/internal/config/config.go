package config

import (
	"time"

	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	NacosConfig struct {
		IsRegister  bool
		Host        string
		Port        uint64
		Username    string
		PassWord    string
		NamespaceId string
		ServiceName string
	} `json:",optional"`
	RedisDB         int           `json:",optional"`
	DB              gormx.Config  `json:",optional"`
	GracePeriod     time.Duration `json:",default=30s"`
	StreamEventConf zrpc.RpcClientConf
	CronTask        CronTaskConfig `json:",optional"`
	// SocketPushConf socketpush 推送服务（可选，未配置时批次确认只记录日志）
	SocketPushConf zrpc.RpcClientConf `json:",optional"`
	// PlanConfirm 批次二次确认通知配置
	PlanConfirm PlanConfirmConfig `json:",optional"`
}

type PlanConfirmConfig struct {
	// NotifyRoomPrefix 机构级房间前缀，最终房间为 ${prefix}${deptCode}
	NotifyRoomPrefix string `json:",default=dept:"`
	// NotifyFallbackRoom 批次未携带机构时的兜底房间
	NotifyFallbackRoom string `json:",default=plan-confirm"`
	// NotifyEvent socket 事件名
	NotifyEvent string `json:",default=plan:batch-confirm-required"`
}

type CronTaskConfig struct {
	MaxDelay time.Duration `json:",default=30m"`
}
