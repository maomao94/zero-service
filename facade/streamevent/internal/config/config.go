package config

import "github.com/zeromicro/go-zero/zrpc"

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
	DisableStmtLog bool `json:",optional"`
	TaosDB         struct {
		DataSource string
		DBName     string `json:",default=default"`
	}
	SqliteDB struct {
		DataSource string
	}
}
