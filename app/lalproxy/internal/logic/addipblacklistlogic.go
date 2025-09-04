package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddIpBlacklistLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddIpBlacklistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddIpBlacklistLogic {
	return &AddIpBlacklistLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 添加IP到黑名单（对应HTTP API：/api/ctrl/add_ip_blacklist，POST请求+JSON Body；目前仅支持HLS协议）
func (l *AddIpBlacklistLogic) AddIpBlacklist(in *lalproxy.AddIpBlacklistReq) (*lalproxy.AddIpBlacklistRes, error) {
	// 参数验证（IP地址格式）
	if net.ParseIP(in.Ip) == nil {
		return nil, fmt.Errorf("无效的IP地址: %s", in.Ip)
	}
	if in.DurationSec < 0 {
		return nil, fmt.Errorf("过期时间不能为负数: %d", in.DurationSec)
	}

	// 构建请求URL
	fullUrl := fmt.Sprintf("%s/api/ctrl/add_ip_blacklist", l.svcCtx.LalBaseUrl)

	type reqData struct {
		ip          string `json:"ip"`
		durationSec int    `json:"duration_sec"`
	}

	reqBody := reqData{
		ip:          in.Ip,
		durationSec: int(in.DurationSec),
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
		ErrorCode int    `json:"error_code"`
		Desp      string `json:"desp"`
	}
	if err := json.Unmarshal(body, &httpResp); err != nil {
		l.Logger.Errorf("解析响应JSON失败: %v, 响应内容: %s", err, string(body))
		return nil, fmt.Errorf("解析响应JSON失败: %w", err)
	}

	return &lalproxy.AddIpBlacklistRes{
		ErrorCode: int32(httpResp.ErrorCode),
		Desp:      httpResp.Desp,
	}, nil
}
