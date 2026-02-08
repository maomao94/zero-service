package logic

import (
	"context"
	"fmt"
	"strings"
	"zero-service/app/logdump/internal/svc"
	"zero-service/app/logdump/logdump"

	"github.com/zeromicro/go-zero/core/logx"
)

type PushLogLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushLogLogic {
	return &PushLogLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 推送日志
func (l *PushLogLogic) PushLog(in *logdump.PushLogReq) (*logdump.PushLogRes, error) {
	// 构建允许的 extra 字段集合
	allowedExtra := make(map[string]struct{}, len(l.svcCtx.Config.ExtraFields))
	for _, key := range l.svcCtx.Config.ExtraFields {
		allowedExtra[key] = struct{}{}
	}
	for _, logEntry := range in.Logs {
		// 基础字段
		fields := []logx.LogField{
			logx.Field("seq", logEntry.Seq),
			logx.Field("service", logEntry.Service),
		}
		// 构建 extra 字符串和结构化字段
		extraParts := make([]string, 0, len(logEntry.Extra))
		for k, v := range logEntry.Extra {
			// 仅添加允许的 extra 字段到结构化字段
			if _, ok := allowedExtra[k]; ok {
				fields = append(fields, logx.Field(k, v))
			}
			// 拼接 extra 字段字符串
			extraParts = append(extraParts, fmt.Sprintf("%s=%s", k, v))
		}
		extraStr := strings.Join(extraParts, ", ")

		// 拼接最终日志消息
		msg := fmt.Sprintf("#[%s] %s", logEntry.Service, logEntry.Message)
		if extraStr != "" {
			msg = fmt.Sprintf("%s | %s#", msg, extraStr)
		} else {
			msg = fmt.Sprintf("%s#", msg)
		}

		// 输出日志
		switch logEntry.Level {
		case logdump.LogLevel_ERROR:
			l.Logger.WithFields(fields...).Error(msg)
		default:
			l.Logger.WithFields(fields...).Info(msg)
		}
	}
	return &logdump.PushLogRes{}, nil
}
