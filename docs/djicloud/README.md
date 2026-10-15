# DJI 云平台

DJI Dock 3 云平台接入服务，封装云平台 API，支持航线任务、直播推流和 DRC 指令飞行。

> **定位说明**：本服务实现的是**第一代上云 API（Cloud API）**，协议收发（topic / method / payload）由 `common/djisdk` 手写封装，适合协议学习、联调参考与存量系统维护。**新项目建议评估[上云 API 2](https://developer.dji.com/cn/cloud-api-2)**（官方仓库 [dji-sdk/Cloud-API-2](https://github.com/dji-sdk/Cloud-API-2)）。完整说明见[《DJI 云平台服务》](./djicloud.md) 开篇。

## 文档

| 文档 | 内容 |
| --- | --- |
| [DJI 云平台](./djicloud.md) | DJI Dock 3 Cloud API、RPC 接口和配置说明 |
| [KML/KMZ 指南](./kml-kmz-guide.md) | 无人机航点任务 KML/KMZ 文件结构 |
