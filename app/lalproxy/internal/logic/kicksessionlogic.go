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

type KickSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewKickSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickSessionLogic {
	return &KickSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 强行踢出关闭指定会话
func (l *KickSessionLogic) KickSession(in *lalproxy.KickSessionReq) (*lalproxy.KickSessionRes, error) {
	// 参数验证
	if in.StreamName == "" || in.SessionId == "" {
		return nil, fmt.Errorf("流名称和会话ID不能为空")
	}

	// 构建请求URL
	fullUrl := fmt.Sprintf("%s/api/ctrl/kick_session", l.svcCtx.LalBaseUrl)

	// 准备请求数据
	reqData := map[string]interface{}{
		"stream_name":  in.StreamName,
		"session_id":   in.SessionId,
		"session_type": in.SessionType, // 可选参数
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

	result := &lalproxy.KickSessionRes{}
	if err := jsonpb.UnmarshalString(string(body), result); err != nil {
		l.Logger.Errorf("解析所有分组响应失败: %v, 响应内容: %s", err, string(body))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return result, nil
}
