# ISP 巡检协议代理

`ispagent` 是面向变电站远程智能巡视场景的 ISP 协议 TCP 代理服务，用于对接上级巡视系统、区域巡视主机或厂商侧机器人/巡视设备平台。

ISP 在本项目中指 **Inspection Substation Protocol**，对应“区域型变电站远程智能巡视系统技术规范”中以 TCP + XML 报文进行巡视设备、任务、模型和告警数据交换的一类协议。公开资料通常只描述远程智能巡视系统的架构、功能、性能和接口范围，具体 Type/Command、XML Item 字段和模型文件格式以项目实现和现场对接规范为准。

## 场景定位

典型变电站智能巡视系统包含区域层和站端层：

| 层级 | 组件 | 说明 |
|------|------|------|
| 区域层 | 区域巡视主机、集中监控系统 | 统一下发巡视任务、拉取模型文件、接收巡视结果和状态 |
| 站端层 | 巡视主机、机器人、摄像机、无人机、传感器 | 执行巡视任务，上报设备状态、坐标、运行数据、告警和结果 |
| 对接层 | `ispagent` | 维护 ISP TCP 长连接，完成注册、心跳、消息编解码、任务和模型同步 |

`ispagent` 当前承担“协议适配层”职责：对上以 gRPC 暴露项目内部调用接口，对外以 ISP TCP 客户端连接上级系统。

## 协议概览

### 传输与帧格式

ISP 使用 TCP 长连接，完整帧结构如下：

```text
0xEB90(2B BE)
+ SendSeq(8B LE)
+ RecvSeq(8B LE)
+ SessionSource(1B)
+ XMLLength(4B LE)
+ XML(UTF-8)
+ 0xEB90(2B BE)
```

字段说明：

| 字段 | 说明 |
|------|------|
| `0xEB90` | 帧起止标志，按大端表示 |
| `SendSeq` | 本端发送序号，8 字节小端 |
| `RecvSeq` | 对端回执序号，等于上次收到的对端 `SendSeq` |
| `SessionSource` | 会话来源，`0x00` 表示客户端发起，`0x01` 表示服务端响应 |
| `XMLLength` | XML 正文长度，4 字节小端 |
| `XML` | UTF-8 XML 报文正文 |

项目实现位于 `common/isp/serializer.go`，通过 `gnetx.LengthPrefixCodec` 处理头尾标志、长度字段和最大帧长度。

### XML 报文

XML 根元素可配置为 `PatrolDevice` 或 `PatrolHost`，默认使用 `PatrolDevice`。

```xml
<?xml version="1.0" encoding="UTF-8"?>
<PatrolDevice>
  <SendCode>local-device-code</SendCode>
  <ReceiveCode>remote-system-code</ReceiveCode>
  <Type>251</Type>
  <Code>200</Code>
  <Command>3</Command>
  <Time>2026-07-10 10:00:00</Time>
  <Items>
    <Item key="value" />
  </Items>
</PatrolDevice>
```

核心字段：

| 字段 | 说明 |
|------|------|
| `SendCode` | 发送方唯一标识 |
| `ReceiveCode` | 接收方唯一标识 |
| `Type` | 消息大类，高 16 位参与 `messageId` 编码 |
| `Command` | 消息子命令，低 16 位参与 `messageId` 编码；上报类消息可省略或为 0 |
| `Code` | 目标对象编码，含义随消息变化，可表示变电站编码、任务编码、巡视执行 ID 等 |
| `Time` | 业务时间 |
| `Items/Item` | 动态属性列表，项目以 `map[string]string` 解析 |

消息路由 ID 计算方式：

```text
messageId = (Type << 16) | Command
```

## 系统消息

| Type | Command | 名称 | 方向 | 说明 |
|------|---------|------|------|------|
| 251 | 1 | 注册 | `ispagent` -> 上级系统 | TCP 建连后发送本端标识 |
| 251 | 2 | 心跳 | `ispagent` -> 上级系统 | 周期保活 |
| 251 | 3 | 通用应答（无 Item） | 双向 | 普通指令成功/失败回执 |
| 251 | 4 | 通用应答（有 Item） | 双向 | 注册或需要携带返回项的应答 |

通用应答状态写入 XML `Code` 字段：

| Code | 含义 |
|------|------|
| `100` | 需重发 |
| `200` | 成功 |
| `400` | 拒绝 |
| `500` | 错误 |

注册应答 `251-4` 可通过 `Item` 下发心跳和上报间隔，单位均为秒：

| 字段 | 适用标签/场景 | 说明 |
|------|---------------|------|
| `heart_beat_interval` | `PatrolDevice` / `PatrolHost` | 心跳间隔，只影响系统心跳 |
| `patroldevice_run_interval` | `PatrolDevice` / `PatrolHost` | 巡视装置运行数据上报间隔 |
| `nest_run_interval` | `PatrolDevice` / `PatrolHost` | 无人机机巢运行数据间隔，当前解析预留 |
| `weather_interval` | `PatrolDevice` / `PatrolHost` | 微气象数据间隔，当前解析预留 |

