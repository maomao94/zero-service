package oryxx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// callSrs 调用 Oryx 代理的 SRS HTTP API（GET /api/v1/*）。
// SRS 响应顶层结构因端点而异（data / streams / clients / vhosts），由 out 按实际结构解析；
// 除 /api/v1/versions 外 Oryx 要求 Bearer 鉴权（NewClient 已统一注入），与 /terraform/v1/* 一致。
func (c *Client) callSrs(ctx context.Context, apiPath string, out any) error {
	resp, err := c.httpc.Do(ctx, http.MethodGet, c.baseUrl+apiPath, nil)
	if err != nil {
		return fmt.Errorf("调用 SRS API 失败: %w", err)
	}
	defer resp.Body.Close()

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应体失败: %w", err)
	}

	// SRS 错误为 HTTP 200 + {code: 非0}（如 1061 raw 未启用、资源未找到）或 Oryx 401。
	var httpResp struct {
		// SRS 业务码，0 表示成功
		Code int `json:"code"`
	}
	marshalErr := json.Unmarshal(rb, &httpResp)
	if resp.StatusCode != http.StatusOK {
		return &OryxError{Code: int32(resp.StatusCode), Message: string(rb)}
	}
	if marshalErr != nil {
		return fmt.Errorf("解析 SRS 响应 JSON 失败: %w", marshalErr)
	}
	if httpResp.Code != 0 {
		return &OryxError{Code: int32(httpResp.Code), Message: "SRS API 业务错误"}
	}
	if out != nil {
		if err := json.Unmarshal(rb, out); err != nil {
			return fmt.Errorf("解析 SRS 响应失败: %w", err)
		}
	}
	return nil
}

// srsRespMeta SRS API 通用响应头（server 标识部署实例，重启变化，可作缓存失效判断）。
type srsRespMeta struct {
	// SRS 实例标识（重启后变化，可作缓存失效判断）
	Server string `json:"server"`
	// 服务标识
	Service string `json:"service"`
	// 进程标识
	Pid string `json:"pid"`
}

// SrsVersionsData SRS 版本信息（/api/v1/versions 的 data）
type SrsVersionsData struct {
	// 主版本
	Major int32 `json:"major"`
	// 次版本
	Minor int32 `json:"minor"`
	// 修订号
	Revision int32 `json:"revision"`
	// 完整版本字符串，如 "6.0.184"
	Version string `json:"version"`
}

// SrsVersions 查询 SRS 版本信息（GET /api/v1/versions，免鉴权）。
func (c *Client) SrsVersions(ctx context.Context) (SrsVersionsData, error) {
	var resp struct {
		// 版本数据
		Data SrsVersionsData `json:"data"`
	}
	if err := c.callSrs(ctx, "/api/v1/versions", &resp); err != nil {
		return SrsVersionsData{}, err
	}
	return resp.Data, nil
}

// SrsStreamsData SRS 流列表响应（/api/v1/streams）
type SrsStreamsData struct {
	srsRespMeta
	// 流列表
	Streams []SrsStream `json:"streams"`
}

// SrsStreams 查询 SRS 流列表（GET /api/v1/streams?start=&count=；count 缺失/小于 10 时 SRS 返回 10）
func (c *Client) SrsStreams(ctx context.Context, start, count int32) (SrsStreamsData, error) {
	path := srsPagingPath("/api/v1/streams/", start, count)
	var resp SrsStreamsData
	if err := c.callSrs(ctx, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// SrsClientsData SRS 客户端列表响应（/api/v1/clients）
type SrsClientsData struct {
	srsRespMeta
	// 客户端列表
	Clients []SrsClient `json:"clients"`
}

// SrsClients 查询 SRS 客户端列表（GET /api/v1/clients?start=&count=）
func (c *Client) SrsClients(ctx context.Context, start, count int32) (SrsClientsData, error) {
	path := srsPagingPath("/api/v1/clients/", start, count)
	var resp SrsClientsData
	if err := c.callSrs(ctx, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// SrsVhostsData SRS 虚拟主机列表响应（/api/v1/vhosts；无分页全量）
type SrsVhostsData struct {
	srsRespMeta
	// 虚拟主机列表
	Vhosts []SrsVhost `json:"vhosts"`
}

// SrsVhosts 查询 SRS 虚拟主机列表（GET /api/v1/vhosts/）
func (c *Client) SrsVhosts(ctx context.Context) (SrsVhostsData, error) {
	var resp SrsVhostsData
	if err := c.callSrs(ctx, "/api/v1/vhosts/", &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// SrsSummariesData SRS 系统状态（/api/v1/summaries）
type SrsSummariesData struct {
	// 采样是否成功
	OK bool `json:"ok"`
	// 采样时间（Unix 毫秒）
	NowMs float64 `json:"now_ms"`
	// 进程自身状态
	Self SrsSelfInfo `json:"self"`
	// 系统整体状态
	System SrsSystem `json:"system"`
}

// SrsSummaries 查询 SRS 系统状态（GET /api/v1/summaries）
func (c *Client) SrsSummaries(ctx context.Context) (SrsSummariesData, error) {
	var resp struct {
		// 系统状态数据
		Data SrsSummariesData `json:"data"`
	}
	if err := c.callSrs(ctx, "/api/v1/summaries", &resp); err != nil {
		return SrsSummariesData{}, err
	}
	return resp.Data, nil
}

// SrsRequestsData SRS 最近请求（/api/v1/tests/requests，调试用）
type SrsRequestsData struct {
	// 请求 uri
	URI string `json:"uri"`
	// 请求路径
	Path string `json:"path"`
	// 请求方法（SRS 字段名大写 METHOD）
	Method string `json:"METHOD"`
}

// SrsRequests 查询 SRS 最近的请求（GET /api/v1/tests/requests）
func (c *Client) SrsRequests(ctx context.Context) (SrsRequestsData, error) {
	var resp struct {
		// 请求数据
		Data SrsRequestsData `json:"data"`
	}
	if err := c.callSrs(ctx, "/api/v1/tests/requests", &resp); err != nil {
		return SrsRequestsData{}, err
	}
	return resp.Data, nil
}

// srsPagingPath 拼接带分页参数路径；参数为非正值时不携带，交由 SRS 默认值（start=0, count=10）。
func srsPagingPath(apiPath string, start, count int32) string {
	query := ""
	if start > 0 {
		query = fmt.Sprintf("?start=%d", start)
	}
	if count > 0 {
		if query == "" {
			query = fmt.Sprintf("?count=%d", count)
		} else {
			query = fmt.Sprintf("%s&count=%d", query, count)
		}
	}
	return apiPath + query
}
