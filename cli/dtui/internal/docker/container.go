package docker

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// Container 代表一个容器记录。
type Container struct {
	ID      string
	Image   string
	Command string
	Created string
	Status  string
	Ports   string
	Name    string
	State   string // running, exited, paused 等
}

// Running 判断容器是否处于运行状态。
func (c Container) Running() bool {
	return strings.EqualFold(c.State, "running")
}

// ListContainers 通过 SDK 的 ContainerList 获取容器列表。
func (c *Client) ListContainers(filter string) ([]Container, error) {
	ctx, cancel := c.withTimeout()
	defer cancel()

	listFilters := filters.NewArgs()
	listFilters.Add("status", "running")
	listFilters.Add("status", "paused")
	listFilters.Add("status", "exited")
	listFilters.Add("status", "created")

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: listFilters,
	})
	if err != nil {
		return nil, fmt.Errorf("查询容器列表失败: %w", err)
	}

	var result []Container
	for _, ctr := range containers {
		// 提取端口映射
		var ports []string
		for _, p := range ctr.Ports {
			if p.PublicPort > 0 {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", p.PublicPort, p.PrivatePort, p.Type))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
			}
		}

		// 从 Names 中取第一个（去掉 / 前缀）
		name := ""
		if len(ctr.Names) > 0 {
			name = strings.TrimPrefix(ctr.Names[0], "/")
		}

		createdTime := time.Unix(ctr.Created, 0).Format("2006-01-02 15:04:05")

		ct := Container{
			ID:      ctr.ID,
			Image:   ctr.Image,
			Command: ctr.Command,
			Created: createdTime,
			Status:  ctr.Status,
			Ports:   strings.Join(ports, ", "),
			Name:    name,
			State:   ctr.State,
		}

		if filter == "" ||
			strings.Contains(strings.ToLower(ct.Name), strings.ToLower(filter)) ||
			strings.Contains(strings.ToLower(ct.Image), strings.ToLower(filter)) {
			result = append(result, ct)
		}
	}
	return result, nil
}

// StartContainer 启动容器。
func (c *Client) StartContainer(id string) error {
	ctx, cancel := c.withTimeout()
	defer cancel()
	return c.cli.ContainerStart(ctx, id, container.StartOptions{})
}

// StopContainer 停止容器。
func (c *Client) StopContainer(id string) error {
	ctx, cancel := c.withTimeout()
	defer cancel()
	stopTimeout := 10
	return c.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &stopTimeout})
}

// RestartContainer 重启容器。
func (c *Client) RestartContainer(id string) error {
	ctx, cancel := c.withTimeout()
	defer cancel()
	stopTimeout := 10
	return c.cli.ContainerRestart(ctx, id, container.StopOptions{Timeout: &stopTimeout})
}

// RemoveContainer 删除容器。
func (c *Client) RemoveContainer(id string, force bool) error {
	ctx, cancel := c.withTimeout()
	defer cancel()
	return c.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: force})
}
