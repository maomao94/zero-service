package isp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
)

// ReportCategory 对应 ISP 协议的 messageId，即 (Type<<16)|Command。
// 值直接使用 common/isp 包的 MessageID 常量，与协议报文一一对应。
type ReportCategory int

const (
	ReportCategoryPatrolDeviceRunData     ReportCategory = ReportCategory(isp.MessageIDPatrolDeviceRunData)     // 2-0 巡视设备运行数据
	ReportCategoryPatrolDeviceStatusData  ReportCategory = ReportCategory(isp.MessageIDPatrolDeviceStatusData)  // 1-0 巡视设备状态数据
	ReportCategoryPatrolDeviceCoordinates ReportCategory = ReportCategory(isp.MessageIDPatrolDeviceCoordinates) // 3-0 巡视设备坐标
)

// 默认上报间隔。
// 运行数据、状态数据默认 1 分钟；坐标/经纬度要求更频繁，默认 2 秒。
const (
	defaultReportInterval = time.Minute
	defaultCoordInterval  = 2 * time.Second
)

// 每个上报类别在缓存中构建唯一 Item key 时使用的属性列表。
// 例如运行数据用 patroldevice_code + type 区分不同设备的同一数据点。
var keyAttrsByCategory = map[ReportCategory][]string{
	ReportCategoryPatrolDeviceRunData:     {"patroldevice_code", "type"},
	ReportCategoryPatrolDeviceStatusData:  {"patroldevice_code", "type"},
	ReportCategoryPatrolDeviceCoordinates: {"patroldevice_code"},
}

// cachedReport 缓存某个变电站（code）下某个上报类别的全部 item。
// lastSent 用于控制上报频率：距离上次发送不足 interval 时跳过。
type cachedReport struct {
	itemByKey map[string]*cachedItem
	lastSent  time.Time
}

// cachedItem 缓存单个上报数据点，记录原始 Item、类别和最后一次 gRPC 更新时间。
type cachedItem struct {
	item      isp.Item
	category  ReportCategory
	updatedAt time.Time
}

// reportSnapshot 单次上报的快照，包含类别、变电站编码和去重/去过期后的 item 列表。
type reportSnapshot struct {
	category ReportCategory
	code     string
	items    []isp.Item
}

// reportManager 管理所有上报类别的缓存、间隔和新鲜度策略。
//
// 并发模型：
//   - 写操作（update/applyRegistrationIntervals/markSent）持 Lock
//   - 读操作（dueReports）持 RLock
//   - setNoFreshCheck 持 Lock
//
// intervals:     每个上报类别的当前间隔（注册响应可覆盖运行数据）
// cache:         三级结构 category → code（变电站编码） → item key → cachedItem
// reservedIntervals: 注册响应中尚未分配 reportSpec 的预留间隔（nest_run_interval、weather_interval）
// noFreshCheck:  设为 true 的类别跳过新鲜度检查，始终上报缓存中的旧数据
type reportManager struct {
	mu                sync.RWMutex
	intervals         map[ReportCategory]time.Duration
	cache             map[ReportCategory]map[string]*cachedReport
	reservedIntervals map[string]time.Duration
	noFreshCheck      map[ReportCategory]bool
}

func newReportManager() *reportManager {
	r := &reportManager{
		intervals:         make(map[ReportCategory]time.Duration, len(keyAttrsByCategory)),
		cache:             make(map[ReportCategory]map[string]*cachedReport, len(keyAttrsByCategory)),
		reservedIntervals: make(map[string]time.Duration),
		noFreshCheck:      make(map[ReportCategory]bool),
	}
	for cat := range keyAttrsByCategory {
		r.intervals[cat] = defaultReportInterval
		r.cache[cat] = make(map[string]*cachedReport)
	}
	r.intervals[ReportCategoryPatrolDeviceCoordinates] = defaultCoordInterval
	r.noFreshCheck[ReportCategoryPatrolDeviceCoordinates] = true
	return r
}

