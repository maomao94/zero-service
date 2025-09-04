package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

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

// 启动中继拉流（对应HTTP API：/api/ctrl/start_relay_pull，POST请求+JSON Body）
func (l *StartRelayPullLogic) StartRelayPull(in *lalproxy.StartRelayPullReq) (*lalproxy.StartRelayPullRes, error) {
	// 参数验证
	if in.Url == "" {
		return nil, fmt.Errorf("拉流地址不能为空")
	}

	// 构建请求URL
	fullUrl := fmt.Sprintf("%s/api/ctrl/start_relay_pull", l.svcCtx.LalBaseUrl)

	type reqData struct {
		Url                      string `json:"url"`
		StreamName               string `json:"stream_name"`
		PullTimeoutMs            int    `json:"pull_timeout_ms"`
		PullRetryNum             int    `json:"pull_retry_num"`
		AutoStopPullAfterNoOutMs int    `json:"auto_stop_pull_after_no_out_ms"`
		RtspMode                 string `json:"rtsp_mode"`
	}

	reqBody := reqData{
		Url:                      in.Url,
		StreamName:               in.StreamName,
		PullTimeoutMs:            int(in.PullTimeoutMs),
		PullRetryNum:             int(in.PullRetryNum),
		AutoStopPullAfterNoOutMs: int(in.AutoStopPullAfterNoOutMs),
		RtspMode:                 string(in.RtspMode),
	}
	// 调用LAL HTTP API（POST请求）
	resp, err := l.svcCtx.LalClient.Do(l.ctx, http.MethodPost, fullUrl, reqBody)
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

	// 解析JSON响应
	var httpResp struct {
		ErrorCode int               `json:"error_code"`
		Desp      string            `json:"desp"`
		Data      map[string]string `json:"data"`
	}
	if err := json.Unmarshal(body, &httpResp); err != nil {
		l.Logger.Errorf("解析响应JSON失败: %v, 响应内容: %s", err, string(body))
		return nil, fmt.Errorf("解析响应JSON失败: %w", err)
	}

	return &lalproxy.StartRelayPullRes{
		ErrorCode: int32(httpResp.ErrorCode),
		Desp:      httpResp.Desp,
	}, nil
}
