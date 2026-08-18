# Zero-Service Context

工业物联网与边缘集成场景的 Go 微服务集合，基于 go-zero 构建，覆盖工业协议接入、流数据处理、任务调度、实时通信和无人机机场等能力。本词汇表定义项目特有术语的规范写法，供文档与 agent 使用同一语言。

## Language

**IEC 104**:
IEC 60870-5-104 协议，基于 TCP 的电力远动通信协议，用于与工业从站采集数据和下发控制命令。
_Avoid_: IEC-104、iec104（指代码目录时可保留小写）

**ASDU**:
应用服务数据单元，IEC 104 协议中承载监视或控制数据的基本报文单元，由 TypeId 标识类型。
_Avoid_: asdu（指 Topic 时可保留小写）

**ASDU 合并**:
iecstash 按字节数聚合多条 Kafka ASDU 消息后批量转发的机制。
_Avoid_: ASDU 聚合

**三通道分发**:
ieccaller 将采集数据通过 Kafka、MQTT、gRPC 三条通道并行推送的分发模式，通道可独立启用。
_Avoid_: 三协议推送（不够精确时）

**弱校验推送模式**:
ieccaller 的推送判定规则：只有配置了点位且 `enable_push=0` 时才不推送，其他情况一律推送。
_Avoid_: 弱校验

**ISP**:
变电站远程智能巡视协议，提供服务端与代理，连接上级平台和下级巡检设备。
_Avoid_: isp 协议

**RRULE**:
RFC 5545 周期规则，Trigger 用它表达 Plan 与 CronJob 的周期调度集合语义。
_Avoid_: rrule、RRULE 规则

**RRULE Set**:
由 DTSTART、RRULE、RDATE、EXDATE 组成的完整周期集合，Plan 与 CronJob 共用 `(RRULE ∪ specifiedTimes) - excludedTimes - expanded(excludeDates)` 语义。
_Avoid_: 规则集

**Socket.IO**:
浏览器与服务器间实时双向通信协议，项目用于 socketgtw 网关与前端长连接。
_Avoid_: SocketIO、socketio、Socket io

**DJI Cloud API**:
大疆上云 MQTT Topic 与 method 体系，djicloud 服务封装其服务端对接。
_Avoid_: DJI 云 API

**DRC**:
DJI 云平台的远程控制通道，`drc/down` 主题用于杆量控制等高频即发即忘指令。
_Avoid_: drc（指代码可保留小写）

**KML/KMZ**:
无人机航点任务的航线和航点参数文件格式，KML 为 XML 标记语言，KMZ 为其 ZIP 压缩格式。
_Avoid_: kml

**CronJob 排除语义**:
RRULE 集合中排除条件优先且同一秒去重的规则：指定时间编译为 RDATE，精确时间和整日排除编译为 EXDATE。
_Avoid_: 排除规则

**Plan/Batch/ExecItem**:
Trigger 计划任务的三级持久化模型：计划、计划执行时间批次、业务执行单元。
_Avoid_: 计划任务三级模型

**asynq 异步任务**:
Trigger 基于 asynq 与 Redis 的一次性或延时回调任务，队列成功不等于业务回调成功。
_Avoid_: 异步任务（与"计划任务"混淆时）

**节假日**:
Trigger 内置的中国大陆节假日规则与数据源管理能力，用于排除法定节假日执行。
_Avoid_: 假期

**弱校验**:
ieccaller 对点位推送采取宽松判定的机制，只有明确配置且关闭推送时才跳过，详见"弱校验推送模式"。
_Avoid_: 宽松校验

**Dock 3**:
DJI 无人机机场产品线，djicloud 服务面向其 Cloud API 对接。
_Avoid_: Dock3、dock3

**流事件协议**:
facade/streamevent 定义的跨语言 gRPC 事件协议，承载 IEC 104 数据与 Trigger 业务回调。
_Avoid_: 流事件 gRPC

**RTU**:
Remote Terminal Unit，远程终端单元，工业场景中的采集与执行设备。
_Avoid_: rtu

## Services

**gtw**:
BFF/API 网关，HTTP + gRPC-Gateway 入口，转发前端请求到领域服务。
_Avoid_: gateway、BFF 网关

**ieccaller**:
IEC 104 主站服务，与多个从站建立 TCP 连接采集数据，经 Kafka、MQTT、gRPC 三通道推送。
_Avoid_: IEC 主站

**iecstash**:
IEC 104 数据合并服务，消费 Kafka ASDU 消息，按字节聚合后批量转发。
_Avoid_: 数据合并服务

**streamevent**:
流事件处理服务，接收流事件、点位过滤、事件分发与 TDengine 落库。
_Avoid_: 流事件服务

**iecagent**:
IEC 104 从站模拟器，用于开发调试和测试。
_Avoid_: IEC 代理

**trigger**:
统一任务调度服务，提供 asynq 异步任务、Plan 计划任务和 RRULE CronJob 三类能力。
_Avoid_: 任务调度服务、定时任务服务

**djicloud**:
DJI Dock 3 云平台接入服务，封装大疆上云 MQTT Topic 与 method。
_Avoid_: 大疆服务

