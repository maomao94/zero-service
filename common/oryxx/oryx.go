package oryxx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// Config 表示 Oryx 服务连接配置。
type Config struct {
	// Oryx 服务器 IP 地址
	Ip string
	// Oryx 服务器 HTTP API 端口
	Port int
	// 超时时间（毫秒）
	Timeout int
	// API Secret，即 Oryx 的 SRS_PLATFORM_SECRET，用于 Bearer 鉴权
	Secret string
}

// Client 是 Oryx HTTP API 客户端（SDK），统一携带 Bearer 鉴权与基础 URL，
// 并提供各业务属性的类型化方法，供上层服务直接调用。
type Client struct {
	baseUrl string
	httpc   httpc.Service
}

// NewClient 根据配置创建 Oryx 客户端。
func NewClient(c Config) *Client {
	timeout := time.Duration(c.Timeout) * time.Millisecond
	httpClient := &http.Client{Timeout: timeout}
	service := httpc.NewServiceWithClient("httpc-oryx", httpClient, func(r *http.Request) *http.Request {
		if c.Secret != "" {
			r.Header.Set("Authorization", "Bearer "+c.Secret)
		}
		return r
	})
	return &Client{
		baseUrl: fmt.Sprintf("http://%s:%d", c.Ip, c.Port),
		httpc:   service,
	}
}

// Do 调用 Oryx HTTP API，apiPath 为相对路径（如 /terraform/v1/mgmt/versions）。
// body 为 nil 时不发送请求体；否则序列化为 JSON 请求体（必须为 struct，go-zero mapping 仅支持 struct）。
func (c *Client) Do(ctx context.Context, method, apiPath string, body any) (*http.Response, error) {
	return c.httpc.Do(ctx, method, c.baseUrl+apiPath, body)
}

// BaseUrl 返回 Oryx 基础 URL。
func (c *Client) BaseUrl() string {
	return c.baseUrl
}

// call 调用 Oryx HTTP API，解析统一响应，返回 data（原始 JSON 字节）。
// Oryx 业务错误（200 且 code != 0 或非 200 且响应为 JSON {code,data}）返回 *OryxError；
// 网络/协议错误、非 200 且响应为文本返回普通 error。
func (c *Client) call(ctx context.Context, method, apiPath string, body any) (data json.RawMessage, err error) {
	resp, err := c.httpc.Do(ctx, method, c.baseUrl+apiPath, body)
	if err != nil {
		return nil, fmt.Errorf("调用 Oryx API 失败: %w", err)
	}
	defer resp.Body.Close()

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	var httpResp struct {
		// 业务码，0 表示成功
		Code int `json:"code"`
		// 数据体（原始 JSON）
		Data json.RawMessage `json:"data"`
	}
	marshalErr := json.Unmarshal(rb, &httpResp)

	// 非 200：Oryx 业务错误常态（如 HTTP 500 + 错误信息）。
	// 响应为 JSON {code,data} 时优先取业务码；否则用 HTTP 状态码作为 code，错误信息作为 Message。
	if resp.StatusCode != http.StatusOK {
		if marshalErr == nil && httpResp.Code != 0 {
			return nil, &OryxError{Code: int32(httpResp.Code), Message: rawMessageString(httpResp.Data)}
		}
		return nil, &OryxError{Code: int32(resp.StatusCode), Message: strings.TrimSpace(string(rb))}
	}

	// 200：正常 JSON {code, data}，code != 0 视为业务错误。
	if marshalErr != nil {
		return nil, fmt.Errorf("解析响应 JSON 失败: %w", marshalErr)
	}
	if httpResp.Code != 0 {
		return nil, &OryxError{Code: int32(httpResp.Code), Message: rawMessageString(httpResp.Data)}
	}
	return httpResp.Data, nil
}

// rawMessageString 将 json.RawMessage 转为字符串；非字符串或空返回 ""。
func rawMessageString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Versions 查询版本信息（GET /terraform/v1/mgmt/versions）。
func (c *Client) Versions(ctx context.Context) (version string, err error) {
	data, err := c.call(ctx, http.MethodGet, "/terraform/v1/mgmt/versions", nil)
	if err != nil {
		return "", err
	}
	var d struct {
		// 版本号（去 v 前缀）
		Version string `json:"version"`
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &d); err != nil {
			return "", fmt.Errorf("解析版本信息失败: %w", err)
		}
	}
	return d.Version, nil
}

// RecordQuery 查询录制配置（POST /terraform/v1/hooks/record/query）。
func (c *Client) RecordQuery(ctx context.Context) (data RecordQueryData, err error) {
	raw, err := c.call(ctx, http.MethodPost, "/terraform/v1/hooks/record/query", nil)
	if err != nil {
		return data, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &data); err != nil {
			return data, fmt.Errorf("解析录制配置失败: %w", err)
		}
	}
	return data, nil
}

