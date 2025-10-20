package nacosx

import (
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
)

// LoggerConfig 用于 nacosx 自定义 logger 配置
type LoggerConfig struct {
	Level          string // debug / info / warn / error
	LogDir         string
	AppendToStdout bool
}

// SetUpLogger 初始化全局 logger
func SetUpLogger(cfg LoggerConfig) error {
	clientConfig := constant.ClientConfig{
		LogLevel:       cfg.Level,
		LogDir:         cfg.LogDir,
		AppendToStdout: cfg.AppendToStdout,
	}
	config := logger.BuildLoggerConfig(clientConfig)
	nacosLog, err := logger.InitNacosLogger(config)
	if err != nil {
		panic(err)
	}
	logger.SetLogger(nacosLog)
	return nil
}
