package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartRtpPubLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartRtpPubLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartRtpPubLogic {
	return &StartRtpPubLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 打开GB28181 RTP接收端口（对应HTTP API：/api/ctrl/start_rtp_pub，POST请求+JSON Body）
func (l *StartRtpPubLogic) StartRtpPub(in *lalproxy.StartRtpPubReq) (*lalproxy.StartRtpPubRes, error) {
	// 参数验证
	if in.Port <= 0 || in.Port > 65535 {
		return nil, fmt.Errorf("无效的端口号: %d", in.Port)
	}
	if in.StreamName == "" {
		return nil, fmt.Errorf("流名称不能为空")
	}

	// 构建请求URL
	fullUrl := fmt.Sprintf("%s/api/ctrl/start_rtp_pub", l.svcCtx.LalBaseUrl)

	type reqData struct {
		streamName string `json:"stream_name"`
		port       string `json:"port"`
	}

	reqBody := reqData{
		streamName: in.StreamName,
		port:       in.StreamName,
	}

	// 调用LAL HTTP API（POST请求）
	resp, err := l.svcCtx.LalClient.Do(l.ctx, "POST", fullUrl, reqBody)
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

	return &lalproxy.StartRtpPubRes{
		ErrorCode: int32(httpResp.ErrorCode),
		Desp:      httpResp.Desp,
	}, nil
}
