package logic

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/golang/protobuf/jsonpb"
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

// 停止从远端拉流
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
	resp, err := l.svcCtx.LalClient.Do(l.ctx, "GET", fullUrl, nil)
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

	result := &lalproxy.StopRelayPullRes{}
	if err := jsonpb.UnmarshalString(string(body), result); err != nil {
		l.Logger.Errorf("解析所有分组响应失败: %v, 响应内容: %s", err, string(body))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return result, nil
}
