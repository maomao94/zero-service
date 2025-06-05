package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	Alarmx struct {
		AppId             string
		AppSecret         string
		EncryptKey        string
		VerificationToken string
		UserId            []string
		Path              string
	}
}
