# GIS 与围栏规范

## 适用范围

修改 `common/gisx`、`app/gis`、坐标转换、多边形/圆、GEOS、H3 索引、围栏事务或 DJI GeoJSON 时读取。

## 坐标契约

- 项目领域坐标统一为 `longitude, latitude`，即几何中的 `X,Y`。函数名、参数和测试必须保持该顺序。
- H3 API 使用 `latitude, longitude`；只在 H3 适配边界显式交换，禁止把 H3 顺序扩散到领域模型。
- 坐标系转换与经纬度顺序是两个概念；WGS84/GCJ02 等转换不能顺便交换轴。
- 外部 GeoJSON、WKT、DJI payload 进入时在一个适配层完成验证和转换，不在多个 Logic 重复。

依据：`common/gisx/doc.go`、`common/gisx` 转换实现与测试、`app/gis/internal/logic/helper.go`。

## 几何边界

- Polygon ring 必须闭合，但 helper 不修改调用方输入 slice；洞环也要经过长度、闭合和拓扑验证。
- GEOS 核心保持与 `orb` 等上层类型解耦，转换集中在 `common/gisx/geos/orbconv` 适配目录。
- 进入 GEOS 前校验空几何、坐标范围、ring 和类型；C/native panic 在边界 recover 为 error，不能让请求进程退出。
- 精确包含/相交/距离等判断由现有 GEOS/几何库完成，不手写射线算法替换成熟实现。
- 依赖版本以 `go.mod` 为准，Spec 不复制易漂移的版本号。

依据：`common/gisx/geos`、`common/gisx/geos/orbconv` 及测试。

## FenceStore 与事务

- `common/gisx` 通过 `FenceStore` 等接口表达领域数据需求，不依赖 GORM 或具体服务 model。
- `app/gis` 的 GORM store 拥有事务、模型转换和 H3 索引写入；新增存储逻辑放在 store，不泄漏到公共几何包。
- 围栏主体与 H3 cell 索引必须在同一事务保持一致；失败回滚，删除/更新同步清理旧索引。
- H3 只用于粗召回候选，最终命中由多边形/圆的精确谓词判断；不能只因 cell 相同就判定进入围栏。

依据：`common/gisx/store.go`、`app/gis/model/fencestore.go` 及测试。

## 反模式

- 同一个 API 中有时传 `lat,lng`、有时传 `lng,lat` 且命名不区分。
- 为闭环直接 append 到调用方 slice，造成重复调用数据增长。
- 公共 GIS 包导入服务 GORM model。
- 围栏与 H3 索引分两次独立提交。
- H3 粗筛结果直接作为最终空间判断。

## 验证

- 覆盖经纬度顺序、坐标边界、空/非法 ring、洞、多边形边界点、重复调用不修改输入。
- Store 测试覆盖创建/更新/删除事务回滚、旧 cell 清理、粗筛后精判。
- GEOS/CGO 环境不可用时明确说明；至少运行纯 Go 转换和 store 单元测试。
