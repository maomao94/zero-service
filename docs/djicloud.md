# DJI 云平台服务

## 概述

`djicloud` 是面向 DJI Dock 3 Cloud API 的云平台服务，封装大疆上云 MQTT Topic 与 method，统一处理设备侧 ACK、上行事件和在线状态。对外提供 gRPC 接口供业务系统调用，对内通过 `common/djisdk` 与 MQTT Broker 通信。

**核心能力**：
- 标准下行指令：属性设置、直播推流、媒体上传、航线任务、远程调试、固件升级、远程日志、配置更新、自定义飞行区、PSDK/ESDK 透传、远程解禁
- DRC 指令飞行：进入/退出 DRC 模式、飞行/负载控制权、相机/云台控制、航点飞行、杆量控制、心跳
- 上行消息处理：订阅 services_reply、events、osd、state、status、requests、drc/up 等主题，持久化到数据库
- 平台侧能力：设备在线状态缓存、拓扑关系、航线进度、HMS 告警、DRC 会话推送

## 架构

```
业务系统 --> djicloud gRPC (21012) --> common/djisdk --> MQTT Broker --> DJI Dock 3/飞行器
                                   ^              |
                                   |              v
                            services_reply / events / osd / state / status / requests / drc/up
                                   |
                                   v
                              PostgreSQL (设备状态/遥测/事件持久化)
```

**两条下行通道**：

| 通道 | 主题 | 模式 | 用途 |
|------|------|------|------|
| services | `thing/product/{sn}/services` | 请求-应答（等待 services_reply） | 属性设置、航线、直播、媒体、调试、固件、日志、DRC 模式切换等 |
| drc/down | `thing/product/{sn}/drc/down` | 即发即忘（无设备回执） | 杆量控制、紧急停桨、强制降落、DRC 相机参数设置等高频指令 |

**模块边界**：
- `app/djicloud/`：gRPC 服务定义（proto + go-zero 生成骨架）、业务 Logic、MQTT 上行 hooks、数据模型
- `common/djisdk/`：DJI Cloud API MQTT topic 构造、协议体序列化、强类型 Client、pending ACK 管理、DRC 会话管理

## 目录结构

```
app/djicloud/
├── djicloud.proto              # gRPC 服务与消息定义
├── gen.sh                      # proto 代码生成脚本
├── djicloud.go                 # go-zero 启动入口
├── etc/djicloud.yaml           # 服务配置
├── internal/
│   ├── config/config.go        # Config 结构体定义
│   ├── svc/
│   │   ├── servicecontext.go   # ServiceContext 初始化（DB、Client、Cache、Push）
│   │   └── device_online_refresher.go  # 定时清理离线设备
│   ├── hooks/                  # MQTT 上行消息处理钩子
│   │   ├── register.go         # 组装所有钩子为 djisdk.ClientOption
│   │   ├── event_notify_up.go  # 事件处理（flighttask_progress、HMS、返航、远程日志等）
│   │   ├── telemetry_up.go     # OSD/State 遥测处理
│   │   ├── sys_status_up.go    # 系统状态处理（update_topo、在线/离线）
│   │   ├── mqtt_request_up.go  # 设备主动请求处理
│   │   ├── mqtt_drc_up.go      # DRC 上行处理
│   │   ├── online_cache.go     # 在线状态缓存
│   │   └── store_helper.go     # 数据库持久化工具
│   ├── logic/                  # 每个 RPC 一个 logic 文件
│   │   ├── helper.go           # 公共工具（errRes、okRes、模型转换）
│   │   └── drchelper.go        # DRC MQTT Broker 地址构造
│   └── server/                 # go-zero 生成的服务骨架
├── model/gormmodel/            # 数据模型
│   ├── dji_device.go           # 设备主表 + 拓扑关系表
│   ├── dji_osd_state.go        # OSD/State 遥测快照表
│   └── dji_event.go            # 事件记录表（HMS、航线、返航、日志、DRC）
└── deploy.sh / Dockerfile      # 部署相关

common/djisdk/
├── client.go                   # Client 核心结构体、构造、SendCommand、SubscribeAll
├── drc.go                      # DRC 会话管理器（心跳、seq、生命周期钩子）
├── topic.go                    # MQTT Topic 构造
├── method.go                   # DJI method 常量定义
├── protocol.go                 # 通用协议体（services、events、osd、state、requests）
├── protocol_drc.go             # DRC 协议体（drc/down、drc/up 载荷）
├── option.go                   # Client 配置选项（钩子注册、TTL、DRC 配置等）
├── handler.go                  # 上行消息分发与回复控制
├── error.go                    # DJIError、PlatformError 错误类型
├── error_descriptions.go       # 约 450 条 DJI 错误码本地化描述
└── *_test.go                   # 单元测试
```

