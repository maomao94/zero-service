# GIS 开放接口

GIS 方法统一通过 `POST /v1/gateway` 调用，沿用开放 API 的 Basic 鉴权、应用签名和响应格式。方法仅在转发服务配置 GIS 下游后注册。字段以 [gis.proto](../../app/gis/gis.proto) 为准，调用规则见 [开放 API 网关](./README.md)。

## 方法清单

下表的 RPC 均属于 `gis.Gis`。

| method | 下游 RPC | 说明 |
| --- | --- | --- |
| `gis.encodeGeoHash` | `EncodeGeoHash` | 按指定精度编码一个经纬度点 |
| `gis.encodeGeoHashMulti` | `EncodeGeoHashMulti` | 将一个点编码为多个精度的 GeoHash |
| `gis.decodeGeoHash` | `DecodeGeoHash` | 查询 GeoHash 格网的中心和边界 |
| `gis.encodeH3` | `EncodeH3` | 按指定分辨率编码一个经纬度点 |
| `gis.encodeH3Multi` | `EncodeH3Multi` | 将一个点编码为多个分辨率的 H3 索引 |
| `gis.decodeH3` | `DecodeH3` | 查询 H3 格网的中心和边界 |
| `gis.gridDisk` | `GridDisk` | 按 H3 索引查询指定圈数的邻格 |
| `gis.gridDiskByPoint` | `GridDiskByPoint` | 按经纬度点查询 H3 邻格 |
| `gis.generateFenceCells` | `GenerateFenceCells` | 计算多边形覆盖的 GeoHash 格网 |
| `gis.generateFenceH3Cells` | `GenerateFenceH3Cells` | 计算多边形覆盖的 H3 格网 |
| `gis.pointsWithinRadius` | `PointsWithinRadius` | 筛选指定半径内的点 |
| `gis.pointInFence` | `PointInFence` | 判断点是否位于指定围栏内 |
| `gis.pointInFences` | `PointInFences` | 判断点命中了哪些指定围栏 |
| `gis.distance` | `Distance` | 计算两点球面距离 |
| `gis.batchDistance` | `BatchDistance` | 批量计算点对距离 |
| `gis.nearbyFences` | `NearbyFences` | 查询点附近已保存的围栏 |
| `gis.transformCoord` | `TransformCoord` | 转换单点坐标系 |
| `gis.batchTransformCoord` | `BatchTransformCoord` | 批量转换坐标系 |
| `gis.routePoints` | `RoutePoints` | 计算从起点访问各途经点的顺序与总距离 |
| `gis.createFence` | `CreateFence` | 创建围栏并保存覆盖格网 |
| `gis.updateFence` | `UpdateFence` | 修改围栏并更新覆盖格网 |
| `gis.deleteFence` | `DeleteFence` | 删除围栏 |
| `gis.listFences` | `ListFences` | 分页查询围栏 |
| `gis.getFence` | `GetFence` | 查询围栏详情 |

## 调用示例

### GeoHash 编码

```json
{
  "method": "gis.encodeGeoHash",
  "data": {"point": {"lon": 120.15, "lat": 30.28}, "precision": 7}
}
```

成功时从 `data.geohash` 读取编码结果。

### 距离计算

```json
{
  "method": "gis.distance",
  "data": {"a": {"lon": 120.15, "lat": 30.28}, "b": {"lon": 120.16, "lat": 30.29}}
}
```

`data.meters` 是球面距离，单位为米。距离为零时 protojson 可能省略该字段。

### 查询已保存的围栏

```json
{
  "method": "gis.getFence",
  "data": {"fenceId": "fence-id"}
}
```

将 `fence-id` 换成已有围栏的 ID，成功时从 `data.fence` 读取详情。

## 字段约定

- 经纬度对象使用 `{lon, lat}`，单位为度。经度范围为 `[-180, 180]`，纬度范围为 `[-90, 90]`。
- 围栏多边形使用 `outer.points` 表示外环，可用 `holes` 指定排除区域。每个环至少包含 3 个点，服务端自动闭合。命中判断可直接传 `polygon`，也可通过 `fenceId` 引用已保存的围栏。
- `gis.createFence`、`gis.updateFence` 和 `gis.deleteFence` 会修改持久化数据。围栏存储及坐标约定见 [GIS 服务说明](../gis/README.md)。
