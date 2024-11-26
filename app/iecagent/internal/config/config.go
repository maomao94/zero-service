package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	IecSetting struct {
		// Settings 连接配置
		Host    string
		Port    int
		LogMode bool //是否开启log
	}
}
