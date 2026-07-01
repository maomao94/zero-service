# 自定义飞行区 GeoJSON 处理 + djcloud OSS 配置与平台接口

## Goal

提供平台级 API 供用户设置自定义飞行区（DFence / NFZ），接收结构化 orb 参数，通过 djisdk 工具生成 DJI 格式 GeoJSON，上传至 OSS 后通过 DJI MQTT 协议下发到机巢设备。

## 已确认事实（来自代码库）

1. **djisdk 已有飞行区协议基础**：`FlightAreasUpdate`/`FlightAreasGet`/`FlightAreasSyncProgress`/`FlightAreasDroneLocation` 方法常量及数据模型已定义，但 `FlightAreasGet` handler 目前返回空文件列表。
2. **orb 包已可用**：`github.com/paulmach/orb v0.13.0` 在 go.mod 中，且 `common/gisx/` 已广泛使用 `orb.Polygon`、`orb.Point` 做空间运算。
3. **ossx 已就绪**：Minio 后端支持上传、签名 URL、删除等操作，满足飞行区文件托管需求。
4. **djicloud.proto 有"平台能力接口"分区**：可用于新增设置自定义飞区的 RPC。
5. **djicloud Config 无 OSS 配置**：当前只有 `Dji`、`DB`、`Telemetry`、`DangerousOps`、`SocketPushConf`。

## 决策记录

| 决策 | 结论 |
|------|------|
| API 输入方式 | 传结构化 orb 参数（多边形坐标、NFZ 圆心+半径等），由服务端调用 djisdk 工具生成 GeoJSON |
| OSS 配置 | 仅 Minio，配置写入 djcloud 的 etc/*.yaml，在 Config 中增加 OssProperties 可选字段 |
| 模型粒度 | 飞行区配置主表（存 OSS 文件信息）+ 设备下发状态表（记录哪些设备收到、同步状态） |

## GeoJSON 格式（DJI Dock3 规范）

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "id": "xxxx_xxxx_xxxx_xxxx",
      "type": "Feature",
      "geofence_type": "dfence",
      "geometry": {
        "type": "Polygon",
        "coordinates": [[[lon, lat], ...]]
      },
      "properties": { "radius": 0, "enable": true }
    },
    {
      "id": "xxxx_xxxx_xxxx_xxxx",
      "type": "Feature",
      "geofence_type": "nfz",
      "geometry": {
        "type": "Point",
        "coordinates": [lon, lat]
      },
      "properties": { "subType": "Circle", "radius": 1000, "enable": true }
    }
  ]
}
```

## Requirements

1. **djisdk GeoJSON 工具文件**：在 `common/djisdk/` 下新增文件，提供 orb → GeoJSON 和 GeoJSON → orb 的转换能力。
2. **djcloud OSS 配置**：在 `app/djicloud/internal/config/config.go` 中增加 OSS 配置（可选），允许 djcloud 服务上传飞行区 GeoJSON 文件至 OSS。
3. **平台级 API**：在 `djicloud.proto` 新增 `SetCustomFlyRegion` RPC，接收结构化参数（dfence 多边形坐标 / nfz 圆心+半径），生成 GeoJSON 上传 OSS，触发设备更新。
4. **通知模型**：新增飞行区配置主表 + 设备下发状态表，存储 OSS 生成的相关信息。

## Acceptance Criteria

- [ ] `common/djisdk/` 新增 GeoJSON 工具文件，支持 `orb.Polygon`/`orb.Point` ↔ DJI GeoJSON FeatureCollection 互转
- [ ] `app/djicloud/internal/config/config.go` 增加 `OssConfig` 可选配置块
- [ ] `djicloud.proto` 新增 `SetCustomFlyRegion` RPC 及请求/响应消息（结构化参数输入）
- [ ] 新增 DB model：`DjiFlyRegion`(配置主表) + `DjiFlyRegionSyncStatus`(设备同步状态表)
- [ ] `FlightAreasGet` handler 能从 DB 返回实际文件列表
- [ ] 代码通过 lint 检查

## Out of Scope

- 飞行区同步进度/告警的高级处理逻辑（已有 event handler 框架，本次不深入实现）
- 多租户飞行区隔离（如需要后续迭代）
