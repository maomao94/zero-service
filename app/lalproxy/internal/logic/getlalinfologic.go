package logic

import (
	"context"
	"fmt"
	"io"
	"strings"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/golang/protobuf/jsonpb"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetLalInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetLalInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLalInfoLogic {
	return &GetLalInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询服务器基本信息
func (l *GetLalInfoLogic) GetLalInfo(in *lalproxy.GetLalInfoReq) (*lalproxy.GetLalInfoRes, error) {
	// 构建请求URL
	fullUrl := fmt.Sprintf("%s/api/stat/lal_info", l.svcCtx.LalBaseUrl)

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

	unmarshaler := &jsonpb.Unmarshaler{
		AllowUnknownFields: true, // 核心配置：允许未知字段，不报错
	}
	result := &lalproxy.GetLalInfoRes{}
	if err := unmarshaler.Unmarshal(strings.NewReader(string(body)), result); err != nil {
		l.Logger.Errorf("解析所有分组响应失败: %v, 响应内容: %s", err, string(body))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return result, nil
}
