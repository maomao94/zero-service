package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/zeromicro/go-zero/core/logx"
)

type StopRelayPullLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopRelayPullLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopRelayPullLogic {
	return &StopRelayPullLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 停止中继拉流（对应HTTP API：/api/ctrl/stop_relay_pull，GET请求+URL参数）
func (l *StopRelayPullLogic) StopRelayPull(in *lalproxy.StopRelayPullReq) (*lalproxy.StopRelayPullRes, error) {
	// 参数验证
	if in.StreamName == "" {
		return nil, fmt.Errorf("流名称不能为空")
	}

	// 构建请求URL（带参数）
	queryParams := url.Values{}
	queryParams.Add("stream_name", in.StreamName)
	fullUrl := fmt.Sprintf("%s/api/ctrl/stop_relay_pull?%s", l.svcCtx.LalBaseUrl, queryParams.Encode())

	// 调用LAL HTTP API
	resp, err := l.svcCtx.LalClient.Do(l.ctx, http.MethodGet, fullUrl, nil)
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
		ErrorCode int    `json:"error_code"`
		Desp      string `json:"desp"`
	}
	if err := json.Unmarshal(body, &httpResp); err != nil {
		l.Logger.Errorf("解析响应JSON失败: %v, 响应内容: %s", err, string(body))
		return nil, fmt.Errorf("解析响应JSON失败: %w", err)
	}

	return &lalproxy.StopRelayPullRes{
		ErrorCode: int32(httpResp.ErrorCode),
		Desp:      httpResp.Desp,
	}, nil
}
