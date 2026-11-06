package modbusx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/syncx"
)

// ErrManagerClosed 表示管理器已退出，不再返回或创建客户端。
var ErrManagerClosed = errors.New("modbus: 客户端管理器已关闭")

// ErrConfigChanged 表示加载结果返回前已被移除，调用方可在下一次请求重新获取。
var ErrConfigChanged = errors.New("modbus: 配置已变更，请重新获取客户端")

// ErrConfigUnavailable 表示配置不存在或未启用。
// 后台清理据此移除客户端，调用方仍会收到 svc 包装好的平台错误码。
var ErrConfigUnavailable = errors.New("modbus: 配置不存在或未启用")

const (
	// clientCleanInterval 是后台配置检查间隔；实际生效时间还受串行查询和加载锁等待影响。
	clientCleanInterval = 30 * time.Second
	// clientCleanTimeout 限制单个 code 的配置查询耗时。
	clientCleanTimeout = 5 * time.Second
)

// cachedClient 保存客户端、创建时的加载函数与配置指纹。
type cachedClient struct {
	client *ModbusClient
	load   func(context.Context) (*ModbusClientConf, error)
	cfgMd5 string
}

// ClientManager 按 modbusCode 缓存长期客户端，空 code 专用于文件默认配置。
// 不同 code 即使地址相同也不共享客户端，从站号和 TLS 配置始终固定。
// 取用只查缓存；后台按周期校验配置，更新、删除或禁用时移除客户端，下次取用重建。
type ClientManager struct {
	mu        sync.Mutex // 只保护缓存，不跨越查库、建连、关闭或日志操作
	clients   map[string]*cachedClient
	loadMu    sync.RWMutex // 加载持读锁，移除持写锁；缓存命中不经过此锁
	barrier   syncx.SingleFlight
	done      chan struct{}
	closeOnce sync.Once
	closed    bool
	cleanupWG sync.WaitGroup // 后台淘汰的连接关闭任务；缓存锁内登记，Close 标记 closed 后等待
}

// NewClientManager 创建客户端管理器，不拨号，后台只做配置校验。
func NewClientManager() *ClientManager {
	m := &ClientManager{
		clients: make(map[string]*cachedClient),
		barrier: syncx.NewSingleFlight(),
		done:    make(chan struct{}),
	}
	go m.cleanLoop()
	return m
}

// GetClient 获取 code 对应的客户端，未缓存时加载配置并创建。
// 命中缓存直接返回，不查库；配置变化由后台校验移除后触发重建。
// load 会被后台校验周期重复调用，必须幂等且无副作用；配置不存在或未启用时
// 返回包含 ErrConfigUnavailable 的错误，后台校验据此立即移除客户端。
// 同 code 的并发加载通过 SingleFlight 合并，共享首次调用者的 ctx 与结果；
// load 不能回调管理器，否则会等待自身。等待期间不支持 ctx 立即取消。
func (m *ClientManager) GetClient(ctx context.Context, code string, load func(context.Context) (*ModbusClientConf, error)) (*ModbusClient, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, ErrManagerClosed
	}
	if entry := m.clients[code]; entry != nil {
		m.mu.Unlock()
		return entry.client, nil
	}
	m.mu.Unlock()

	value, err := m.barrier.Do(code, func() (any, error) {
		return m.loadClient(ctx, code, load)
	})
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, err
	}
	// 加载函数 panic 时 SingleFlight 的等待者会拿到 (nil, nil)，不能直接断言。
	cli, ok := value.(*ModbusClient)
	if !ok || cli == nil {
		return nil, fmt.Errorf("modbus: 配置加载结果无效")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, ErrManagerClosed
	}
	// 移除可能发生在 loadClient 解锁后、SingleFlight 返回前。
	if entry := m.clients[code]; entry == nil || entry.client != cli {
		return nil, ErrConfigChanged
	}
	return cli, nil
}

// loadClient 持加载读锁直到发布，移除必须等加载完成后再删除缓存。
// 不持缓存锁查库；不同 code 可以并行加载，写锁等待时新的加载也会等待。
// 配置不存在、未启用或查询失败直接返回错误，失败结果不缓存。
func (m *ClientManager) loadClient(ctx context.Context, code string,
	load func(context.Context) (*ModbusClientConf, error),
) (*ModbusClient, error) {
	m.loadMu.RLock()
	defer m.loadMu.RUnlock()
	m.mu.Lock()
	closed := m.closed
	m.mu.Unlock()
	if closed {
		return nil, ErrManagerClosed
	}
	if load == nil {
		return nil, fmt.Errorf("modbus: 配置加载函数不能为空")
	}

	// 同一 code 的加载已由 SingleFlight 串行，这里再查一次缓存，
	// 避免上一轮加载刚发布后重复建连。
	m.mu.Lock()
	if entry := m.clients[code]; entry != nil {
		cli := entry.client
		m.mu.Unlock()
		return cli, nil
	}
	m.mu.Unlock()

	conf, err := load(ctx)
	if err != nil {
		return nil, err
	}
	if conf == nil {
		return nil, fmt.Errorf("modbus: 配置加载函数返回空配置")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cli, err := NewModbusClient(conf)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		_ = cli.Close()
		return nil, ErrManagerClosed
	}
	replaced := m.clients[code]
	if replaced != nil {
		replaced.client.invalidate()
	}
	m.clients[code] = &cachedClient{client: cli, load: load, cfgMd5: cfgMd5(conf)}
	m.mu.Unlock()

	if replaced != nil {
		_ = replaced.client.Close()
		logx.WithContext(ctx).Infow("重建 Modbus 客户端", logx.Field("modbusCode", code))
		return cli, nil
	}
	logx.WithContext(ctx).Infow("创建 Modbus 客户端", logx.Field("modbusCode", code))
	return cli, nil
}

