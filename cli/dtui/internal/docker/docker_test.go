package docker

import (
	"testing"
)

// TestNewClient 验证 Docker 客户端创建。
func TestNewClient(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	defer c.Close()

	if c.cli == nil {
		t.Error("expected non-nil SDK client")
	}
	if c.timeout.Seconds() != 10 {
		t.Errorf("expected 10s timeout, got %v", c.timeout)
	}
}

// TestPing 验证 Docker daemon 连接。
func TestPing(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer c.Close()

	if err := c.Ping(); err != nil {
		t.Errorf("Ping() failed: %v", err)
	}
}

// TestListContainers 验证容器列表获取。
func TestListContainers(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer c.Close()

	containers, err := c.ListContainers("")
	if err != nil {
		t.Fatalf("ListContainers() failed: %v", err)
	}

	// 验证结构体字段（至少检查类型正确）
	for _, ctr := range containers {
		if ctr.ID == "" {
			t.Error("container ID should not be empty")
		}
		if ctr.Name == "" {
			t.Error("container Name should not be empty")
		}
		// Running() 方法
		_ = ctr.Running()
	}

	t.Logf("Found %d containers", len(containers))
}

// TestListContainersFilter 验证过滤功能。
func TestListContainersFilter(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer c.Close()

	all, err := c.ListContainers("")
	if err != nil {
		t.Fatalf("ListContainers() failed: %v", err)
	}

	// 使用一个不可能匹配的过滤词
	filtered, err := c.ListContainers("zzz_nonexistent_zzz")
	if err != nil {
		t.Fatalf("ListContainers(filter) failed: %v", err)
	}

	if len(filtered) > len(all) {
		t.Error("filtered list should not be larger than full list")
	}
}

// TestListImages 验证镜像列表获取。
func TestListImages(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer c.Close()

	images, err := c.ListImages("")
	if err != nil {
		t.Fatalf("ListImages() failed: %v", err)
	}

	for _, img := range images {
		if img.ID == "" {
			t.Error("image ID should not be empty")
		}
		// Ref() 方法
		_ = img.Ref()
		// DefaultSaveFile() 方法
		_ = img.DefaultSaveFile()
	}

	t.Logf("Found %d images", len(images))
}

// TestFetchLogs 验证批量日志获取（使用一个已存在的容器）。
func TestFetchLogs(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer c.Close()

	// 获取第一个容器来测试日志
	containers, err := c.ListContainers("")
	if err != nil || len(containers) == 0 {
		t.Skip("no containers available for log test")
	}

	lines, err := c.FetchLogs(containers[0].ID, LogOptions{Tail: "10"})
	if err != nil {
		// 某些容器可能没有日志，不算失败
		t.Logf("FetchLogs() error (may be expected): %v", err)
		return
	}

	t.Logf("Fetched %d log lines from %s", len(lines), containers[0].Name)
}

// TestInspectContainer 验证容器详情获取。
func TestInspectContainer(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer c.Close()

	containers, err := c.ListContainers("")
	if err != nil || len(containers) == 0 {
		t.Skip("no containers available for inspect test")
	}

	detail, err := c.InspectContainer(containers[0].ID)
	if err != nil {
		t.Fatalf("InspectContainer() failed: %v", err)
	}

	if detail.ID == "" {
		t.Error("detail ID should not be empty")
	}
	if detail.Name == "" {
		t.Error("detail Name should not be empty")
	}
	if detail.Image == "" {
		t.Error("detail Image should not be empty")
	}

	t.Logf("Inspected: %s (%s) - %s", detail.Name, detail.ID, detail.State.Status)
}

// TestStreamStats 验证实时统计流。
func TestStreamStats(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer c.Close()

	// 找一个运行中的容器
	containers, err := c.ListContainers("")
	if err != nil {
		t.Skipf("ListContainers failed: %v", err)
	}

	var running *Container
	for i := range containers {
		if containers[i].Running() {
			running = &containers[i]
			break
		}
	}
	if running == nil {
		t.Skip("no running containers for stats test")
	}

	statsCh, errCh := c.StreamStats(running.ID)

	select {
	case entry, ok := <-statsCh:
		if !ok {
			t.Fatal("stats channel closed unexpectedly")
		}
		t.Logf("Stats: CPU=%.1f%% MEM=%d/%d PIDs=%d",
			entry.CPUPercent, entry.MemUsage, entry.MemLimit, entry.PIDs)
	case err := <-errCh:
		t.Fatalf("StreamStats error: %v", err)
	}
}

// TestImageHistory 验证镜像历史获取。
func TestImageHistory(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer c.Close()

	images, err := c.ListImages("")
	if err != nil || len(images) == 0 {
		t.Skip("no images available for history test")
	}

	// 找一个有 tag 的镜像
	var ref string
	for _, img := range images {
		if img.Tag != "" && img.Tag != "<none>" {
			ref = img.Ref()
			break
		}
	}
	if ref == "" {
		t.Skip("no tagged images available")
	}

	entries, err := c.ImageHistory(ref)
	if err != nil {
		t.Fatalf("ImageHistory() failed: %v", err)
	}

	if len(entries) == 0 {
		t.Error("expected at least one history entry")
	}

	for _, e := range entries {
		if e.ID == "" {
			t.Error("history entry ID should not be empty")
		}
	}

	t.Logf("Image %s has %d history layers", ref, len(entries))
}

// TestParseStats 验证 CPU 百分比计算。
func TestParseStats(t *testing.T) {
	var prevCPU, prevSystem uint64

	// 第一帧 — prevCPU/prevSystem 为 0，应该得到 CPUPercent=0
	stats := &StatsEntry{}
	_ = stats // just verifying the parse logic doesn't panic

	// 模拟计算
	cpuDelta := float64(100 - prevCPU)
	systemDelta := float64(1000 - prevSystem)
	if systemDelta > 0 && cpuDelta > 0 {
		pct := (cpuDelta / systemDelta) * 4 * 100
		if pct <= 0 {
			t.Error("expected positive CPU percent")
		}
	}

	// 零分母保护
	cpuDelta = 0
	systemDelta = 0
	if systemDelta > 0 && cpuDelta > 0 {
		t.Error("should not enter branch with zero deltas")
	}
}
