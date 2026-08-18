# KML/KMZ 指南

## 概述

KML（Keyhole Markup Language）是 OGC 标准的 XML 标记语言，用于描述地理数据。KMZ 是 KML 的 ZIP 压缩格式。在无人机航点任务中，KML 用于定义飞行路径和航点参数。

## 文件结构

```xml
<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
  <Document>
    <Placemark>...</Placemark>
  </Document>
</kml>
```

| 标签 | 用途 |
|------|------|
| `<Document>` | 文档容器 |
| `<Folder>` | 组织要素分组 |
| `<Placemark>` | 地理点（航点） |
| `<Point>` / `<LineString>` / `<Polygon>` | 点 / 线 / 面 |
| `<ExtendedData>` | 自定义扩展数据 |

## 坐标系统

- 格式：`经度,纬度,高度`（十进制度数，高度默认米）
- 高度模式：`absolute`（绝对）、`relativeToGround`（相对地面）、`clampToGround`（贴地）

## 航点任务结构

```xml
<Placemark>
  <name>Waypoint1</name>
  <Point>
    <altitudeMode>absolute</altitudeMode>
    <coordinates>100.0,30.0,100.0</coordinates>
  </Point>
  <ExtendedData>
    <mis:type>Waypoint</mis:type>
    <mis:gimbalPitch>-45</mis:gimbalPitch>
    <mis:heading>0</mis:heading>
    <mis:speed>1</mis:speed>
    <mis:actions>ShootPhoto</mis:actions>
  </ExtendedData>
</Placemark>
```

### DJI 扩展参数

| 标签 | 说明 | 示例 |
|------|------|------|
| `mis:gimbalPitch` | 云台俯仰角 | -45 |
| `mis:heading` | 航向角 | 0 |
| `mis:speed` | 飞行速度 | 1 |
| `mis:turnMode` | 转弯模式 | Counterclockwise |
| `mis:actions` | 航点动作 | ShootPhoto |

## 执行流程

1. 起飞 → 按序飞至各航点 → 执行预设动作 → 完成任务

常见航点类型：
- **拍摄点**：执行拍照/录像
- **过渡点**：调整位置和姿态
- **爬升点**：提升高度

## 工具

- **Google Earth** / **QGIS**：查看和编辑
- **DJI GS Pro**：航点规划

## 注意事项

- 避开禁飞区，遵守飞行规定
- 备份文件，添加清晰命名
- 注意坐标数据敏感性