// update 将 gRPC 收到的上报数据写入缓存。
// category+code 定位到具体缓存槽位，itemKey 区分同一上报里的多个数据点。
// 每次 update 会刷新 updatedAt，使该 item 重新进入新鲜期。
func (r *reportManager) update(category ReportCategory, code string, items []isp.Item, now time.Time) bool {
	keyAttrs, ok := keyAttrsByCategory[category]
	if !ok {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	report := r.cache[category][code]
	if report == nil {
		report = &cachedReport{itemByKey: make(map[string]*cachedItem)}
		r.cache[category][code] = report
	}
	for i, item := range items {
		key := itemKey(keyAttrs, item, i)
		report.itemByKey[key] = &cachedItem{item: cloneItem(item), category: category, updatedAt: now}
	}
	return true
}

// applyRegistrationIntervals 解析注册响应（251-4）中的间隔配置并应用。
// 注册响应固定只有一条 <Item>，包含 patroldevice_run_interval 等属性。
//   - patroldevice_run_interval → 直接覆盖运行数据上报间隔
//   - nest_run_interval / weather_interval → 存入 reservedIntervals 预留
//
// 解析完成后重置全部缓存的上次发送时间，使注册成功后能立即上报。
func (r *reportManager) applyRegistrationIntervals(items []isp.Item) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(items) > 0 {
		item := items[0]
		if d := parseItemInterval(item, "patroldevice_run_interval", 0); d > 0 {
			r.intervals[ReportCategoryPatrolDeviceRunData] = d
		}
		if d := parseItemInterval(item, "nest_run_interval", 0); d > 0 {
			r.reservedIntervals["nest_run_interval"] = d
		}
		if d := parseItemInterval(item, "weather_interval", 0); d > 0 {
			r.reservedIntervals["weather_interval"] = d
		}
	}

	for _, reports := range r.cache {
		for _, report := range reports {
			report.lastSent = time.Time{}
		}
	}
}

// dueReports 返回当前到期的上报快照列表。
//
// 到期判定：距离上次发送 >= category 的当前间隔（首次发送 lastSent 为零值直接通过）。
// 新鲜度判定：每个 item 的 now-updatedAt 与 freshnessTimeout(interval) 比较，
//
//	超过则视为过期，不在本次上报中包含（除非 noFreshCheck 为 true）。
//
// 持 RLock，仅在方法内 clone 数据，返回后调用方可安全使用。
func (r *reportManager) dueReports(now time.Time) []reportSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]reportSnapshot, 0, len(r.cache))
	for category, reports := range r.cache {
		interval := r.intervals[category]
		if interval <= 0 {
			continue
		}
		timeout := freshnessTimeout(interval)
		for code, report := range reports {
			if len(report.itemByKey) == 0 {
				continue
			}
			if !report.lastSent.IsZero() && now.Sub(report.lastSent) < interval {
				continue
			}
			var items []isp.Item
			if r.noFreshCheck[category] {
				items = cloneAll(report.itemByKey)
			} else {
				items = freshItems(report.itemByKey, code, now, timeout)
			}
			if len(items) == 0 {
				continue
			}
			out = append(out, reportSnapshot{category: category, code: code, items: items})
		}
	}
	return out
}

// markSent 记录指定 category+code 的上次发送时间，用于控制上报频率。
func (r *reportManager) markSent(category ReportCategory, code string, sentAt time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if report := r.cache[category][code]; report != nil {
		report.lastSent = sentAt
	}
}

