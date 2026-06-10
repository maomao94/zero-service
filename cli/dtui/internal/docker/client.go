package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
)

// Client 封装 Docker SDK 客户端，提供本工具所需的所有 Docker API。
type Client struct {
	cli     *client.Client
	ctx     context.Context
	cancel  context.CancelFunc
	timeout time.Duration
}

// NewClient 创建 Docker 客户端。
// 默认从环境变量 DOCKER_HOST / unix socket 自动发现。
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("创建 Docker 客户端失败: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		cli:     cli,
		ctx:     ctx,
		cancel:  cancel,
		timeout: 10 * time.Second,
	}, nil
}

// Ping 检查 Docker daemon 连接。
func (c *Client) Ping() error {
	ctx, cancel := context.WithTimeout(c.ctx, 3*time.Second)
	defer cancel()
	_, err := c.cli.Ping(ctx)
	return err
}

// Close 取消后台上下文并释放连接。
func (c *Client) Close() error {
	c.cancel()
	return c.cli.Close()
}

// withTimeout 创建带超时的 context（默认 10s）。
func (c *Client) withTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.ctx, c.timeout)
}

// withLongTimeout 创建长时间超时的 context（5min），用于 image save 等耗时操作。
func (c *Client) withLongTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.ctx, 5*time.Minute)
}

// RawClient 暴露原始 SDK Client，供特殊操作使用。
func (c *Client) RawClient() *client.Client {
	return c.cli
}