心跳间隔、上报间隔、下游数据新鲜度检查是三个独立概念：心跳用于 TCP 会话保活，上报间隔用于向上级 ISP 系统周期发送缓存数据，新鲜度检查用于判断本地缓存是否已经过期。

## 支持的业务能力

### 巡视设备上报

项目支持由内部 gRPC 写入本地上报缓存，`ispagent` 在注册成功后按注册响应中的间隔周期性向上级 ISP 系统上报：

| Type | Command | 名称 | 对外 gRPC |
|------|---------|------|-----------|
| 1 | 0 | 巡视设备状态数据 | `SendPatrolDeviceStatusData` |
| 2 | 0 | 巡视设备运行数据 | `SendPatrolDeviceRunData` |
| 3 | 0 | 巡视设备坐标 | `SendPatrolDeviceCoordinates` |

典型字段来自 `app/ispagent/ispagent.proto`：

| 数据 | 关键字段 |
|------|----------|
| 坐标 | `patrol_device_name`、`patrol_device_code`、`coordinate_pixel`、`coordinate_geography`、`task_patrolled_id` |
| 运行数据 | `patrol_device_name`、`patrol_device_code`、`type`、`value`、`value_unit`、`unit` |
| 状态数据 | `patrol_device_name`、`patrol_device_code`、`type`、`value`、`value_unit`、`unit` |

当前三类 proto 上报均使用同一套通用上报模式：

| 阶段 | 行为 |
|------|------|
| gRPC 输入 | 写入本地内存缓存，返回本地受理成功，不再同步等待上级 ISP 响应 |
| 定时上报 | 按各上报类别自己的间隔从缓存读取最新数据并发送 ISP 上报报文 |
| 新鲜度检查 | 如果下游长时间未刷新缓存，识别为 expired |
| 过期处理 | 非持续上报类 expired 数据不上报，并在 2 秒 tick 扫描时从本地缓存清理 |

缓存上报按 `category + Code + Item key` 组织，避免多个变电站或多个巡视装置互相覆盖：

| 上报类别 | 全局唯一维度 |
|----------|--------------|
| 运行数据 | `Code`（变电站编码）+ `patroldevice_code` + `type` |
| 状态数据 | `Code`（变电站编码）+ `patroldevice_code` + `type` |
| 坐标 | `Code`（变电站编码）+ `patroldevice_code` |

协议中 `patroldevice_code` 由区域巡视主机自定义，并要求区域内巡视装置不重复；实现仍保留 `Code` 维度，确保不同变电站或未来多区域场景不会覆盖缓存。

当前注册协议只给出 `patroldevice_run_interval`，因此巡视装置运行数据收到注册响应后使用注册返回间隔。状态数据和坐标/经纬度数据是独立 ISP 上报类别，不复用运行数据间隔，也不使用心跳间隔；当前默认每 1 分钟检查并上报，后续可由独立协议字段或配置覆盖。

非持续上报类缓存只保留仍有效的数据。tick 扫描发现 item 超过新鲜度窗口后，会先从本地缓存删除；如果同一 `itemKey` 后续重新由 gRPC 写入，会被视为新加入数据，并触发下一次 2 秒 tick 立即上报该 `Code` 下的完整有效快照。坐标类当前配置为持续上报模式，不做新鲜度清理，即使下游暂时不刷新，也继续按坐标间隔上报最后一次坐标。

`nest_run_interval`、`weather_interval` 已作为注册返回字段解析和预留；后续新增无人机机巢、环境、微气象、状态或坐标等 proto 上报时，应复用同一套“缓存 + 独立间隔 + 定时上报 + 过期检查”模式。

### 任务下发与任务控制

| Type | Command | 名称 | 当前处理 |
|------|---------|------|----------|
| 101 | 1 | 任务下发 | 解析任务 Item，写入 `crontask` 任务配置 |
| 102 | 1 | 联动任务下发 | 常量已定义，按任务配置类消息处理 |
| 41 | 1 | 任务启动 | 查找任务配置，生成巡视执行 ID，写入巡视任务表 |
| 41 | 2 | 任务暂停 | 按巡视执行 ID 更新任务状态 |
| 41 | 3 | 任务继续 | 按巡视执行 ID 更新任务状态 |
| 41 | 4 | 任务停止 | 按巡视执行 ID 更新任务状态 |

任务下发 Item 主要字段包括：

| 字段 | 说明 |
|------|------|
| `task_code` | 任务编码 |
| `task_name` | 任务名称 |
| `type` | 巡视类型 |
| `priority` | 优先级 |
| `device_level` | 设备层级 |
| `device_list` | 设备列表，逗号分隔 |
| `fixed_start_time` | 定期开始时间 |
| `cycle_*` | 周期任务参数 |
| `interval_*` | 间隔任务参数 |
| `invalid_*` | 不可用时间窗口 |
| `isenable` | `0` 启用、`1` 禁用、`2` 删除 |

