package config

import (
	"github.com/zeromicro/go-zero/zrpc"

	"zero-service/common/gormx"
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
	DB                   gormx.Config
	Oss                  OssConf
	ThumbTaskConcurrency int `json:",default=2"`
	Upload               UploadConf
}

type OssConf struct {
	TenantMode bool
}

type UploadConf struct {
	TempDir       string `json:",default=/opt/data/temp"`
	KeepTempFiles bool   `json:",default=false"`
	Image         ImageUploadConf
}

type ImageUploadConf struct {
	MaxExifRead int `json:",default=65536"`
	Thumb       ImageVariantConf
}

type ImageVariantConf struct {
	Enabled bool `json:",default=false"`
	Width   int  `json:",default=300"`
	Height  int  `json:",default=300"`
}
