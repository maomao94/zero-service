package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"zero-service/app/podengine/internal/svc"
	"zero-service/app/podengine/podengine"
	"zero-service/common/tool"

	"github.com/docker/docker/api/types/container"
	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetPodStatsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPodStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPodStatsLogic {
	return &GetPodStatsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetPodStatsLogic) GetPodStats(in *podengine.GetPodStatsReq) (*podengine.GetPodStatsRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	dockerClient, ok := l.svcCtx.GetDockerClient(in.Node)
	if !ok {
		return nil, fmt.Errorf("node %s not found", in.Node)
	}

	// Inspect container to get basic info
	containerInfo, err := dockerClient.ContainerInspect(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}

	// Get container stats
	stats, err := dockerClient.ContainerStats(l.ctx, in.Id, false) // false = non-stream
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var v container.StatsResponse
	err = json.NewDecoder(stats.Body).Decode(&v)
	if err != nil {
		return nil, err
	}

	// Calculate CPU usage percent
	var cpuUsagePercent float64
	if v.CPUStats.CPUUsage.TotalUsage > 0 && v.PreCPUStats.CPUUsage.TotalUsage > 0 && v.PreCPUStats.SystemUsage > 0 {
		cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
		if systemDelta > 0 {
			if cpuDelta > 0 {
				cpuUsagePercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100
			} else {
				// cpuDelta is 0, use a small non-zero value to avoid division by zero
				cpuUsagePercent = 0.01 // Very small CPU usage
			}
		}
	} else if v.CPUStats.CPUUsage.TotalUsage > 0 && v.CPUStats.SystemUsage > 0 {
		// Fallback approach: using absolute values
		cpuUsagePercent = (float64(v.CPUStats.CPUUsage.TotalUsage) / float64(v.CPUStats.SystemUsage)) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100
	} else {
		// Insufficient data, use a reasonable default
		cpuUsagePercent = 0.01 // Very small CPU usage
	}

	// Calculate memory usage percent
	var memoryUsagePercent float64
	if v.MemoryStats.Limit > 0 {
		memoryUsagePercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100
	}

	// Get network stats
	var networkRxBytes, networkTxBytes uint64
	for _, netStats := range v.Networks {
		networkRxBytes += netStats.RxBytes
		networkTxBytes += netStats.TxBytes
	}

	// Get storage stats
	var storageReadBytes, storageWriteBytes uint64
	for _, blkioStat := range v.BlkioStats.IoServiceBytesRecursive {
		if blkioStat.Op == "read" {
			storageReadBytes += blkioStat.Value
		} else if blkioStat.Op == "write" {
			storageWriteBytes += blkioStat.Value
		}
	}

	// Create container stats with real info
	containerStats := &podengine.ContainerStats{
		ContainerId:               containerInfo.ID,
		ContainerName:             containerInfo.Name[1:], // Remove leading slash
		CpuUsagePercent:           cpuUsagePercent,
		CpuUsagePercentDisplay:    fmt.Sprintf("%.2f%%", cpuUsagePercent),
		CpuUsageTotal:             int64(v.CPUStats.CPUUsage.TotalUsage),
		CpuUsageTotalDisplay:      fmt.Sprintf("%d ns", v.CPUStats.CPUUsage.TotalUsage),
		MemoryUsage:               int64(v.MemoryStats.Usage),
		MemoryUsageDisplay:        tool.BinaryBytes(int64(v.MemoryStats.Usage), 2),
		MemoryLimit:               int64(v.MemoryStats.Limit),
		MemoryLimitDisplay:        tool.BinaryBytes(int64(v.MemoryStats.Limit), 2),
		MemoryUsagePercent:        memoryUsagePercent,
		MemoryUsagePercentDisplay: fmt.Sprintf("%.2f%%", memoryUsagePercent),
		NetworkRxBytes:            int64(networkRxBytes),
		NetworkRxBytesDisplay:     tool.DecimalBytes(int64(networkRxBytes), 2),
		NetworkTxBytes:            int64(networkTxBytes),
		NetworkTxBytesDisplay:     tool.DecimalBytes(int64(networkTxBytes), 2),
		StorageReadBytes:          int64(storageReadBytes),
		StorageReadBytesDisplay:   tool.DecimalBytes(int64(storageReadBytes), 2),
		StorageWriteBytes:         int64(storageWriteBytes),
		StorageWriteBytesDisplay:  tool.DecimalBytes(int64(storageWriteBytes), 2),
		Timestamp:                 carbon.NewCarbon(time.Now()).ToDateTimeString(),
	}

	return &podengine.GetPodStatsRes{
		Stats: []*podengine.ContainerStats{containerStats},
	}, nil
}
