# 公共包设计规范

## 适用范围

新增 `common/` 包、扩展公共 API、设计 client option、字节/寄存器转换或通用工作流时读取。

## 进入 `common/` 的条件

- 至少有明确的跨服务复用需求，且能力可以脱离具体业务 proto/model 独立描述。
- 包拥有清晰单一职责、稳定输入输出、错误语义和测试；不能只因代码重复几行就抽象。
- 业务策略留在调用服务，公共包提供机制。例如 `common/mqttx` 管理连接与关联响应，具体 topic/payload 由协议或领域包拥有。
- 优先扩展现有 `common/` 包；`common/tool` 是历史混合工具包，不继续堆放领域能力。

依据：`common/netx`、`common/mqttx`、`common/bytex`、`common/gisx` 及调用点。

## API 与构造

- 必需依赖使用构造参数，行为变体用小接口/函数，非必需配置用 function option。
- option 写入 `XxxOptions` 或私有配置结构，由构造函数校验、规范化并生成运行态对象；option 不直接修改连接、锁、缓存或状态机。
- 零值是否可用必须明确；无法提供安全零值时由构造函数返回 error，而不是延迟到首个调用 panic。
- 公开接口只暴露调用方需要的能力。适配具体框架时在服务或 transport 边界包中完成。
- 长期资源必须暴露幂等 `Close`/`Stop` 或由 context 明确管理，文档说明谁负责调用。

依据：`common/netx/client.go`、`common/djisdk/option.go`、`common/antsx/replypool.go`、`common/wsx/config.go`。

## 二进制与转换

- 多字节编码必须显式指定 byte order、宽度、符号和越界行为；复用 `common/bytex` 已有函数。
- 解码前检查长度，错误输入返回 error；不能依靠 slice 越界 panic 作为校验。
- 整数窄化、浮点位模式、寄存器顺序等有损/协议相关转换必须由名称和测试表达语义。
- 泛型转换只消除同构重复；调用方仍负责确认窄化或符号转换是协议允许的。

依据：`common/bytex/` 及其测试、`common/isp/serializer.go`。

## Scenario: Carbon 日期时间格式化

### 1. Scope / Trigger

- 将 `time.Time`、Go 零时间或 `sql.NullTime` 输出为项目现有的秒、毫秒、微秒 Carbon 文本，或迁移直接 Carbon 格式化调用时适用。

### 2. Signatures

```go
func FormatDateTime(value time.Time, timezone ...string) string
func FormatDateTimeMilli(value time.Time, timezone ...string) string
func FormatDateTimeMicro(value time.Time, timezone ...string) string
func FormatDateTimeOrEmpty(value time.Time, timezone ...string) string
func FormatDateTimeMicroOrEmpty(value time.Time, timezone ...string) string
func FormatNullDateTime(value sql.NullTime, timezone ...string) string
func ToNullTime(value time.Time) sql.NullTime
```

### 3. Contracts

- 未传 `timezone` 时保留输入 `time.Time.Location()`；不能因 Carbon 全局默认上海时区而隐式换区。传入 timezone 时转换同一 instant。
- 秒、毫秒、微秒分别委托 Carbon 的 `ToDateTimeString`、`ToDateTimeMilliString`、`ToDateTimeMicroString`，保留毫秒/微秒尾零裁剪行为。
- 普通 `FormatDateTime*` 格式化 Go 零时间；只有 `OrEmpty` 明确将 Go 零时间输出为空字符串。
- Carbon 的 `IsZero()` 只提供判断：零 Carbon 调用 `ToDateTimeString` / `ToDateTimeMicroString` 仍输出 `0001-01-01 00:00:00`，不能用 Carbon 原生格式化替代 `OrEmpty` 契约。
- 零值为空属于缺失值语义，使用命名明确的 `OrEmpty` API；不要给格式函数增加不自解释的 `emptyOnZero bool` 或为此引入 function options。
- `FormatNullDateTime` 仅以 `sql.NullTime.Valid` 判断为空；`Valid=true` 的 Go 零时间仍格式化。`ToNullTime` 则将 Go 零时间标记为 invalid。
- 严格解析、RRULE、EXIF、Docker、协议 Unix 单位、日期/时间-only、紧凑路径日期及业务时间运算仍由原领域包负责。

### 4. Validation & Error Matrix

- 非法 timezone -> Carbon 错误语义，格式函数返回空字符串。
- Go 零时间 + `OrEmpty` -> 空字符串。
- `sql.NullTime.Valid=false` -> 空字符串。
- `sql.NullTime.Valid=true` + Go 零时间 -> `0001-01-01 00:00:00`（按指定/输入时区的 Carbon 行为）。

### 5. Good/Base/Bad Cases

- Good: `carbonx.FormatDateTime(value)` 保留 `value.Location()`，用于替换 `carbon.CreateFromStdTime(value).ToDateTimeString()`。
- Base: `carbonx.FormatDateTimeOrEmpty(value)` 明确表达回调字段的零时间为空契约。
- Bad: 把微秒字段改用 `FormatDateTime`，或为“统一上海时区”给原先保留 location 的调用新增 timezone。

### 6. Tests Required

- 使用非上海 location 断言默认格式化不换区，显式 timezone 转换 instant 不变。
- 秒、毫秒、微秒分别与 Carbon 原生输出对比，并覆盖小数尾零。
- 覆盖 Go 零时间、有效/无效 `sql.NullTime`、有效 SQL 零时间和非法 timezone。
- 跨服务迁移后搜索旧 helper 和可直接等价的 Carbon 组合，并运行所有直接调用方测试。

### 7. Wrong vs Correct

#### Wrong

```go
text := carbonx.FormatDateTime(value, carbon.Shanghai) // 原调用保留 value.Location()
optional := carbonx.FormatDateTime(value)             // 原契约要求 Go 零时间为空
optional = carbonx.FormatDateTime(value, true)         // 布尔参数无法在调用点表达 true 的含义
```

#### Correct

```go
text := carbonx.FormatDateTime(value)
optional := carbonx.FormatDateTimeOrEmpty(value)
```

## 工作流封装

- 拦截器顺序属于行为：按声明顺序进入、逆序退出；新增埋点或重试前先确认是否改变 attempt/step 语义。
- context 注入的字段用 typed key 或包内 helper，不能由调用方手工复制内部 key。
- wrapper 要保留原 error、取消和 panic 语义；若做 recover，必须转换成可定位错误而非吞掉。

依据：`common/flowx/`、`common/antsx/invoke.go`。

## 反模式

- 公共包导入具体服务的 `internal/` 或生成 pb。
- 为单个服务的临时流程创建通用框架。
- 暴露内部 map/mutex/连接，使调用方可以绕过不变量。
- 对二进制输入不做长度校验，或在多个包重复 endian 转换。

## 验证

- 公共 API 变更运行目标包及所有直接调用方测试。
- option 覆盖默认值、自定义值、nil/非法输入；转换覆盖边界长度、端序和溢出语义。
- 有共享状态或 goroutine 时增加 `go test -race`。
