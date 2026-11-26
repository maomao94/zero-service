package nacosx

import (
	"os"
	"path/filepath"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
)

// LoggerConfig 用于 nacosx 自定义 logger 配置
type LoggerConfig struct {
	Level          string // debug / info / warn / error
	LogDir         string
	AppendToStdout bool
}

// SDK 的 logger.InitLogger 只在全局 logger 为空时生效，这里在包初始化阶段
// 设置的 logger 会覆盖之后各 client 的 LogLevel/LogDir，因此默认配置必须
// 支持环境变量覆盖，避免现场无法调整 Nacos 日志级别。
func init() {
	if err := SetUpLogger(defaultLoggerConfig()); err != nil {
		// 包初始化阶段不能阻断进程启动，退化为 SDK 默认 logger（仅标准输出）。
		logger.InitDefaultLogger()
	}
}

func defaultLoggerConfig() LoggerConfig {
	cfg := LoggerConfig{
		Level:          "error",
		LogDir:         filepath.Join(os.TempDir(), "nacos", "log"),
		AppendToStdout: true,
	}
	if v := os.Getenv(envLogLevel); v != "" {
		cfg.Level = v
	}
	if v := os.Getenv(envLogDir); v != "" {
		cfg.LogDir = v
	}
	return cfg
}

// SetUpLogger 初始化全局 logger，可在进程启动早期调用以覆盖默认配置。
func SetUpLogger(cfg LoggerConfig) error {
	clientConfig := constant.ClientConfig{
		LogLevel:       cfg.Level,
		LogDir:         cfg.LogDir,
		AppendToStdout: cfg.AppendToStdout,
	}
	config := logger.BuildLoggerConfig(clientConfig)
	nacosLog, err := logger.InitNacosLogger(config)
	if err != nil {
		return err
	}
	logger.SetLogger(nacosLog)
	return nil
}

// applyClientDefaults 补齐缓存与日志目录，避免各服务把 cache/log 目录
// 落到进程工作目录（容器内通常是只读的应用目录）。
func applyClientDefaults(cc *constant.ClientConfig) {
	if cc.CacheDir == "" {
		cc.CacheDir = filepath.Join(os.TempDir(), "nacos", "cache")
	}
	if cc.LogDir == "" {
		cc.LogDir = filepath.Join(os.TempDir(), "nacos", "log")
	}
}