## RPC 接口

### 属性设置

| 方法 | 说明 | 主题 |
|------|------|------|
| `PropertySet` | 设置设备属性（JSON 键值对） | property/set |

### 直播功能

| 方法 | 说明 | 主题 |
|------|------|------|
| `LiveStartPush` | 开始直播推流 | services |
| `LiveStopPush` | 停止直播推流 | services |
| `LiveSetQuality` | 设置直播画质 | services |
| `LiveLensChange` | 切换直播镜头 | services |
| `LiveCameraChange` | 切换直播相机 | services |

### 媒体功能

| 方法 | 说明 | 主题 |
|------|------|------|
| `MediaUploadFlighttaskMediaPrioritize` | 优先上传指定航线任务媒体 | services |
| `MediaFastUpload` | 快速上传指定媒体文件 | services |
| `MediaHighestPriorityUploadFlighttask` | 最高优先级上传航线任务媒体 | services |

### 航线功能

| 方法 | 说明 | 主题 |
|------|------|------|
| `FlightTaskPrepare` | 航线任务准备（下发 WPML 航线文件） | services |
| `FlightTaskExecute` | 航线任务执行 | services |
| `FlightTaskUndo` | 取消航线任务 | services |
| `PauseFlightTask` | 暂停航线任务 | services |
| `FlightTaskRecovery` | 恢复已暂停的航线任务 | services |
| `StopFlightTask` | 强制停止当前航线任务 | services |
| `ReturnHome` | 一键返航 | services |
| `ReturnHomeCancelAutoReturn` | 取消自动返航 | services |
| `ReturnSpecificHome` | 返航至指定备降点 | services |

### 远程调试

| 方法 | 说明 | 主题 |
|------|------|------|
| `DebugModeOpen` / `DebugModeClose` | 机巢调试模式开关 | services |
| `CoverOpen` / `CoverClose` / `CoverForceClose` | 机巢舱盖控制 | services |
| `DroneOpen` / `DroneClose` | 无人机电源控制 | services |
| `DeviceReboot` | 重启机巢设备 | services |
| `ChargeOpen` / `ChargeClose` | 充电功能控制 | services |
| `SupplementLightOpen` / `SupplementLightClose` | 补光灯控制 | services |
| `DroneFormat` / `DeviceFormat` | 无人机/机巢存储格式化 | services |
| `BatteryStoreModeSwitch` | 电池保养存储模式切换 | services |
| `AlarmStateSwitch` | 声光报警开关 | services |
| `AirConditionerModeSwitch` | 空调模式切换 | services |
| `BatteryMaintenanceSwitch` | 电池保养功能开关 | services |

### 固件升级

| 方法 | 说明 | 主题 |
|------|------|------|
| `OtaCreate` | 创建固件升级任务 | services |

### 远程日志

| 方法 | 说明 | 主题 |
|------|------|------|
| `RemoteLogFileList` | 查询可上传的远程日志文件列表 | services |
| `RemoteLogFileUploadStart` | 开始上传远程日志文件 | services |
| `RemoteLogFileUploadUpdate` | 更新远程日志上传任务 | services |
| `RemoteLogFileUploadCancel` | 取消远程日志上传任务 | services |

### 配置更新

| 方法 | 说明 | 主题 |
|------|------|------|
| `ConfigUpdate` | 下发设备配置更新 | services |

### 自定义飞行区

| 方法 | 说明 | 主题 |
|------|------|------|
| `FlightAreasUpdate` | 触发自定义飞行区文件更新 | services |

