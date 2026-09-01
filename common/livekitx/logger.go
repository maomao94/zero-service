package livekitx

import (
	"github.com/livekit/protocol/logger"
	"github.com/livekit/protocol/logger/zaputil"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

// logxAdapter 把 LiveKit 的 protocol Logger 接到 go-zero logx 上。
//
// protocol v1.49.0 的 Logger 接口方法集（以源码核对）：
//   - Debugw/Infow/Warnw/Errorw：msg + zap 风格 keysAndValues 键值对；
//     Warnw/Errorw 额外携带 err（zap 实现里 err 作为 "error" 字段附加）。
//   - 派生方法：WithValues/WithUnlikelyValues/WithName/WithComponent/
//     WithCallDepth/WithItemSampler/WithoutSampler/WithDeferredValues。
//
// logx 是包级函数 + 全局字段模型，没有 logger 实例派生概念，因此所有
// 派生方法都返回同一个适配器；name/component/call depth 只影响输出
// 上下文，不影响日志内容是否输出。
type logxAdapter struct{}

var _ logger.Logger = (*logxAdapter)(nil)

// Debugw 映射到 logx.Debugw。
func (a *logxAdapter) Debugw(msg string, keysAndValues ...any) {
	logx.Debugw(msg, toLogFields(keysAndValues)...)
}

// Infow 映射到 logx.Infow。
func (a *logxAdapter) Infow(msg string, keysAndValues ...any) {
	logx.Infow(msg, toLogFields(keysAndValues)...)
}

// Warnw 映射到 logx.Infow：logx v1.10.3 没有 warn 级别（只有
// debug/info/error/severe），按 info 输出，避免把 SDK 的瞬时告警抬进
// error 告警链路；err 附加为 error 字段。
func (a *logxAdapter) Warnw(msg string, err error, keysAndValues ...any) {
	logx.Infow(msg, append(toLogFields(keysAndValues), logx.Field("error", err))...)
}

// Errorw 映射到 logx.Errorw；err 附加为 error 字段。
func (a *logxAdapter) Errorw(msg string, err error, keysAndValues ...any) {
	logx.Errorw(msg, append(toLogFields(keysAndValues), logx.Field("error", err))...)
}

func (a *logxAdapter) WithValues(keysAndValues ...any) logger.Logger { return a }

func (a *logxAdapter) WithUnlikelyValues(keysAndValues ...any) logger.UnlikelyLogger {
	return logger.NewUnlikelyLogger(a, keysAndValues...)
}

func (a *logxAdapter) WithName(name string) logger.Logger           { return a }
func (a *logxAdapter) WithComponent(component string) logger.Logger { return a }
func (a *logxAdapter) WithCallDepth(depth int) logger.Logger        { return a }
func (a *logxAdapter) WithItemSampler() logger.Logger               { return a }
func (a *logxAdapter) WithoutSampler() logger.Logger                { return a }

func (a *logxAdapter) WithDeferredValues() (logger.Logger, logger.DeferredFieldResolver) {
	return a, zaputil.NoOpDeferrer{}
}

// toLogFields 把 zap 风格的 keysAndValues 键值对转换为 logx.LogField。
// key 必须是 string（SDK 内部保证）；非 string key 直接跳过。
func toLogFields(keysAndValues ...any) []logx.LogField {
	if len(keysAndValues) == 0 {
		return nil
	}
	fields := make([]logx.LogField, 0, len(keysAndValues)/2)
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		fields = append(fields, logx.Field(key, keysAndValues[i+1]))
	}
	return fields
}

// initDefaultLogger 把 LiveKit SDK/protocol 的全局日志接到 go-zero logx。
// 这是包级全局行为：
//   - protocol logger.SetLogger 设置 protocol 包全局日志（name 参数用于
//     日志命名，logx 无此概念，传空串）；
//   - SDK 另有自己的包级 logger 变量（lksdk.SetLogger），不设置的话
//     SDK 内部日志仍走默认 stdr，因此同步设置。
//
// 两者都只影响设置之后创建的 Room/Participant 等对象；重复调用只是覆盖
// 全局值，幂等。业务后续仍可自行调用 SDK 的 SetLogger 覆盖。
func initDefaultLogger() {
	adapter := &logxAdapter{}
	logger.SetLogger(adapter, "")
	lksdk.SetLogger(adapter)
}
