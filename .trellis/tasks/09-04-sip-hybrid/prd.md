# SIP 电话混合会议功能

## Goal

在现有 LiveKit 会议系统基础上，增加 SIP 电话能力，实现**电话用户**与**WebRTC 用户**在同一 Room 内混合通话。
第一阶段以局域网拨号为验证目标，接口设计兼容后续对接国内 SIP trunk 供应商和 AI 语音 Agent。

## 用户场景

### 场景 1：点对点拨号

用户 A（浏览器）想和用户 B（电话）通话：
1. 用户 A 在前端点击"拨打电话"，输入电话号码
2. 后端自动创建 Meeting + Room，发起 SIP 外呼
3. 用户 B 手机振铃 → 接听 → 自动进入 Room
4. 用户 A 在浏览器看到"电话用户已加入"，双方开始通话

### 场景 2：会议中邀请电话加入

用户 A 正在开会，需要邀请场外人员：
1. 用户 A 在会议界面点击"邀请电话参会者"
2. 输入电话号码，发起外呼
3. 对方接听后以 SIP Participant 身份加入当前 Room
4. Room 内所有用户（Web + 电话）可互相通话

### 场景 3：来电接入

外部用户拨打已配置的电话号码：
1. 来电经 SIP trunk 进入 LiveKit SIP Server
2. Dispatch Rule 自动路由到指定 Room（或创建新 Room）
3. Web 端看到"来电用户已加入"

## 功能范围

### P0 — 第一阶段（局域网验证）

| 模块 | 功能 | 说明 |
|------|------|------|
| SDK 验证 | livekitx SIP API 示例测试 | 验证 CreateSIPParticipant 等 API 可用 |
| Proto 接口 | SIP trunk CRUD | 创建/列出/删除 inbound/outbound trunk |
| Proto 接口 | Dispatch rule CRUD | 创建/列出/删除路由规则 |
| Proto 接口 | 外呼拨号 | 指定号码 + Room，发起 SIP 外呼 |
| Proto 接口 | 挂断 | 终止进行中的 SIP 通话 |
| 业务逻辑 | Trunk 管理逻辑 | 调用 LiveKit SIP API，结果持久化到数据库 |
| 业务逻辑 | 路由规则管理逻辑 | 调用 LiveKit SIP API，结果持久化到数据库 |
| 业务逻辑 | 外呼逻辑 | 创建 Meeting + 发起 SIP call + 写通话记录 |
| 业务逻辑 | 来电处理 | Webhook 接收 SIP 事件 → 关联 Meeting |
| HTTP 网关 | SIP 管理 API | Trunk/Rule/Dial/Hangup HTTP 端点 |
| 部署 | LiveKit SIP Server 配置 | Docker Compose 部署，局域网可用 |
| 前端 | 点对点拨号 UI | 拨号盘 + 通话状态展示 |
| 前端 | 会议中邀请 UI | 邀请电话参会者弹窗 |

### P1 — 第二阶段（生产可用）

| 模块 | 功能 | 说明 |
|------|------|------|
| 供应商适配 | 国内 SIP trunk 对接 | 阿里云/天润融通等 |
| 通话记录 | CDR 持久化 | 通话时长、状态、挂断原因 |
| 会议集成 | 来电自动关联会议 | Webhook 事件处理 |
| 前端 | 来电提醒 UI | 来电弹窗 + 接听/拒绝 |

## 接口设计约束

1. **供应商无关**：Proto 接口不暴露特定供应商细节，供应商信息作为 metadata 存储
2. **注释规范**：所有 Proto 字段和 RPC 方法必须有中文注释，说明业务含义
3. **向后兼容**：新增字段使用 optional，不破坏已有接口
4. **最小化**：不做 Agent 集成，不做智能话务，只做基础电话能力

## 验收标准

### P0 验收

- [ ] livekitx SIP API 示例测试通过（局域网环境）
- [ ] Proto 编译通过，生成 Go 代码
- [ ] HTTP 网关 SIP 管理端点可调用
- [ ] 局域网外呼：浏览器用户拨号 → 软电话振铃 → 接听 → 双方通话
- [ ] 会议中邀请电话：已有会议中添加电话参会者
- [ ] 通话记录写入数据库

### 技术约束

- LiveKit Server 版本：v2.18.1（已有）
- LiveKit Protocol 版本：v1.49.0（已有）
- Go 版本：1.26
- 框架：go-zero v1.10.3
- 数据库：GORM + MySQL/PostgreSQL