任务控制成功后会返回 `task_patrolled_id`，并异步上报 `41-0` 任务状态数据。

### 模型同步

上级系统可通过 `61-1` 到 `61-12` 拉取模型或文件路径，`ispagent` 返回对应 FTPS 文件路径。

| Type | Command | 模型类型 | 返回字段 |
|------|---------|----------|----------|
| 61 | 1 | 区域主机及边缘节点装置模型 | `host_file_path` |
| 61 | 2 | 机器人模型 | `robot_file_path` |
| 61 | 3 | 摄像机模型及硬盘录像机模型 | `video_file_path` |
| 61 | 4 | 点位模型 | `device_file_path` |
| 61 | 5 | 无人机模型及无人机机巢模型 | `drone_file_path` |
| 61 | 6 | 声纹模型 | `voice_file_path` |
| 61 | 7 | 任务文件 | `task_file_path` |
| 61 | 8 | 检修区域配置文件 | `overhaularea_file_path` |
| 61 | 9 | 地图文件 | `map_file_path` |
| 61 | 10 | 维护记录文件 | `maintain_file_path` |
| 61 | 11 | 联动配置文件 | `source_file_path` |
| 61 | 12 | 告警阈值模型 | `alarm_file_path` |

当前实现中，机器人模型和点位模型可由 `common/isp` 流式生成 XML 后上传 FTPS；地图模型读取 `local/<stationCode>/map.jpeg` 后上传；其他模型默认返回约定路径。

### 机器人与设备控制

`ispagent` 已注册并记录以下控制类指令，当前实现以协议接入和日志确认作为主，实际硬件动作需接入设备控制 SDK 后扩展：

| Type | 范围 | 示例 |
|------|------|------|
| 1 | 机器人本体 | 远方复位、系统自检、一键返航、手动充电、控制模式切换、控制权获得/释放 |
| 2 | 机器人车体 | 前进、后退、左转、右转、停止、升降、平移、步态切换 |
| 3 | 机器人云台 | 上仰、下俯、左右转、升降、预置位、停止、复位 |
| 4 | 辅助设备 | 红外电源、雨刷、超声、红外射灯、辅助照明 |
| 21 | 可见光摄像机 | 变焦、聚焦、自动聚焦、抓图、重启、录像、倍率设置 |
| 22 | 红外热像仪 | 设定焦距、自动聚焦、抓图、重启 |
| 23 | 局放传感器 | 伸长、收缩、停止、复位 |

## 对外 gRPC 接口

`ispagent` 通过 gRPC 暴露内部调用接口：

| RPC | 说明 |
|-----|------|
| `ExecuteCommand` | 通用指令透传，调用方指定 `type`、`command`、`code` 和 `items` |
| `SendPatrolDeviceRunData` | 写入巡视装置运行数据缓存，后续按间隔上报 |
| `SendPatrolDeviceStatusData` | 写入巡视装置状态数据缓存，后续按间隔上报 |
| `SendPatrolDeviceCoordinates` | 写入巡视装置坐标缓存，后续按间隔上报 |
| `ListTaskExecutions` | 查询任务未来执行时间，用于验证周期规则 |
| `ListTaskConfigs` | 分页查询 ISP 任务配置 |
| `TestFTPSUpload` | 测试模型同步 FTPS 上传 |
| `ListFTPSDirectory` | 查看 FTPS 远端目录 |

## 配置项

核心配置结构位于 `app/ispagent/internal/config/config.go`。

```yaml
IspSetting:
  ServerAddr: 127.0.0.1:9000
  SendCode: local-device-code
  RegisterReceiveCode: remote-system-code
  RootName: PatrolDevice
  HeartbeatInterval: 60s
  RequestTimeout: 10s
  ReconnectInterval: 3s
  MaxFrameLength: 1048576
  DebugLog: false

ModelSync:
  FTPS:
    Address: 127.0.0.1:990
    Username: user
    Password: pass
    RemoteDir: /isp-models
    TLSMode: implicit
    InsecureSkipVerify: true
    Timeout: 30s
    UseTemporaryFile: true
```

## 实现边界

- `common/isp` 提供协议帧编解码、XML 动态 Item 解析、Type/Command 常量和模型 XML 生成。
- `app/ispagent/internal/isp` 负责 TCP 长连接、注册、心跳、请求响应匹配和消息路由。
- `app/ispagent/internal/handler` 负责任务下发、任务控制、模型同步和控制指令处理。
- 未匹配的入站消息会记录日志并返回 `251-3` 通用成功应答，避免上级系统阻塞。
- 控制类指令当前未直接驱动机器人硬件；如需落地动作，需要在 handler 中接入实际设备控制链路。

## 参考资料

- 项目实现：`common/isp`、`app/ispagent`
- 项目规范：`.trellis/spec/backend/isp-guidelines.md`
- 公开资料关键词：`区域型变电站远程智能巡视系统技术规范`、`远程智能巡视集中监控系统技术规范`、`变电站远程智能巡视系统技术规范`
