package logic

import (
	"context"
	"fmt"
	"io"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/golang/protobuf/jsonpb"
	"github.com/zeromicro/go-zero/core/logx"
)

type StartRelayPullLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartRelayPullLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartRelayPullLogic {
	return &StartRelayPullLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 控制服务器从远端拉流至本地
func (l *StartRelayPullLogic) StartRelayPull(in *lalproxy.StartRelayPullReq) (*lalproxy.StartRelayPullRes, error) {
	// 参数验证
	if in.Url == "" {
		return nil, fmt.Errorf("拉流地址不能为空")
	}

	// 构建请求URL
	fullUrl := fmt.Sprintf("%s/api/ctrl/start_relay_pull", l.svcCtx.LalBaseUrl)

	// 准备请求数据（转换为LAL API要求的下划线格式）
	reqData := map[string]interface{}{
		"url":                            in.Url,
		"stream_name":                    in.StreamName,
		"pull_timeout_ms":                in.PullTimeoutMs,
		"pull_retry_num":                 in.PullRetryNum,
		"auto_stop_pull_after_no_out_ms": in.AutoStopPullAfterNoOutMs,
		"rtsp_mode":                      in.RtspMode,
	}

	// 调用LAL HTTP API（POST请求）
	resp, err := l.svcCtx.LalClient.Do(l.ctx, "POST", fullUrl, reqData)
	if err != nil {
		l.Logger.Errorf("调用LAL API失败: %v, URL: %s", err, fullUrl)
		return nil, fmt.Errorf("调用LAL API失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		l.Logger.Errorf("LAL API返回非200状态码: %d, 响应内容: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("LAL API返回异常状态码: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Logger.Errorf("读取响应体失败: %v", err)
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	result := &lalproxy.StartRelayPullRes{}
	if err := jsonpb.UnmarshalString(string(body), result); err != nil {
		l.Logger.Errorf("解析所有分组响应失败: %v, 响应内容: %s", err, string(body))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return result, nil
}