// cleanLoop 按周期校验缓存客户端的配置。
func (m *ClientManager) cleanLoop() {
	ticker := time.NewTicker(clientCleanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			m.cleanStale()
		}
	}
}

// cleanStale 重新加载每个已缓存 code 的配置：
// 配置不存在/未启用或指纹变化时移除客户端，下次取用重建；
// 查询失败（例如数据库不可用）保留客户端，避免把配置故障放大成通信中断。
// 空 code 使用文件配置，不参与校验。
func (m *ClientManager) cleanStale() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	entries := make(map[string]*cachedClient, len(m.clients))
	for code, entry := range m.clients {
		if code == "" {
			continue
		}
		entries[code] = entry
	}
	m.mu.Unlock()

	for code, entry := range entries {
		ctx, cancel := context.WithTimeout(context.Background(), clientCleanTimeout)
		conf, err := entry.load(ctx)
		cancel()
		if err != nil {
			if errors.Is(err, ErrConfigUnavailable) {
				m.removeIfCurrent(code, entry, "配置不存在或未启用")
			} else {
				logx.Debugf("校验 Modbus 配置失败，保留当前客户端: code=%s err=%v", code, err)
			}
			continue
		}
		if conf != nil && cfgMd5(conf) != entry.cfgMd5 {
			m.removeIfCurrent(code, entry, "配置已更新")
		}
	}
}

// removeIfCurrent 仅移除仍然有效的缓存项，避免误删并发重建的客户端。
// 连接异步关闭，避免等待一个设备的在途 I/O 时阻塞其他配置的校验和淘汰。
func (m *ClientManager) removeIfCurrent(code string, entry *cachedClient, reason string) {
	m.loadMu.Lock()
	m.mu.Lock()
	if m.clients[code] != entry {
		m.mu.Unlock()
		m.loadMu.Unlock()
		return
	}
	entry.client.invalidate()
	delete(m.clients, code)
	// 与 Close 清空缓存共用缓存锁，确保 Close 开始等待后不会再登记新任务。
	m.cleanupWG.Add(1)
	m.mu.Unlock()
	m.loadMu.Unlock()

	go func() {
		defer m.cleanupWG.Done()
		_ = entry.client.Close()
	}()
	logx.Infow("移除 Modbus 客户端", logx.Field("modbusCode", code), logx.Field("reason", reason))
}

// cfgMd5 生成配置指纹，用于后台比对是否变化。
// 新增字段会自动纳入，不依赖结构体可比较性。
func cfgMd5(conf *ModbusClientConf) string {
	raw, err := json.Marshal(conf)
	if err != nil {
		raw = []byte(fmt.Sprintf("%+v", *conf))
	}
	return cryptor.Md5String(string(raw))
}

// RemoveClient 等待在途加载发布后作废旧客户端并清除缓存，再在锁外关闭连接。
// 旧引用的排队及后续请求返回 ErrClientInvalidated，已通过失效检查的调用允许结束。
// 后续请求会重新读取配置，已删除或禁用则报错。
func (m *ClientManager) RemoveClient(code string) bool {
	m.loadMu.Lock()
	m.mu.Lock()
	entry, ok := m.clients[code]
	if ok {
		entry.client.invalidate()
	}
	delete(m.clients, code)
	m.mu.Unlock()
	m.loadMu.Unlock()
	if !ok {
		return false
	}
	_ = entry.client.Close()
	logx.Infow("移除 Modbus 客户端", logx.Field("modbusCode", code))
	return true
}

// Close 停止后台校验，作废全部缓存客户端并关闭连接，等待后台淘汰的连接关闭，可重复调用。
func (m *ClientManager) Close() {
	m.closeOnce.Do(func() { close(m.done) })
	m.mu.Lock()
	m.closed = true
	clients := m.clients
	for _, entry := range clients {
		entry.client.invalidate()
	}
	m.clients = make(map[string]*cachedClient)
	m.mu.Unlock()
	for _, entry := range clients {
		_ = entry.client.Close()
	}
	m.cleanupWG.Wait()
}
