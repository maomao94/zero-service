package ispclient

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
	ReportCategoryDroneNestRunData        ReportCategory = ReportCategory(isp.MessageIDDroneNestRunData)        // 10004-0 无人机机巢运行数据
	ReportCategoryEnvData                 ReportCategory = ReportCategory(isp.MessageIDEnvData)                 // 21-0 环境/微气象数据
)

// 默认上报间隔。
// 坐标默认 2 秒（noFreshCheck=true，不做过期清理）；其余类别默认 1 分钟。
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
	ReportCategoryDroneNestRunData:        {"nest_code", "type"},
	ReportCategoryEnvData:                 {"patroldevice_code", "type"},
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
// snapLastSent 记录快照时刻的 lastSent，markSent 用它判断是否被并发 update 重置过。
type reportSnapshot struct {
	category     ReportCategory
	code         string
	items        []isp.Item
	snapLastSent time.Time
}

type expiredReportItem struct {
	category  ReportCategory
	code      string
	itemKey   string
	updatedAt time.Time
}

type emptyReportRef struct {
	category ReportCategory
	code     string
}

// reportManager 管理所有上报类别的缓存、间隔和新鲜度策略。
//
// 并发模型：
//   - 写操作（update/applyRegistrationIntervals/markSent/setInterval/deleteExpired）持 Lock
//   - dueReports 扫描/clone 时持 RLock，过期清理在扫描后短暂持 Lock
//
// intervals:     每个上报类别的当前间隔（注册响应可覆盖运行数据）
// cache:         三级结构 category → code（变电站编码） → item key → cachedItem
// noFreshCheck:  设为 true 的类别跳过新鲜度检查，始终按间隔上报缓存中的已有数据

// ReportManagerOptions 上报管理器构造配置，零值表示使用默认间隔。
type ReportManagerOptions struct {
	RunDataInterval        time.Duration
	StatusDataInterval     time.Duration
	CoordInterval          time.Duration
	NestRunInterval        time.Duration
	EnvDataInterval        time.Duration
	NoFreshCheckCategories []ReportCategory
}

// ReportManagerOption 上报管理器构造选项。
type ReportManagerOption func(*ReportManagerOptions)

// WithRunDataInterval 设置巡视装置运行数据上报间隔，非正值使用默认值。
func WithRunDataInterval(d time.Duration) ReportManagerOption {
	return func(o *ReportManagerOptions) { o.RunDataInterval = d }
}

// WithStatusDataInterval 设置巡视装置状态数据上报间隔，非正值使用默认值。
func WithStatusDataInterval(d time.Duration) ReportManagerOption {
	return func(o *ReportManagerOptions) { o.StatusDataInterval = d }
}

// WithCoordInterval 设置巡视装置坐标上报间隔，非正值使用默认值。
func WithCoordInterval(d time.Duration) ReportManagerOption {
	return func(o *ReportManagerOptions) { o.CoordInterval = d }
}

// WithNestRunInterval 设置无人机机巢运行数据上报间隔，非正值使用默认值。
func WithNestRunInterval(d time.Duration) ReportManagerOption {
	return func(o *ReportManagerOptions) { o.NestRunInterval = d }
}

// WithEnvDataInterval 设置环境/微气象数据上报间隔，非正值使用默认值。
func WithEnvDataInterval(d time.Duration) ReportManagerOption {
	return func(o *ReportManagerOptions) { o.EnvDataInterval = d }
}

// WithNoFreshCheck 设置指定类别跳过新鲜度检查（始终按间隔上报已有数据）。
func WithNoFreshCheck(categories ...ReportCategory) ReportManagerOption {
	return func(o *ReportManagerOptions) {
		o.NoFreshCheckCategories = append(o.NoFreshCheckCategories, categories...)
	}
}

type reportManager struct {
	mu           sync.RWMutex
	intervals    map[ReportCategory]time.Duration
	cache        map[ReportCategory]map[string]*cachedReport
	noFreshCheck map[ReportCategory]bool
}