### PSDK/ESDK 透传

| 方法 | 说明 | 主题 |
|------|------|------|
| `PsdkUIResourceUpload` | PSDK UI 资源上传 | services |
| `CustomDataTransmissionToPsdk` | 自定义数据透传至 PSDK 负载设备 | services |
| `CustomDataTransmissionToEsdk` | 自定义数据透传至 ESDK 设备 | services |

### 远程解禁

| 方法 | 说明 | 主题 |
|------|------|------|
| `UnlockLicenseSwitch` | 启用/禁用解禁证书 | services |
| `UnlockLicenseUpdate` | 更新解禁证书 | services |
| `UnlockLicenseList` | 获取解禁证书列表 | services |

### DRC 指令飞行 — 服务通道（services，等待 ACK）

| 方法 | 说明 |
|------|------|
| `DrcModeEnter` | 进入 DRC 模式 |
| `DrcModeExit` | 退出 DRC 模式 |
| `FlightAuthorityGrab` | 获取飞行控制权 |
| `PayloadAuthorityGrab` | 获取负载控制权 |
| `FlyToPoint` | 飞向指定航点 |
| `FlyToPointStop` | 停止飞向航点 |
| `TakeoffToPoint` | 一键起飞到指定坐标 |
| `CameraModeSwitch` | 切换相机拍摄模式 |
| `CameraPhotoTake` / `CameraPhotoStop` | 拍照/停止连续拍照 |
| `CameraRecordingStart` / `CameraRecordingStop` | 开始/停止录像 |
| `CameraFocalLengthSet` | 设置相机焦距 |
| `GimbalReset` | 重置云台角度 |
| `CameraAim` | 相机指点对准 |
| `CameraPointFocusAction` | 相机指点对焦 |
| `CameraScreenSplit` | 相机分屏控制 |
| `CameraPhotoStorageSet` / `CameraVideoStorageSet` | 拍照/录像存储位置设置 |
| `CameraLookAt` | 相机朝向指定坐标 |
| `CameraScreenDrag` | 相机屏幕拖拽 |
| `CameraIrMeteringPoint` | 红外测温点设置 |
| `CameraIrMeteringArea` | 红外区域测温设置 |

### DRC 指令飞行 — drc/down 通道（即发即忘）

| 方法 | 说明 |
|------|------|
| `StickControl` | 杆量控制（建议 5~10 Hz） |
| `DroneEmergencyStop` | 飞行器紧急停桨（需配置开启） |
| `DrcForceLanding` | 强制降落 |
| `DrcEmergencyLanding` | 紧急降落 |
| `DrcLinkageZoomSet` | 红外联动变焦 |
| `DrcVideoResolutionSet` | 设置视频分辨率 |
| `DrcIntervalPhotoSet` | 设置定时拍 |
| `DrcInitialStateSubscribe` | 订阅 DRC 初始状态 |
| `DrcNightLightsStateSet` | 夜航灯状态设置 |
| `DrcStealthStateSet` | 隐蔽模式状态设置 |
| `DrcCameraApertureValueSet` | 设置相机光圈 |
| `DrcCameraShutterSet` | 设置相机快门 |
| `DrcCameraIsoSet` | 设置相机 ISO |
| `DrcCameraMechanicalShutterSet` | 设置机械快门 |
| `DrcCameraDewarpingSet` | 镜头去畸变设置 |

### 平台能力接口

| 方法 | 说明 |
|------|------|
| `IsDeviceOnline` | 查询设备在线状态（内存缓存 + 数据库兜底） |
| `ListDevices` | 查询设备列表（机巢、无人机、负载） |
| `GetDeviceDetail` | 查询设备详情（聚合基础信息 + OSD + State + 拓扑） |
| `GetDeviceOsdSnapshot` | 查询设备最近 OSD 遥测快照 |
| `GetDeviceStateSnapshot` | 查询设备最近 State 状态快照 |
| `ListHmsAlerts` | 查询 HMS 告警记录 |
| `AckHmsAlert` | 确认 HMS 告警 |
| `ListFlightTaskProgress` | 查询航线任务进度记录 |
| `GetFlightTaskProgressLast` | 查询最近一条航线任务进度 |
| `QueryDrcStatus` | 查询设备 DRC 运行状态 |

