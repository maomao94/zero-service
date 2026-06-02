# DJI 云平台服务

## 概述

`djicloud` 是面向 DJI Dock3 Cloud API 的云平台服务，封装大疆上云 MQTT Topic 与 method，统一处理设备侧 ACK、上行事件和在线状态。

**核心能力**：
- 标准下行指令：属性设置、直播推流、媒体上传、航线任务、远程调试、固件升级、远程日志
- DRC 指令飞行：进入/退出 DRC、飞行/负载控制权、杆量控制、心跳、飞向航点
- 上行消息处理：订阅 services_reply、events、osd、state、drc/up 等主题
- 平台侧能力：设备在线状态缓存、航线进度缓存、危险操作配置开关

## 架构

```
业务系统 --> djicloud gRPC --> common/djisdk --> MQTT Broker --> DJI Dock3/飞行器
                         ^              |
                         |              v
                  services_reply / events / osd / state / drc/up
```

**模块边界**：
- `app/djicloud/`：gRPC 服务定义、go-zero 生成骨架、业务 Logic 与配置
- `common/djisdk/`：DJI Cloud API MQTT topic、协议体、强类型 Client、pending ACK 与上行回调

## RPC 接口

### 属性设置

| 方法 | 说明 |
|------|------|
| `SetProperty` | 设置设备属性（JSON 键值对） |

### 直播功能

| 方法 | 说明 |
|------|------|
| `LiveStartPush` | 开始直播推流 |
| `LiveStopPush` | 停止直播推流 |
| `LiveSetQuality` | 设置直播画质 |
| `LiveLensChange` | 切换直播镜头 |
| `LiveCameraChange` | 切换直播相机 |

### 媒体功能

| 方法 | 说明 |
|------|------|
| `MediaUploadFlighttaskMediaPrioritize` | 优先上传指定航线任务媒体 |
| `MediaFastUpload` | 快速上传指定媒体文件 |
| `MediaHighestPriorityUploadFlighttask` | 最高优先级上传航线任务媒体 |

### 航线功能

| 方法 | 说明 |
|------|------|
| `FlightTaskPrepare` | 航线任务准备 |
| `FlightTaskExecute` | 航线任务执行 |
| `CancelFlightTask` | 取消航线任务 |
| `PauseFlightTask` | 暂停航线任务 |
| `ResumeFlightTask` | 恢复已暂停的航线任务 |
| `StopFlightTask` | 强制停止当前航线任务 |
| `ReturnHome` | 一键返航 |
| `ReturnHomeCancelAutoReturn` | 取消自动返航 |
| `ReturnSpecificHome` | 返航至指定备降点 |

### 远程调试

| 方法 | 说明 |
|------|------|
| `DebugModeOpen` | 开启机巢调试模式 |
| `DebugModeClose` | 关闭机巢调试模式 |
| `CoverOpen` / `CoverClose` | 机巢舱盖控制 |
| `DroneOpen` / `DroneClose` | 无人机电源控制 |
| `DeviceReboot` | 重启机巢设备 |
| `ChargeOpen` / `ChargeClose` | 充电功能控制 |
| `SupplementLightOpen` / `SupplementLightClose` | 补光灯控制 |

### 固件升级

| 方法 | 说明 |
|------|------|
| `OtaCreate` | 创建固件升级任务 |

### 远程日志

| 方法 | 说明 |
|------|------|
| `RemoteLogFileList` | 查询可上传的远程日志文件列表 |

### DRC 指令飞行

| 方法 | 说明 |
|------|------|
| `DrcOpen` | 进入 DRC 模式 |
| `DrcClose` | 退出 DRC 模式 |
| `FlightControl` | 飞行控制权操作 |
| `PayloadControl` | 负载控制权操作 |
| `JoystickCommand` | 杆量控制指令 |
| `DrcHeartbeat` | DRC 心跳 |
| `FlyToWaypoint` | 飞向指定航点 |

## 配置说明

```yaml
# DJI 上云 MQTT Broker 配置
MqttConfig:
  Broker: tcp://your-mqtt-broker:1883
  ClientID: your-client-id
  Username: your-username
  Password: your-password
  QoS: 1

# 下行命令 ACK 超时控制
AckTimeout: 30s
PendingTTL: 60s

# 上行 reply 自动回复开关
UpstreamReply:
  EventsReply: true
  StatusReply: true
  RequestsReply: true

# 危险操作开关（默认关闭）
DangerousOps:
  EnableDroneEmergencyStop: false
```

## 扩展约定

新增 DJI 标准能力或云平台组合接口时：

1. 先改 `app/djicloud/djicloud.proto`
2. 执行 `app/djicloud/gen.sh`
3. 在 `internal/logic/` 和 `common/djisdk/` 中补业务实现
4. 不要手写或随意修改生成文件
