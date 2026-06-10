package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
)

// StatsEntry 实时资源统计。
type StatsEntry struct {
	Timestamp  time.Time
	CPUPercent float64
	MemUsage   uint64 // bytes
	MemLimit   uint64 // bytes
	MemPercent float64
	NetRx      uint64 // bytes
	NetTx      uint64 // bytes
	BlockRead  uint64 // bytes
	BlockWrite uint64 // bytes
	PIDs       uint64
}

// StreamStats 流式获取容器实时资源统计。
func (c *Client) StreamStats(id string) (<-chan StatsEntry, <-chan error) {
	statsCh := make(chan StatsEntry, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(statsCh)

		ctx, cancel := context.WithCancel(c.ctx)
		defer cancel()

		resp, err := c.cli.ContainerStats(ctx, id, true)
		if err != nil {
			errCh <- fmt.Errorf("获取容器 stats 失败: %w", err)
			return
		}
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)

		// 用于计算 CPU 百分比的上一帧数据
		var prevCPU, prevSystem uint64

		for {
			var v container.StatsResponse
			if err := decoder.Decode(&v); err != nil {
				if err == io.EOF {
					return
				}
				errCh <- err
				return
			}

			entry := parseStats(&v, &prevCPU, &prevSystem)
			select {
			case statsCh <- entry:
			case <-ctx.Done():
				return
			}
		}
	}()

	return statsCh, errCh
}

// parseStats 解析 Docker stats JSON 为 StatsEntry。
func parseStats(v *container.StatsResponse, prevCPU *uint64, prevSystem *uint64) StatsEntry {
	entry := StatsEntry{Timestamp: time.Now()}

	// CPU 百分比计算
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - *prevCPU)
	systemDelta := float64(v.CPUStats.SystemUsage - *prevSystem)
	*prevCPU = v.CPUStats.CPUUsage.TotalUsage
	*prevSystem = v.CPUStats.SystemUsage

	if systemDelta > 0 && cpuDelta > 0 {
		numCPUs := float64(v.CPUStats.OnlineCPUs)
		if numCPUs == 0 {
			numCPUs = float64(len(v.CPUStats.CPUUsage.PercpuUsage))
		}
		if numCPUs == 0 {
			numCPUs = 1
		}
		entry.CPUPercent = (cpuDelta / systemDelta) * numCPUs * 100
	}

	// 内存
	entry.MemUsage = v.MemoryStats.Usage
	entry.MemLimit = v.MemoryStats.Limit
	if entry.MemLimit > 0 {
		entry.MemPercent = float64(entry.MemUsage) / float64(entry.MemLimit) * 100
	}

	// 网络 IO
	for _, net := range v.Networks {
		entry.NetRx += net.RxBytes
		entry.NetTx += net.TxBytes
	}

	// 磁盘 IO
	for _, bio := range v.BlkioStats.IoServiceBytesRecursive {
		switch bio.Op {
		case "read":
			entry.BlockRead += bio.Value
		case "write":
			entry.BlockWrite += bio.Value
		}
	}

	// 进程数
	entry.PIDs = v.PidsStats.Current

	return entry
}
