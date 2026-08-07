# 技术设计

## 边界与所有权

- `common/djisdk` 拥有 DJI HMS 协议字段、字典加载、tip 选择、语言回退和模板渲染。
- `app/djicloud/internal/hooks` 只负责调用解析器并追加写入告警历史，不复制文案匹配规则。
- `app/djicloud/model/gormmodel` 拥有持久化字段；`.proto` 是 RPC 响应契约源。

## SDK 设计

新增 `common/djisdk/hms.go`：

- 使用 `//go:embed hms.json` 提供默认字典，避免部署时依赖工作目录。
- `HmsConfig` 提供 `Language` 和可选 `DictionaryPath`；空路径使用嵌入字典。
- `HmsResolver` 构造时一次性将 JSON 解码为只读 map，构造失败返回错误；服务启动使用 Must 构造方式失败即终止。
- `Resolve(HmsItem)` 返回结构化结果，至少包含最终 key、language、template 和 message。

`HmsItem.Args` 使用 `map[string]any` 完整保留开放参数；SDK 提供整数和字符串读取 helper。官方字段优先使用 `alarmid`，不把十六进制文本强制重编码为整数。

新增严格设备型号值对象和只读注册表：domain 固定为 `0=飞机`、`1=负载`、`2=遥控器`、`3=机场`；三元组查询当前产品支持表中的产品名称；`gimbalindex` 单独表示负载挂载位置。相同负载三元组对应多个宿主上下文时返回共用型号名，不伪造宿主机型或机场代际。

## 匹配流程

1. 严格解析 `device_type` 三段数字；domain 0 映射 `fpv`，domain 3 映射 `dock`，domain 1/2 无 HMS 前缀。
2. `in_the_sky=1` 时先查 `{prefix}_tip_{code}_in_the_sky`，再查 `{prefix}_tip_{code}`；地面告警只查普通键。
3. 不跨 `dock`/`fpv` 类别回退，不选择固件专用后缀。
4. 按配置语言精确读取文案；对应值为空或缺失时返回未知告警，不切换语言。
5. 无任何模板时返回包含 code 的未知告警文案，保证入库 message 非空。

## 模板渲染

- `%component_index`、`%index` 使用协议索引加一。
- `%battery_index`、`%dock_cover_index` 使用 0 左、其他值右的本地化文本。
- `%charging_rod_index` 使用 0/1/2/3 的前/后/左/右本地化文本。
- `%alarmid` 使用 args 中官方 `alarmid` 的十六进制文本。
- `%gimbal_index`、`%lidar_index`、`%lte_index` 使用同名参数。
- `%s`、`%1$s`、`%d` 等位置参数仅按已确认的 HMS 索引约定替换；无法确定或参数缺失时保留占位符并记录日志。

渲染不得修改共享字典，解析器可被并发调用且不需要写锁。

## 服务数据流

1. `ServiceContext` 从 `c.Dji.Hms` 构造一个 `HmsResolver`。
2. resolver 经 `RegisterDjiClientOptions` 注入 `NewHmsEventNotifyHandler`。
3. hook 对每个 `HmsItem` 调用产品注册表和 resolver，将 device_type_name、三元组、message、已知 args 与 `item_json` 一并 `Create`。
4. `DjiHmsAlert` 持有 message、device_type_name 和平铺字段。
5. `HmsAlertInfo` 只追加新字段号，Logic 原样返回数据库值。

## 兼容性与迁移

- Proto 仅追加字段，不复用或重排已有字段号。
- 开发/测试模式的 AutoMigrate 自动增加列；生产数据库迁移由现有部署流程负责。
- 历史数据的 message 为空，查询接口按数据库值返回，不在查询时动态补算。
- `item_json`、确认流程、排序与过滤保持不变。

## 回滚

- 应用回滚可停止写入和返回 message；新增数据库列可保留，不影响旧版本。
- 若字典加载失败，服务启动失败而不是运行时静默产生错误文案。
