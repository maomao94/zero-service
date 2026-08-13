# 日期时间工具统一设计

## 设计目标

以 `common/carbonx` 作为项目通用 Carbon 入口，提供日常高频的组合操作，统一标准日期时间文本输出并消除 `common/tool/timeutil.go`。公共层只拥有格式化和基础时间转换机制，领域层继续拥有输入校验、时区策略、协议单位、调度与持久化语义。

## 包边界

### `common/carbonx` 负责

- 初始化 Carbon 全局默认布局、上海时区、locale 和周定义。
- 从当前时间或 `time.Time` 构造适合继续链式操作的 `*carbon.Carbon`，包括秒精度归一组合。
- 将 `time.Time` 输出为 Carbon 标准秒、毫秒、微秒日期时间文本。
- 以名称明确表达 Go 零时间或 `sql.NullTime` 输出空字符串的可选值格式化。
- 提供单位明确的当前 Unix 秒、毫秒和微秒值。

### 原所有者继续负责

- `common/rrulex`：RFC 5545 解析、查询、时区描述。
- `common/crontask`：调度时间状态、lease、重试、零值耗尽语义。
- Trigger/ISP：上海墙钟输入、严格校验、日期时间组合和业务范围。
- `common.DateTime` / `common/copierx`：JSON 与 copier wire contract。
- EXIF、Docker、DJI、IEC、CLI、路径命名：各自格式、单位、fallback 和展示语义。

## API 形态

API 使用 `time.Time`、`sql.NullTime`、`*carbon.Carbon` 和基础整数，不导入任何业务 model/proto。

计划提供以下能力，最终命名在实现时按 Go 可读性和相邻 Carbon 命名统一：

```go
func NowStartOfSecond() *carbon.Carbon
func FromTime(value time.Time, timezone ...string) *carbon.Carbon
func FromTimeStartOfSecond(value time.Time, timezone ...string) *carbon.Carbon

func FormatDateTime(value time.Time, timezone ...string) string
func FormatDateTimeMilli(value time.Time, timezone ...string) string
func FormatDateTimeMicro(value time.Time, timezone ...string) string
func FormatDateTimeOrEmpty(value time.Time, timezone ...string) string
func FormatNullDateTime(value sql.NullTime, timezone ...string) string

func NowUnix() int64
func NowUnixMilli() int64
func NowUnixMicro() int64
```

- `FromTime` 与格式函数未传 timezone 时保留输入 `time.Time.Location()`；传入 timezone 时显式转换，与 Carbon API 一致。
- 三种格式 API 与 Carbon 当前便捷输出一一对应：`DateTime`、`DateTimeMilli`、`DateTimeMicro` 分别委托 `ToDateTimeString`、`ToDateTimeMilliString`、`ToDateTimeMicroString`，禁止降精度、互换或改变小数宽度。
- Proto 的业务分类只决定哪些现有调用可以接入对应工具，不用于推导或修正格式；字段名不是精度依据，固定协议和业务专用格式仍由原适配器拥有。
- 普通 `FormatDateTime*` 不暗中赋予业务空值策略；需要零值为空使用 `OrEmpty` / `Null` API。
- `FormatNullDateTime` 以 `Valid` 为唯一空值依据；有效但值为 Go 零时间时仍按有效值处理，避免混淆 SQL NULL 与 Go zero。若实际调用契约要求两者都为空，由调用方先规范化或使用零值 API。
- `NowUnix*` 只提供当前时刻，不替调用方决定协议字段单位。
- 不提供 `ParseDateTime` 通用入口：现有解析场景在严格性、空值、location 和外部格式上差异明显，Carbon 原生解析已经足够简洁。

## 迁移规则

### 阶段一：API 原型与契约测试

- 仅在 `common/carbonx` 新增 API 和单元测试。
- 保留 `common/tool/timeutil.go`、旧测试和全部调用，避免 API 尚未确认时产生大范围返工。
- 测试展示函数签名、链式操作、location、显式 timezone、精度、零值、SQL NULL 和 Unix 单位的可观察行为。
- 阶段一完成后暂停实施，由用户审查命名、粒度和调用体验；不合适时只调整新包 API。

### 阶段二：全仓迁移

- 仅在用户明确确认阶段一 API 后开始。

### 直接迁移

- `tool.NowStartOfSecond`、`tool.CarbonFromTimeStartOfSecond`、`tool.Gen*TS` 全部迁移到 `carbonx` 新 API。
- `carbon.CreateFromStdTime(t).ToDateTimeString()` 等纯格式组合迁移到对应 `FormatDateTime*`。
- `carbon.Now().ToDateTimeString()` 和 `carbon.Now().Format("Y-m-d H:i:s")` 等纯当前标准格式输出迁移到统一 API；若后续还要链式计算则保留 Carbon 对象。
- Trigger `formatTime`、`formatOptionalTime` 等语义相同的局部 helper 迁移到零值/NULL API并删除。
- 重复标准布局常量若迁移后没有严格解析用途则删除；严格解析继续使用 `carbon.DateTimeLayout` 或领域常量。

### 保留原实现

- `carbon.Parse(...).ToDateTimeString()`：解析与错误容忍属于调用边界，链式表达更清楚。
- `carbon.Now().AddHours(...).ToDateTimeString()`、`carbon.NewCarbon(t).Format(custom)`：包含业务日期运算或自定义格式。
- 微秒、毫秒调用只在输出契约完全一致时替换，不改变精度。
- 日期-only、时间-only、RFC3339/RFC5545、紧凑日期、带 offset 输出和 Unix 协议转换不纳入标准秒机械替换。

## 兼容与风险

- 不保留 `common/tool` 旧 API；同一提交完成定义、调用和测试迁移，保证仓库最终可编译。
- 最大风险是相同字符串布局下隐藏的 location、零值和精度差异。每类替换先按调用语义分类，替换后再搜索残留并记录保留原因。
- Carbon v2.6.17 的毫秒/微秒便捷输出底层分别使用 `.999` 和 `.999999`，会裁剪末尾零。新工具直接保持该行为；固定宽度调整属于协议行为变更，不在本任务范围。
- `common/carbonx` 目前常被服务入口副作用导入；改为普通导入调用 API 后仍会执行同一 `init`，不会增加另一套默认配置。
- 不修改 `.proto`、数据库 schema、生成文件和 Carbon 版本，回滚可整体回退本任务代码与任务文档。
- 阶段一保持新 API 无调用方，回滚或改签名成本局限在 `common/carbonx`；阶段二才承担跨仓迁移风险。

## 验证设计

- `common/carbonx` 表驱动测试：输入 location 保留、显式上海转换、秒/毫秒/微秒精度、Go zero、有效/无效 `sql.NullTime`、秒精度截断、Unix 单位范围。
- 精度测试使用包含尾零和非尾零的小数样例，与迁移前 Carbon 便捷输出做等价断言，而非强制固定宽度。
- 编译和测试所有直接迁移调用方，重点覆盖 Trigger/ISP 调度适配器与回调。
- 全仓搜索确认旧 API 和可统一的重复格式组合清零；残留必须属于设计中的保留类别。
- 最后运行 `go test ./...`、`go vet ./...`（环境允许时）、`git diff --check`。