### 通用响应

所有标准 DJI 下行接口返回 `CommonRes`：

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int32 | 结果码，0 成功，-1 失败 |
| `message` | string | 结果描述，失败时为错误详情（含中文描述） |
| `tid` | string | 事务 ID，追踪 MQTT 链路 |
| `reason_code` | int32 | DJI 设备原始错误码，成功时为 0 |

## 配置说明

```yaml
Name: djicloud.rpc
ListenOn: 0.0.0.0:21012
Timeout: 10000                       # gRPC 超时（ms）
Mode: dev                            # dev/test 模式下自动迁移数据库

Log:
  Encoding: plain
  Path: /opt/logs/djicloud
  Level: info
  KeepDays: 300

# DJI 上云 MQTT Broker 配置
Dji:
  MqttConfig:
    Broker:                           # MQTT Broker 地址列表
      - tcp://127.0.0.1:1883
    ClientID: dji-cloud-001
    Username: "mqtt"
    Password: "password"
    Qos: 0                            # MQTT QoS（0/1/2）
  PendingTTL: 30s                     # 下行命令等待 ACK 超时
  Reply:                              # 上行消息自动回复开关
    EnableEventReply: true            # events 回复（推荐开启）
    EnableStatusReply: false          # status 回复
    EnableRequestReply: false         # requests 回复
  Drc:                                # DRC 会话配置
    HeartbeatInterval: 2s             # 心跳发送间隔
    HeartbeatTimeout: 300s            # 心跳超时（超时触发 SessionExpired 钩子）
    Address: 127.0.0.1:21012          # 传递给设备的 DRC MQTT 公网地址

# 数据库配置
DB:
  DataSource: postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable&TimeZone=Asia/Shanghai
  MaxIdleConns: 10
  MaxOpenConns: 100
  SlowThreshold: 200ms
  LogLevel: error
  ParameterizedQueries: false
  IgnoreRecordNotFoundError: false

# 遥测日志控制
Telemetry:
  DisableOsdSQLTrace: true            # 抑制高频 OSD 写入的 GORM SQL trace 日志

# 危险操作开关（默认全部关闭）
DangerousOps:
  EnableDroneEmergencyStop: false     # 紧急停桨（立即停止所有电机，极端危险）

# SocketPush 配置（可选，未配置时不推送 DRC 会话事件）
SocketPushConf:
  Endpoints:
    - 127.0.0.1:25002
  Timeout: 10000
  NonBlock: true

# Nacos 服务注册（可选）
NacosConfig:
  IsRegister: false
  Host: 127.0.0.1
  Port: 8848
  Username: nacos
  PassWord: nacos
  NamespaceId: public
  ServiceName: djicloud
```

## 数据模型

所有模型位于 `app/djicloud/model/gormmodel/`，dev/test 模式下自动迁移。

| 表 | 模型 | 说明 |
|---|------|------|
| `dji_device` | `DjiDevice` | 设备主表，记录所有出现过设备的 SN、别名、固件/硬件版本、在线状态、首次/最后上线时间 |
| `dji_device_topo` | `DjiDeviceTopo` | 网关与子设备拓扑关系，支持蛙跳场景（同一子设备可关联多个网关） |
| `dji_device_osd_snapshot` | `DjiDeviceOsdSnapshot` | OSD 遥测快照，每设备一行，原始 JSON 存储 |
| `dji_device_state_snapshot` | `DjiDeviceStateSnapshot` | State 状态快照，每设备一行，原始 JSON 存储 |
| `dji_hms_alert` | `DjiHmsAlert` | HMS 告警记录，支持确认操作 |
| `dji_dock_flight_task` | `DjiDockFlightTask` | 航线任务进度快照，按 gateway_sn + flight_id 去重 |
| `dji_dock_device_flight_task_state` | `DjiDockDeviceFlightTaskState` | 机巢当前航线任务状态，按 gateway_sn 去重 |
| `dji_flight_task_ready` | `DjiFlightTaskReady` | 任务就绪事件历史记录 |
| `dji_remote_log_event` | `DjiRemoteLogEvent` | 远程日志上传进度事件历史记录 |
| `dji_return_home_event` | `DjiReturnHomeEvent` | 返航信息事件历史记录 |
| `dji_drc_up_event` | `DjiDrcUpEvent` | DRC 上行事件历史记录（高频心跳类型丢弃不存） |

