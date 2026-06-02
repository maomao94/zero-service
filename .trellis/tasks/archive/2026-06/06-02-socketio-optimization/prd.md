# SocketIO 代码优化分析

## 目标

优化 `common/socketiox/` 核心包，提升代码质量、可维护性和性能。

## 范围

本次聚焦 `common/socketiox/` 目录下的三个文件：
- `server.go` — SocketIO 服务器核心实现（829 行）
- `container.go` — 服务发现与客户端容器（425 行）
- `handler.go` — HTTP 处理器封装（40 行）

`socketapp/`（socketgtw、socketpush）的 Proto 去重和 Logic 去重暂不处理。

## 用户价值

- 通过代码去重降低维护成本
- 提升代码质量和一致性
- 在高连接数场景下获得更好的性能
- 降低新开发者上手难度

## 已确认事实（代码审查结果）

### 架构概况
- 两个服务：`socketgtw`（网关）和 `socketpush`（推送 API）
- `common/socketiox/` 提供核心 SocketIO 服务器实现
- 服务发现支持 Etcd、Nacos、直连三种方式
- Token 验证支持密钥轮换

### server.go 问题

1. **事件处理代码重复**：`EventJoinRoom`、`EventLeaveRoom`、`EventRoomBroadcast`、`EventGlobalBroadcast` 四个事件处理逻辑结构高度相似（参数校验、解析、响应），约 200 行可提取公共函数
2. **错误处理不一致**：
   - `JoinRoom` 返回 error，但 `connectHook` 中调用时只 log 不返回
   - `statLoop` 中 `EmitString` 失败被静默忽略
3. **锁粒度问题**：`Session.SetMetadata` 用 Mutex，但 `Server.sessions` 用 RWMutex，高并发下可能有竞争
4. **死代码**：`randomUUID()` 方法未被使用
5. **statLoop 性能**：每分钟遍历所有 session 发送统计，O(n) 复杂度

### container.go 问题

1. **`subset()` 函数效率低**：先 `rand.Shuffle` 整个切片再取前 N 个，O(n) 操作；对大集合可优化为 Fisher-Yates 部分洗牌 O(k)
2. **gRPC 客户端创建重复**：`getConn4Etcd`、`getConn4Direct`、`getConn4Nacos` 三个方法中创建客户端的代码几乎相同（约 10 行），可提取公共函数
3. **Nacos 定时拉取逻辑**：60 秒全量拉取 + Subscribe 回调，两者可能产生竞争更新

### handler.go 问题

1. **文件过小**（40 行），仅做了一层转发，可考虑是否有必要独立存在

## 需求

### 必须完成
- [ ] 提取 server.go 中事件处理的公共函数，消除重复代码
- [ ] 修复死代码（`randomUUID()`）
- [ ] 统一错误处理模式
- [ ] 提取 container.go 中 gRPC 客户端创建的公共函数
- [ ] 优化 `subset()` 函数为部分洗牌算法

### 应该完成
- [ ] 评估 handler.go 是否有必要独立存在
- [ ] 改善 statLoop 性能（按需统计或降低频率）
- [ ] 评估 Nacos 双重更新机制的竞争风险

### 不在范围内
- socketapp/ 的 Proto/Llogic 去重
- 第三方依赖（socket.io-golang）变更
- 前端客户端变更
- 架构层面的服务合并

## 待确认问题

1. 优先级：可维护性 vs 性能 vs 两者兼顾？
2. Proto 变更是否有向后兼容约束？
3. 预期规模是多少（连接数/秒、消息吞吐量）？
4. 是否考虑将 socketgtw 和 socketpush 合并为单个服务？
