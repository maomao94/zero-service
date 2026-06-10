package docker

import (
	"fmt"
	"strings"
	"time"
)

// ContainerDetail 容器完整详情。
type ContainerDetail struct {
	ID            string
	Name          string
	Image         string
	Platform      string
	Created       string
	State         ContainerState
	Mounts        []MountInfo
	Network       []NetworkInfo
	Ports         []PortBinding
	Env           []string
	Cmd           []string
	Entrypoint    []string
	WorkingDir    string
	RestartPolicy string
}

// ContainerState 容器运行状态。
type ContainerState struct {
	Status     string
	Running    bool
	Paused     bool
	Restarting bool
	StartedAt  string
	FinishedAt string
	ExitCode   int
	Error      string
}

// MountInfo 挂载信息。
type MountInfo struct {
	Type        string // bind / volume / tmpfs
	Source      string
	Destination string
	Mode        string // rw / ro
}

// NetworkInfo 网络信息。
type NetworkInfo struct {
	Name       string
	IPAddress  string
	Gateway    string
	MacAddress string
}

// PortBinding 端口映射。
type PortBinding struct {
	ContainerPort string
	HostPort      string
	HostIP        string
	Protocol      string
}

// InspectContainer 获取容器完整详情。
func (c *Client) InspectContainer(id string) (*ContainerDetail, error) {
	ctx, cancel := c.withTimeout()
	defer cancel()

	info, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("查询容器详情失败: %w", err)
	}

	detail := &ContainerDetail{
		ID:       info.ID[:12],
		Name:     strings.TrimPrefix(info.Name, "/"),
		Image:    info.Config.Image,
		Platform: info.Platform,
		Created:  parseDockerTime(info.Created),
		State: ContainerState{
			Status:     info.State.Status,
			Running:    info.State.Running,
			Paused:     info.State.Paused,
			Restarting: info.State.Restarting,
			StartedAt:  parseDockerTime(info.State.StartedAt),
			FinishedAt: parseDockerTime(info.State.FinishedAt),
			ExitCode:   info.State.ExitCode,
			Error:      info.State.Error,
		},
		Env:           info.Config.Env,
		Cmd:           info.Config.Cmd,
		Entrypoint:    info.Config.Entrypoint,
		WorkingDir:    info.Config.WorkingDir,
		RestartPolicy: string(info.HostConfig.RestartPolicy.Name),
	}

	// 挂载卷
	for _, m := range info.Mounts {
		detail.Mounts = append(detail.Mounts, MountInfo{
			Type:        string(m.Type),
			Source:      m.Source,
			Destination: m.Destination,
			Mode:        m.Mode,
		})
	}

	// 网络
	for name, net := range info.NetworkSettings.Networks {
		detail.Network = append(detail.Network, NetworkInfo{
			Name:       name,
			IPAddress:  net.IPAddress,
			Gateway:    net.Gateway,
			MacAddress: net.MacAddress,
		})
	}

	// 端口映射
	for containerPort, hostBindings := range info.HostConfig.PortBindings {
		for _, bind := range hostBindings {
			detail.Ports = append(detail.Ports, PortBinding{
				ContainerPort: string(containerPort),
				HostPort:      bind.HostPort,
				HostIP:        bind.HostIP,
			})
		}
	}
	if len(detail.Ports) == 0 {
		for p := range info.Config.ExposedPorts {
			detail.Ports = append(detail.Ports, PortBinding{
				ContainerPort: string(p),
			})
		}
	}

	return detail, nil
}

// parseDockerTime 解析 Docker API 返回的时间字符串。
func parseDockerTime(s string) string {
	if s == "" {
		return "-"
	}
	// Docker API 返回 RFC3339Nano 格式
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02 15:04:05")
}