## MQTT 上行处理

SDK 通过 `SubscribeAll()` 订阅 6 个通配符主题，每个主题对应一个 hooks 文件：

| 主题模式 | 处理文件 | 主要职责 |
|----------|----------|----------|
| `thing/product/+/osd` | `telemetry_up.go` | 刷新设备在线缓存、更新 OSD 快照、同步设备版本信息 |
| `thing/product/+/state` | `telemetry_up.go` | 更新 State 快照、提取固件/硬件版本 |
| `thing/product/+/events` | `event_notify_up.go` | 分发航线进度、任务就绪、返航信息、HMS、远程日志等事件 |
| `sys/product/+/status` | `sys_status_up.go` | 处理 update_topo 拓扑变更、设备在线/离线 |
| `thing/product/+/requests` | `mqtt_request_up.go` | 响应设备主动拉取平台数据的请求 |
| `thing/product/+/drc/up` | `mqtt_drc_up.go` | 记录 DRC 上行事件（避障、时延、OSD、心跳等） |

## 核心约定

### 在线状态管理

双层判断机制：
1. **内存缓存**（60 秒 TTL）：OSD 上行写入，供高频 `IsDeviceOnline` 查询
2. **数据库快照**：`dji_device.is_online` + `last_online_at` 兜底
3. **定时清理**：`device_online_refresher` 每 15 秒将超时设备标记为离线

### DRC 会话管理

- SDK 内置 `drcManager`，进入 DRC 模式后自动维护心跳（默认 2 秒间隔）
- 心跳超时（默认 300 秒）触发 `SessionExpired` 钩子，可通过 SocketPush 推送到 WebSocket 房间
- `StickControl` 等 drc/down 指令由平台内部管理 seq，调用方无需传入
- `DrcModeEnter`/`DrcModeExit` 走 services 通道（等待设备 ACK），与 drc/down 通道不同

### 错误处理

- 设备错误码通过 `DJIError` 类型包装，`error_descriptions.go` 提供约 450 条中文描述
- Logic 层通过 `errRes(tid, err)` 将 SDK 错误映射到 `CommonRes`，自动提取 `reason_code` 和中文描述
- 部分模块（HMS、Organization、AirSense、PSDK）仅处理上行消息，gRPC 暂未暴露下行接口

### 蛙跳拓扑

同一飞行器可被多个机巢绑定，`DjiDeviceTopo` 允许同一 `sub_device_sn` 存在多条记录：
- `DjiDevice.GatewaySn` 仅由 OSD/State 更新（反映当前通信通道）
- 拓扑绑定关系由 `DjiDeviceTopo` 独立维护
- `update_topo` 上行不覆盖非 dock 域设备的 `GatewaySn`

### 原始 JSON 存储

OSD/State 快照以原始 JSON（`jsonb`）存储，不解析为列。避免协议字段变更时的 schema 迁移，同时保留完整遥测数据用于排查。

## 扩展约定

新增 DJI 标准能力或云平台组合接口时：

1. 先改 `app/djicloud/djicloud.proto`，确认 method 与 DJI Cloud API 文档一致
2. 执行 `app/djicloud/gen.sh` 生成 gRPC 骨架
3. 在 `common/djisdk/method.go` 添加 method 常量，`protocol.go` 或 `protocol_drc.go` 添加载荷类型
4. 在 `common/djisdk/client.go` 添加便利方法（DRC 下行方法加在 `drc.go`）
5. 在 `internal/logic/` 添加 logic 文件，按现有模式委托 SDK Client
6. 如需持久化上行数据，在 `internal/hooks/` 添加处理逻辑，在 `model/gormmodel/` 添加表结构
7. 不要手写或随意修改生成文件