// parseItemInterval 从 Item 中解析秒级间隔字段。
// key 不存在或值非法时返回 fallback。
func parseItemInterval(item isp.Item, key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(item[key])
	if raw == "" {
		return fallback
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec <= 0 {
		return fallback
	}
	return time.Duration(sec) * time.Second
}

// freshnessTimeout 计算上报数据的新鲜度超时阈值。
// 公式：max(interval*2, interval+10s)。
// 超过此阈值未收到 gRPC 更新的 item 视为过期，不再上报。
func freshnessTimeout(interval time.Duration) time.Duration {
	twice := interval * 2
	plus := interval + 10*time.Second
	if twice > plus {
		return twice
	}
	return plus
}

func cloneItem(item isp.Item) isp.Item {
	cloned := make(isp.Item, len(item))
	for k, v := range item {
		cloned[k] = v
	}
	return cloned
}

// freshItems 从缓存中筛选未过期的 item，返回克隆后的列表。
// 已过期的 item 记录日志（含类别名称、变电站编码、item key），不包含在返回结果中。
func freshItems(items map[string]*cachedItem, code string, now time.Time, timeout time.Duration) []isp.Item {
	out := make([]isp.Item, 0, len(items))
	for key, cached := range items {
		if cached.updatedAt.IsZero() {
			continue
		}
		if now.Sub(cached.updatedAt) >= timeout {
			logx.Debugf("[ispagent] report cache item expired name=%s code=%s itemKey=%s updated_at=%s", categoryMessageName(cached.category), code, key, cached.updatedAt.Format(time.RFC3339))
			continue
		}
		out = append(out, cloneItem(cached.item))
	}
	return out
}

// itemKey 根据 keyAttrs 构建 Item 的唯一标识。
// 例如 patroldevice_code=robot-1|type=3。
// keyAttrs 全部缺失时用 fallbackIndex 保证至少有一个区分符。
func itemKey(keyAttrs []string, item isp.Item, fallbackIndex int) string {
	parts := make([]string, 0, len(keyAttrs))
	complete := true
	for _, attr := range keyAttrs {
		value := strings.TrimSpace(item[attr])
		if value == "" {
			complete = false
			continue
		}
		parts = append(parts, attr+"="+value)
	}
	if len(parts) == 0 {
		return fmt.Sprintf("item_index=%d", fallbackIndex)
	}
	if !complete {
		parts = append(parts, fmt.Sprintf("item_index=%d", fallbackIndex))
	}
	return strings.Join(parts, "|")
}

// setNoFreshCheck 控制指定类别是否跳过新鲜度检查。
// 设为 true 后，即使下游长时间未刷新缓存，也会继续上报旧数据。
func (r *reportManager) setNoFreshCheck(category ReportCategory, skip bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if skip {
		r.noFreshCheck[category] = true
	} else {
		delete(r.noFreshCheck, category)
	}
}

// cloneAll 复制缓存中全部 item（不检查过期），在 noFreshCheck 时使用。
func cloneAll(items map[string]*cachedItem) []isp.Item {
	out := make([]isp.Item, 0, len(items))
	for _, cached := range items {
		out = append(out, cloneItem(cached.item))
	}
	return out
}

// ReportIntervals 返回所有上报类别的当前间隔和预留间隔。
func (c *Client) ReportIntervals() map[ReportCategory]time.Duration {
	c.reports.mu.RLock()
	defer c.reports.mu.RUnlock()
	out := make(map[ReportCategory]time.Duration, len(c.reports.intervals))
	for k, v := range c.reports.intervals {
		out[k] = v
	}
	return out
}

// ReservedIntervals 返回注册响应中预留的间隔（nest_run_interval、weather_interval 等）。
func (c *Client) ReservedIntervals() map[string]time.Duration {
	c.reports.mu.RLock()
	defer c.reports.mu.RUnlock()
	out := make(map[string]time.Duration, len(c.reports.reservedIntervals))
	for k, v := range c.reports.reservedIntervals {
		out[k] = v
	}
	return out
}

// SetNoFreshCheck 暴露给外部调用方，控制指定上报类别的新鲜度检查开关。
func (c *Client) SetNoFreshCheck(category ReportCategory, skip bool) {
	c.reports.setNoFreshCheck(category, skip)
}

// CategoryMessageName 返回 ISP 协议中该 messageId 对应的中文名称。
func CategoryMessageName(category ReportCategory) string {
	typ, cmd := isp.DecodeMessageID(int(category))
	return (&isp.Message{Type: typ, Command: cmd}).MessageName()
}

func categoryMessageName(category ReportCategory) string {
	return CategoryMessageName(category)
}

// CacheReport gRPC 上报入口：将 proto 数据写入本地缓存，立即返回受理结果。
// 后续由 reportTick 按间隔定时发送到上级 ISP 系统。
func (c *Client) CacheReport(ctx context.Context, category ReportCategory, code string, items []isp.Item) error {
	if c.reports.update(category, code, items, time.Now()) {
		logx.WithContext(ctx).Infof("[ispagent] report cache updated name=%s code=%s items=%d", categoryMessageName(category), code, len(items))
		return nil
	}
	return isp.NewIspError(isp.StatusReject, "未知上报类别")
}
