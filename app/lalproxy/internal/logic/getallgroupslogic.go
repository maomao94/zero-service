package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"zero-service/common/lalx"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetAllGroupsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAllGroupsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAllGroupsLogic {
	return &GetAllGroupsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询所有流分组的信息
func (l *GetAllGroupsLogic) GetAllGroups(in *lalproxy.GetAllGroupsReq) (*lalproxy.GetAllGroupsRes, error) {
	// 构建请求URL
	fullUrl := fmt.Sprintf("%s/api/stat/all_group", l.svcCtx.LalBaseUrl)

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
		ErrorCode int              `json:"error_code"`
		Desp      string           `json:"desp"`
		Groups    []lalx.GroupData `json:"groups"`
	}
	if err := json.Unmarshal(body, &httpResp); err != nil {
		l.Logger.Errorf("解析响应JSON失败: %v, 响应内容: %s", err, string(body))
		return nil, fmt.Errorf("解析响应JSON失败: %w", err)
	}

	groups := make([]*lalproxy.GroupData, 0, len(httpResp.Groups))
	// 遍历所有GroupData，转换为PB对象
	for _, item := range httpResp.Groups {
		v := &lalproxy.GroupData{}
		err = copier.Copy(v, item)
		if err != nil {
			l.Logger.Errorf("转换GroupData失败: %v", err)
			continue
		}
		groups = append(groups, v)
	}
	pbRes := &lalproxy.GetAllGroupsRes{
		ErrorCode: int32(httpResp.ErrorCode),
		Desp:      httpResp.Desp,
		Groups:    groups,
	}

	return pbRes, nil
}
