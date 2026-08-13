# 统一日期时间工具与全仓迁移

## Goal

建立职责清晰、零值与时区语义明确的基础日期时间格式化能力，并在全仓按语义迁移可安全复用的调用，减少重复布局、重复 Carbon 转换和局部 helper，同时保持既有协议、调度、持久化与展示行为兼容。

## Background

- `common/carbonx/carbonx.go:7-15` 当前只通过 `init` 配置 Carbon 默认布局、上海时区、中文 locale 和周定义，没有公开工具 API。
- `common/tool/timeutil.go:9-31` 已提供秒精度 Carbon 构造及当前 Unix 秒/毫秒/微秒函数，但 `common/tool` 是历史混合工具包，规范要求不继续堆放公共能力。
- 全仓同时存在标准秒、微秒、毫秒、日期、时间、RFC3339/RFC5545、紧凑路径日期及多种 Unix 单位；相同文本布局也可能具有不同的时区和空值契约。
- Trigger 有零值转空字符串、`sql.NullTime` 转空字符串和严格上海时间解析等局部逻辑；Trigger 与 ISP 还存在相同的零值 `time.Time` 转 `sql.NullTime` 代码。
- Carbon 从已有 `time.Time` 构造时保留原 `Location`；全局默认上海时区不会自动转换该值。Carbon 也不会把 Go 零时间自动视为空字符串。

## Requirements

- R1. 在 `common/carbonx` 提供基础日期时间格式化 API，内部使用 Carbon，并通过函数名、参数和测试明确布局、精度、时区保留/转换及零值行为。
- R2. 优先覆盖全仓高频的标准秒格式 `yyyy-MM-dd HH:mm:ss`；需要空值语义时提供类型明确的 `time.Time` / `sql.NullTime` 能力，不让调用方重复判断。
- R3. 全仓搜索并迁移与新 API 语义完全一致的直接 Carbon 格式化、Go layout 格式化和局部 helper；删除迁移后无用的重复常量与函数。
- R4. 每个替换点必须保留原值的 location、秒/微秒精度、零值/NULL 输出及错误行为；不能借统一工具隐式改变时区。
- R5. RRULE、Cron 调度状态机、Trigger 严格输入校验、ISP 时间段组合、EXIF、Docker、DJI、JSON wire type、紧凑路径日期和 CLI 展示等领域/协议逻辑保留在原所有者中；仅可复用其中与新基础 API 完全等价的最终格式化步骤。
- R6. 不机械替换 `time.Now`、`time.Parse`、Unix 单位转换、`Add(24*time.Hour)`、epoch sentinel、protobuf 字段类型或生成文件。
- R7. 将 `common/tool/timeutil.go` 中已有时间能力迁移到 `common/carbonx`，同步全部调用和测试，不保留旧包装。
- R8. 记录未迁移调用及原因，使“全项目替换”可审计，而不是以文本匹配数量代替语义一致性。
- R9. 删除 `common/tool/timeutil.go`，将其中全部能力迁移到 `common/carbonx` 并整体替换调用；不提供旧包兼容包装。
- R10. `common/carbonx` 面向日常高频使用提供小而明确的组合 API：Carbon 构造与秒精度归一、标准秒/毫秒/微秒格式化、零值/SQL NULL 格式化和单位明确的当前 Unix 时间戳。已经清晰的 Carbon 解析、日期加减、起止边界和链式操作继续直接使用 Carbon，不逐方法包装。
- R11. 日期时间文本必须按现有业务区分三套精度，不能相互替换：秒级 `yyyy-MM-dd HH:mm:ss`；3 位毫秒文本在 Proto 中属于 LAL 上游会话时间，项目内另有 DJI 诊断日志使用；6 位微秒级用于 StreamEvent、DJI `reported_at`、IEC、File、`common.DateTime` 和 copier 等既有契约。
- R12. Proto 日期时间的业务分类仅用于识别现状：普通日期时间通常为秒级，上报类字段可能要求微秒，固定接入协议及其他字段遵循各自协议、数据源和系统属性。本任务不新增、修正或重新解释这套协议规则。
- R13. 新工具必须保持被替换调用的当前实际输出，包括 Carbon 毫秒/微秒格式的小数位行为；不以 Proto 注释或字段名为由调整宽度、精度、时区、空值或格式。
- R14. 实施分成两个阶段并设置人工确认门禁：第一阶段只新增 `common/carbonx` API 及测试，不修改调用方、不删除 `common/tool/timeutil.go`；用户确认 API 合适后，才执行全仓迁移和旧工具删除。若 API 不合适，先在 `common/carbonx` 内调整并重新确认。

## Acceptance Criteria

- [x] `common/carbonx` 具有覆盖标准秒格式、Go 零时间和 `sql.NullTime` 的表驱动测试，包含非上海 location，证明默认格式化不会意外换区。
- [x] 第一阶段交付时，diff 仅包含 `common/carbonx` 实现、测试及任务文档；项目现有调用和 `common/tool/timeutil.go` 保持不变，供用户独立审查 API。
- [x] 用户明确确认第一阶段 API 后才开始第二阶段替换；本次完成审计会话中用户再次确认 API 与 Phase 2 迁移已经完成，不需重做。
- [x] `common/tool/timeutil.go` 及其测试被删除，仓库不存在对其中旧函数的引用，所有调用迁移至 `common/carbonx`。
- [x] 所有语义等价的标准秒格式化调用使用基础 API；重复的 `"2006-01-02 15:04:05"` / `"Y-m-d H:i:s"` 仅在协议解析、展示或工具程序等有明确所有权的场景保留。
- [x] Trigger 的 `formatTime`、`formatOptionalTime` 等等价局部 helper 被替换或有明确保留理由，空值仍输出空字符串。
- [x] 秒、毫秒、微秒、RFC3339/RFC5545、日期/时间-only、Unix 单位及 epoch sentinel 契约无变化。
- [x] 秒、Carbon 毫秒和 Carbon 微秒格式分别有等价输出测试；迁移清单中的每个调用点保持原精度类别及当前小数位输出，不允许以默认秒级格式覆盖毫秒或微秒字段。
- [x] `.proto` 文件和生成文件无 diff；所有 Proto 字段的实际日期时间输出与迁移前一致，固定协议和业务专用格式保持原样。
- [x] `common/carbonx`、迁移涉及的公共包和直接服务调用方测试通过；全仓 `go test ./...` 通过。
- [x] `gofmt`、相关包 `go vet`、`git diff --check` 通过，最终 diff 不包含既有无关修改 `.opencode/package.json`。

## Out Of Scope

- 不改变 API/proto 字段格式、数据库 schema 或软删除 sentinel。
- 不借工具迁移修正 Proto 注释与当前实现之间的潜在格式差异。
- 第一阶段不修改任何现有日期时间调用方。
- 不把所有日期、时间、Unix 时间戳强制收敛成一个万能函数。
- 不修改 Carbon 第三方库或依赖版本。
- 不因本次迁移改变服务默认时区、系统 `time.Local` 或调度规则。
- 不为 Carbon 已有的每个解析、比较、日期加减或链式方法建立一一对应包装。

## Technical Evidence

- 全仓分类与调用证据：`research/datetime-usage-inventory.md`。
- 工具归属及不安全替换边界：`research/datetime-ownership-and-replacement-boundaries.md`。
- Proto 日期时间格式、字段数量及契约差异：`research/proto-datetime-format-audit.md`。
- 公共包准入依据：`.trellis/spec/backend/common-package-design.md:7-13`。