// RecordApply 应用录制配置（POST /terraform/v1/hooks/record/apply）。
func (c *Client) RecordApply(ctx context.Context, all bool) error {
	_, err := c.call(ctx, http.MethodPost, "/terraform/v1/hooks/record/apply",
		struct {
			// 是否录制所有流
			All bool `json:"all"`
		}{All: all})
	return err
}

// RecordEnd 结束录制任务（POST /terraform/v1/hooks/record/end）。
func (c *Client) RecordEnd(ctx context.Context, uuid string) error {
	_, err := c.call(ctx, http.MethodPost, "/terraform/v1/hooks/record/end",
		struct {
			// 录制任务 UUID
			UUID string `json:"uuid"`
		}{UUID: uuid})
	return err
}

// RecordRemove 删除录制文件（POST /terraform/v1/hooks/record/remove）。
func (c *Client) RecordRemove(ctx context.Context, uuid string) error {
	_, err := c.call(ctx, http.MethodPost, "/terraform/v1/hooks/record/remove",
		struct {
			// 录制任务 UUID
			UUID string `json:"uuid"`
		}{UUID: uuid})
	return err
}

// RecordFiles 列出录制文件（POST /terraform/v1/hooks/record/files）。
func (c *Client) RecordFiles(ctx context.Context) (files []RecordFile, err error) {
	raw, err := c.call(ctx, http.MethodPost, "/terraform/v1/hooks/record/files", nil)
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &files); err != nil {
			return nil, fmt.Errorf("解析录制文件列表失败: %w", err)
		}
	}
	return files, nil
}

// HooksQuery 查询回调配置（POST /terraform/v1/mgmt/hooks/query）。
func (c *Client) HooksQuery(ctx context.Context) (data HooksQueryData, err error) {
	raw, err := c.call(ctx, http.MethodPost, "/terraform/v1/mgmt/hooks/query", nil)
	if err != nil {
		return data, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &data); err != nil {
			return data, fmt.Errorf("解析回调配置失败: %w", err)
		}
	}
	return data, nil
}

// HooksApply 应用回调配置（POST /terraform/v1/mgmt/hooks/apply）。
func (c *Client) HooksApply(ctx context.Context, req HooksApplyReq) error {
	_, err := c.call(ctx, http.MethodPost, "/terraform/v1/mgmt/hooks/apply", req)
	return err
}

// RecordGlobs 更新录制 glob 过滤器（POST /terraform/v1/hooks/record/globs）。
func (c *Client) RecordGlobs(ctx context.Context, globs []string) error {
	_, err := c.call(ctx, http.MethodPost, "/terraform/v1/hooks/record/globs",
		struct {
			// glob 过滤器
			Globs []string `json:"globs"`
		}{Globs: globs})
	return err
}

// RecordPostProcessing 更新录制后处理配置（POST /terraform/v1/hooks/record/post-processing）。
func (c *Client) RecordPostProcessing(ctx context.Context, postProcess, postCpDir string) error {
	_, err := c.call(ctx, http.MethodPost, "/terraform/v1/hooks/record/post-processing",
		struct {
			// 后处理类型，固定为 post-cp-file
			PostProcess string `json:"postProcess"`
			// 后处理拷贝目录
			PostCpDir string `json:"postCpDir"`
		}{PostProcess: postProcess, PostCpDir: postCpDir})
	return err
}

// DvrQuery 查询 DVR 云录制配置（POST /terraform/v1/hooks/dvr/query）。
func (c *Client) DvrQuery(ctx context.Context) (data DvrQueryData, err error) {
	raw, err := c.call(ctx, http.MethodPost, "/terraform/v1/hooks/dvr/query", nil)
	if err != nil {
		return data, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &data); err != nil {
			return data, fmt.Errorf("解析 DVR 录制配置失败: %w", err)
		}
	}
	return data, nil
}

// DvrApply 应用 DVR 云录制配置（POST /terraform/v1/hooks/dvr/apply）。
func (c *Client) DvrApply(ctx context.Context, all bool) error {
	_, err := c.call(ctx, http.MethodPost, "/terraform/v1/hooks/dvr/apply",
		struct {
			// 是否录制所有流
			All bool `json:"all"`
		}{All: all})
	return err
}

// DvrFiles 列出 DVR 云录制文件（POST /terraform/v1/hooks/dvr/files）。
func (c *Client) DvrFiles(ctx context.Context) (files []DvrFile, err error) {
	raw, err := c.call(ctx, http.MethodPost, "/terraform/v1/hooks/dvr/files", nil)
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &files); err != nil {
			return nil, fmt.Errorf("解析 DVR 录制文件列表失败: %w", err)
		}
	}
	return files, nil
}
