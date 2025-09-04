package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"zero-service/common/lalx"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetGroupInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupInfoLogic {
	return &GetGroupInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询指定group信息（对应HTTP API：/api/stat/group，GET请求+URL参数）
func (l *GetGroupInfoLogic) GetGroupInfo(in *lalproxy.GetGroupInfoReq) (*lalproxy.GetGroupInfoRes, error) {
	// 参数验证
	if in.StreamName == "" {
		return nil, fmt.Errorf("流名称不能为空")
	}
	// 构建请求URL
	queryParams := url.Values{}
	queryParams.Add("stream_name", in.StreamName)
	fullUrl := fmt.Sprintf("%s/api/stat/group?%s", l.svcCtx.LalBaseUrl, queryParams.Encode())
	// 调用LAL HTTP API，失败直接返回error
	resp, err := l.svcCtx.LalClient.Do(l.ctx, "GET", fullUrl, nil)
	if err != nil {
		l.Logger.Errorf("调用LAL API失败: %v, URL: %s", err, fullUrl)
		return nil, fmt.Errorf("调用LAL API失败: %w", err)
	}
	defer resp.Body.Close()

	// 非200状态码返回error
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
		ErrorCode int            `json:"error_code"`
		Desp      string         `json:"desp"`
		Data      lalx.GroupData `json:"data"`
	}
	if err := json.Unmarshal(body, &httpResp); err != nil {
		l.Logger.Errorf("解析响应JSON失败: %v, 响应内容: %s", err, string(body))
		return nil, fmt.Errorf("解析响应JSON失败: %w", err)
	}
	data := &lalproxy.GroupData{}
	err = copier.Copy(data, httpResp.Data)
	if err != nil {
		l.Logger.Errorf("转换数据结构失败: %v", err)
		return nil, fmt.Errorf("转换数据结构失败: %w", err)
	}
	// LAL返回的错误通过响应结构体传递，不返回error
	return &lalproxy.GetGroupInfoRes{
		ErrorCode: int32(httpResp.ErrorCode),
		Desp:      httpResp.Desp,
		Data:      data,
	}, nil
}