**socketgtw**:
Socket.IO 网关服务，管理 WebSocket 长连接、房间、消息路由与 Token 认证。
_Avoid_: SocketIO 网关（写法不规范时）

**socketpush**:
Socket 推送服务，为后端服务提供 gRPC 推送接口，经 socketgtw 推送到前端。
_Avoid_: 推送服务

**ispagent**:
ISP 巡检协议代理，代理连接上级平台。
_Avoid_: ISP 代理（中英文混排时可）

**ispserver**:
ISP 巡检协议服务端，连接下级巡检设备。
_Avoid_: ISP 服务端

**file**:
文件与 OSS 服务，提供分片文件传输、对象存储管理与中继。
_Avoid_: 文件服务

**gis**:
地理信息服务，提供 H3、GeoHash、电子围栏和坐标转换能力。
_Avoid_: 地理信息服务

**podengine**:
容器管理引擎服务，基于 Docker SDK 管理容器生命周期。
_Avoid_: 容器服务

**alarm**:
告警服务，通过飞书推送告警消息。
_Avoid_: 告警通知服务

**lalhook**:
LAL 流媒体回调服务，接收 LAL HTTP 回调事件。
_Avoid_: 流媒体回调

**lalproxy**:
LAL 流媒体代理服务，提供 gRPC 接口。
_Avoid_: 流媒体代理

**logdump**:
日志导出服务，gRPC 到 logx 的日志汇聚桥接。
_Avoid_: 日志导出

**bridgedump**:
南瑞反向隔离装置桥接服务。
_Avoid_: 反向隔离桥接

**bridgegtw**:
gRPC-Gateway 代理转发服务。
_Avoid_: 网关代理（与其他网关混淆时）

**bridgekafka**:
Kafka 桥接服务，将消息桥接到其他协议或通道。
_Avoid_: Kafka 桥接

**bridgemodbus**:
Modbus TCP/RTU 桥接服务。
_Avoid_: Modbus 桥接

**bridgemqtt**:
MQTT 桥接服务。
_Avoid_: MQTT 桥接

**zero.rpc**:
核心业务 RPC 服务，位于 zerorpc/。
_Avoid_: zero

**xfusionmock**:
X-Fusion 模拟服务，用于 Mock 与 Demo。
_Avoid_: 模拟服务

**streamevent.rpc**:
streamevent 服务的 gRPC 契约名，业务回调经其下发。
_Avoid_: StreamEvent（指服务时）

## Protocols

**Modbus TCP/RTU**:
工业设备通信协议，支持 TCP 与串口 RTU 两种传输方式。
_Avoid_: modbus

**MQTT**:
轻量发布订阅消息协议，用于物联网场景与 DJI 云平台通信。
_Avoid_: mqtt

**gRPC**:
Google 的高性能跨语言 RPC 框架，项目服务间主要通信方式。
_Avoid_: grpc

**Kafka**:
分布式消息队列，用于高吞吐数据分发。
_Avoid_: kafka

**TDengine**:
时序数据库，用于 IEC 104 遥信遥测数据存储。
_Avoid_: tdengine

**H3**:
Uber 提出的六边形地理索引系统，gis 服务用于围栏与空间检索。
_Avoid_: h3

**GeoHash**:
地理坐标编码系统，将经纬度编码为字符串。
_Avoid_: geohash

**OpenTelemetry**:
可观测性标准，项目用于链路追踪与指标。
_Avoid_: otel

**Nacos**:
配置中心与服务注册中心。
_Avoid_: nacos

## Concepts

**电子围栏**:
gis 服务管理的空间围栏，支持点内检测、邻近查询与 H3 格网生成。
_Avoid_: 围栏

**DRC 指令飞行**:
通过 DRC 下行通道对 DJI 飞行器进行高频即发即忘的杆量控制指令飞行。
_Avoid_: 指令飞行

**航线任务**:
DJI 云平台按 KML/KMZ 航点文件执行的飞行任务。
_Avoid_: 航点任务

**直播推流**:
DJI 云平台的实时视频推流能力，经 lal 流媒体链路转发。
_Avoid_: 推流

**对象存储**:
OSS 类存储服务，file 服务集成 MinIO、阿里 OSS、腾讯 COS。
_Avoid_: oss

**分片上传**:
file 服务将大文件分片传输后合并的机制。
_Avoid_: 分片文件

**容器生命周期**:
podengine 对 Docker 容器的创建、启动、停止、重启、删除管理。
_Avoid_: 容器管理

**服务发现**:
基于 Nacos 的服务注册与发现机制。
_Avoid_: 注册中心

**集群路由**:
IEC 104 控制命令在集群模式下的 MQTT 广播路由。
_Avoid_: 广播路由

**链路追踪**:
基于 OpenTelemetry 的请求链路追踪，traceId 随消息流转。
_Avoid_: tracing

**反向隔离**:
南瑞反向隔离装置，bridgedump 对接的网络安全设备。
_Avoid_: 隔离装置

**房间**:
Socket.IO 网关的消息分组单元，客户端加入房间后接收该房间广播。
_Avoid_: room（代码场景外）

**会话**:
Socket.IO 网关中一次客户端长连接，以 Session ID 标识。
_Avoid_: session（代码场景外）