func newReportManager(opts ...ReportManagerOption) *reportManager {
	o := &ReportManagerOptions{}
	for _, opt := range opts {
		opt(o)
	}
	r := &reportManager{
		intervals:    make(map[ReportCategory]time.Duration, len(keyAttrsByCategory)),
		cache:        make(map[ReportCategory]map[string]*cachedReport, len(keyAttrsByCategory)),
		noFreshCheck: make(map[ReportCategory]bool),
	}
	for cat := range keyAttrsByCategory {
		r.intervals[cat] = defaultReportInterval
		r.cache[cat] = make(map[string]*cachedReport)
	}
	if o.RunDataInterval > 0 {
		r.intervals[ReportCategoryPatrolDeviceRunData] = o.RunDataInterval
	}
	if o.StatusDataInterval > 0 {
		r.intervals[ReportCategoryPatrolDeviceStatusData] = o.StatusDataInterval
	}
	if o.CoordInterval > 0 {
		r.intervals[ReportCategoryPatrolDeviceCoordinates] = o.CoordInterval
	} else {
		r.intervals[ReportCategoryPatrolDeviceCoordinates] = defaultCoordInterval
	}
	if o.NestRunInterval > 0 {
		r.intervals[ReportCategoryDroneNestRunData] = o.NestRunInterval
	}
	if o.EnvDataInterval > 0 {
		r.intervals[ReportCategoryEnvData] = o.EnvDataInterval
	}
	for _, cat := range o.NoFreshCheckCategories {
		r.noFreshCheck[cat] = true
	}
	return r
}

// update 将 gRPC 收到的上报数据写入缓存。
// category+code 定位到具体缓存槽位，itemKey 区分同一上报里的多个数据点。
// 每次 update 会刷新 updatedAt，使该 item 重新进入新鲜期。
// 非法 category 返回 error。
func (r *reportManager) update(category ReportCategory, code string, items []isp.Item, now time.Time) error {
	keyAttrs, ok := keyAttrsByCategory[category]
	if !ok {
		return fmt.Errorf("未知上报类别: %d", int(category))
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
		_, exists := report.itemByKey[key]
		report.itemByKey[key] = &cachedItem{item: cloneItem(item), category: category, updatedAt: now}
		if !exists && !report.lastSent.IsZero() {
			report.lastSent = time.Time{}
		}
	}
	return nil
}

