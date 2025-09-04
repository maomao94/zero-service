package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"
	"zero-service/common/lalx"

	"github.com/jinzhu/copier"
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

// 查询服务器基础信息（对应HTTP API：/api/stat/lal_info，GET请求，无参数）
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

	// 解析JSON响应
	var httpResp struct {
		ErrorCode int                `json:"error_code"`
		Desp      string             `json:"desp"`
		Data      lalx.LalServerData `json:"data"`
	}
	if err := json.Unmarshal(body, &httpResp); err != nil {
		l.Logger.Errorf("解析响应JSON失败: %v, 响应内容: %s", err, string(body))
		return nil, fmt.Errorf("解析响应JSON失败: %w", err)
	}
	data := &lalproxy.LalServerData{}
	err = copier.Copy(data, httpResp.Data)
	if err != nil {
		l.Logger.Errorf("转换数据结构失败: %v", err)
		return nil, fmt.Errorf("转换数据结构失败: %w", err)
	}
	return &lalproxy.GetLalInfoRes{
		ErrorCode: int32(httpResp.ErrorCode),
		Desp:      httpResp.Desp,
		Data:      data,
	}, nil
}
