package config

import (
	"time"

	"zero-service/common/gormx"
	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config 为 ispagent gRPC 服务配置，包含 zrpc 基础配置和 ISP TCP 客户端参数。
type Config struct {
	zrpc.RpcServerConf
	IspSetting isp.ClientConfig
	DB         gormx.Config `json:",optional"`
	CronTask   CronTaskConfig
	ModelSync  ModelSyncConfig
}

type ModelSyncConfig struct {
	FTPS ModelSyncFTPSConfig
}

type ModelSyncFTPSConfig struct {
	Address            string        `json:",optional"`
	Username           string        `json:",optional"`
	Password           string        `json:",optional"`
	RemoteDir          string        `json:",optional"`
	TLSMode            string        `json:",default=implicit"` // implicit or explicit
	InsecureSkipVerify bool          `json:",optional"`
	Timeout            time.Duration `json:",default=30s"`
	DisableEPSV        bool          `json:",optional"`
	UseTemporaryFile   bool          `json:",default=true"`
}

type CronTaskConfig struct {
	Interval   time.Duration `json:",default=2s"`
	LockExpire time.Duration `json:",default=300s"`
	MaxDelay   time.Duration `json:",default=30m"` // 最大延迟容忍，超过则跳过执行直接计算下次时间
}