// applyRegistrationIntervals 解析注册响应（251-4）中的间隔配置并应用。
//   - patroldevice_run_interval → 巡视装置运行数据上报间隔
//   - nest_run_interval → 无人机机巢运行数据上报间隔
//   - weather_interval → 环境/微气象数据上报间隔
//
// 字段缺失或非法时保持当前值不变。
// 解析完成后重置全部缓存的 lastSent，使注册成功后能立即上报。
func (r *reportManager) applyRegistrationIntervals(items []isp.Item) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(items) > 0 {
		item := items[0]
		if d := parseItemInterval(item, "patroldevice_run_interval", 0); d > 0 {
			r.intervals[ReportCategoryPatrolDeviceRunData] = d
		}
		if d := parseItemInterval(item, "nest_run_interval", 0); d > 0 {
			r.intervals[ReportCategoryDroneNestRunData] = d
		}
		if d := parseItemInterval(item, "weather_interval", 0); d > 0 {
			r.intervals[ReportCategoryEnvData] = d
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
// 持 RLock 扫描并 clone 到期数据，释放读锁后短写锁清理过期/空缓存。
func (r *reportManager) dueReports(now time.Time) []reportSnapshot {
	var expired []expiredReportItem
	var empty []emptyReportRef
	r.mu.RLock()
	out := make([]reportSnapshot, 0, len(r.cache))
	for category, reports := range r.cache {
		interval := r.intervals[category]
		if interval <= 0 {
			continue
		}
		timeout := freshnessTimeout(interval)
		for code, report := range reports {
			if len(report.itemByKey) == 0 {
				empty = append(empty, emptyReportRef{category: category, code: code})
				continue
			}
			shouldReport := report.lastSent.IsZero() || now.Sub(report.lastSent) >= interval
			var items []isp.Item
			if r.noFreshCheck[category] {
				if !shouldReport {
					continue
				}
				items = cloneAll(report.itemByKey)
			} else {
				var fresh []isp.Item
				var expiredItems []expiredReportItem
				fresh, expiredItems = freshItems(report.itemByKey, code, now, timeout)
				expired = append(expired, expiredItems...)
				if shouldReport {
					items = fresh
				}
			}
			if !shouldReport {
				continue
			}
			if len(items) == 0 {
				continue
			}
			out = append(out, reportSnapshot{category: category, code: code, items: items, snapLastSent: report.lastSent})
		}
	}
	r.mu.RUnlock()
	if len(expired) > 0 || len(empty) > 0 {
		r.deleteExpired(expired, empty)
	}
	return out
}

func (r *reportManager) deleteExpired(expired []expiredReportItem, empty []emptyReportRef) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ref := range empty {
		reports := r.cache[ref.category]
		if reports == nil {
			continue
		}
		report := reports[ref.code]
		if report != nil && len(report.itemByKey) == 0 {
			delete(reports, ref.code)
		}
	}
	for _, item := range expired {
		reports := r.cache[item.category]
		if reports == nil {
			continue
		}
		report := reports[item.code]
		if report == nil {
			continue
		}
		cached := report.itemByKey[item.itemKey]
		if cached == nil || !cached.updatedAt.Equal(item.updatedAt) {
			continue
		}
		delete(report.itemByKey, item.itemKey)
		if len(report.itemByKey) == 0 {
			delete(reports, item.code)
		}
	}
}

// markSent 记录指定 category+code 的上次发送时间，用于控制上报频率。
// snapLastSent 是快照时刻的 lastSent，如果 lastSent 在快照后被并发 update 重置为零，则跳过更新。
func (r *reportManager) markSent(category ReportCategory, code string, sentAt time.Time, snapLastSent time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if report := r.cache[category][code]; report != nil {
		if !snapLastSent.IsZero() && report.lastSent.IsZero() {
			return
		}
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

// freshItems 从缓存中筛选未过期的 item，返回克隆后的列表和过期 key。
// 过期 key 统一由 deleteExpired 在释放读锁后清理。
func freshItems(items map[string]*cachedItem, code string, now time.Time, timeout time.Duration) ([]isp.Item, []expiredReportItem) {
	out := make([]isp.Item, 0, len(items))
	expired := make([]expiredReportItem, 0)
	for key, cached := range items {
		if cached.updatedAt.IsZero() {
			continue
		}
		if now.Sub(cached.updatedAt) >= timeout {
			logx.Debugf("[ispagent] report cache item expired name=%s code=%s itemKey=%s updated_at=%s now=%s timeout=%s",
				categoryMessageName(cached.category), code, key, cached.updatedAt.Format(time.RFC3339), now.Format(time.RFC3339), timeout)
			expired = append(expired, expiredReportItem{category: cached.category, code: code, itemKey: key, updatedAt: cached.updatedAt})
			continue
		}
		out = append(out, cloneItem(cached.item))
	}
	return out, expired
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

func (r *reportManager) setInterval(category ReportCategory, d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.intervals[category]; ok && d > 0 {
		r.intervals[category] = d
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
func (c *IspClient) ReportIntervals() map[ReportCategory]time.Duration {
	c.reports.mu.RLock()
	defer c.reports.mu.RUnlock()
	out := make(map[ReportCategory]time.Duration, len(c.reports.intervals))
	for k, v := range c.reports.intervals {
		out[k] = v
	}
	return out
}

// SetInterval 运行时覆盖指定类别的上报间隔，非正值忽略。
func (c *IspClient) SetInterval(category ReportCategory, d time.Duration) {
	c.reports.setInterval(category, d)
}

// CategoryKeyAttrs 返回指定类别的缓存 key 属性列表。
func CategoryKeyAttrs(category ReportCategory) []string {
	return keyAttrsByCategory[category]
}

// CategoryNoFreshCheck 返回指定类别是否跳过新鲜度检查。
func (c *IspClient) CategoryNoFreshCheck(category ReportCategory) bool {
	c.reports.mu.RLock()
	defer c.reports.mu.RUnlock()
	return c.reports.noFreshCheck[category]
}

// SetNoFreshCheck 暴露给外部调用方，控制指定上报类别的新鲜度检查开关。
func (c *IspClient) SetNoFreshCheck(category ReportCategory, skip bool) {
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
func (c *IspClient) CacheReport(ctx context.Context, category ReportCategory, code string, items []isp.Item) error {
	if err := c.reports.update(category, code, items, time.Now()); err != nil {
		return err
	}
	logx.WithContext(ctx).Infof("[ispagent] report cache updated name=%s code=%s items=%d", categoryMessageName(category), code, len(items))
	return nil
}
