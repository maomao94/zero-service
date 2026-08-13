# 阶段二日期时间迁移审计

## 已迁移类别

- `common/tool/timeutil.go` 的秒精度 Carbon 构造与当前 Unix 秒/毫秒/微秒 API 已迁移至 `common/carbonx`；旧文件在本阶段开始前已不在 `HEAD` 中，仓库也不再存在旧 API 引用。
- 纯 `time.Time -> Carbon -> 标准秒/微秒文本` 组合已改用 `carbonx.FormatDateTime*`；Go 零时间使用 `OrEmpty` 语义，现有 SQL NULL 调用点继续通过 `Valid` 分支保持空值行为，公共 API 另以 `FormatNullDateTime` 明确该能力。
- 纯 `carbon.Now()` 标准秒/微秒文本已改用 `carbonx.NowDateTime*`。
- Trigger/ISP 的重复 `toNullTime` 以及 Trigger 的零值 formatter 已由 `carbonx` API 取代。

## 保留的直接 Carbon / Go 格式化

- **解析后格式化**：Podengine 的 Docker RFC3339/RFC3339Nano 输入继续由 `carbon.Parse(...).ToDateTimeString()` 处理，保留解析容错和时区行为。
- **业务链式时间运算**：Trigger execdelay、ISP 规则测试及 StreamEvent 延时示例继续在同一 Carbon 值上执行 `Add*`/`StartOf*`/`EndOf*` 后格式化，避免拆分链式语义或重新取当前时间。
- **同一当前值多用途**：BridgeDump 复用同一个 `now` 同时生成文件名和消息微秒时间，未改成会再次取时钟的 `NowDateTimeMicro`。
- **协议单位转换**：DJI 毫秒 epoch 诊断、Podengine Unix 秒输入继续由 `CreateFromTimestamp*` 格式化；单位选择属于协议边界。
- **领域格式/严格解析**：Trigger 的 `dateTimeLayout`、EXIF 墙钟格式、Docker 工具/CLI 展示、RRULE 带 offset 描述和测试诊断格式继续归原包所有。
- **自定义展示与命名**：紧凑路径日期、CLI 日志时间、Docker UI、`Ymd_His` 文件名等非标准 API 日期时间格式不迁移。
- **注释代码**：IEC client handler 中旧的 `carbon.Now().ToDateTimeString()` 仅存在于注释，不是运行调用，未为本任务改写历史注释。

## 兼容性结论

- 未改变 `.proto`、生成文件、数据库 schema、Carbon 版本、Unix 单位、epoch sentinel、解析严格性或业务日期运算。
- 秒、毫秒、微秒格式仍直接委托 Carbon 对应便捷输出，保留小数尾零裁剪行为。
- 默认格式化保留输入 `time.Time.Location()`；仅现有调用显式要求时区时才通过 `carbonx` timezone 参数转换。

## 完成审计（2026-08-13）

- 用户在本次完成/归档请求中明确确认 Phase 1 API 和 Phase 2 迁移已经实施，要求只审计并记录完成；未重做或回退既有迁移。
- `common/tool/timeutil.go`、`common/tool/timeutil_test.go` 均不存在；仓库搜索未发现 `tool.NowStartOfSecond`、`tool.CarbonFromTimeStartOfSecond`、`tool.GenSecondTS`、`tool.GenMilliTS`、`tool.GenMicroTS` 或对应旧函数定义。
- 仓库搜索未发现 Trigger 的 `formatTime` / `formatOptionalTime` 或 Trigger、ISP 的 `toNullTime` 局部 helper。直接 Carbon/Go 标准布局残留均属于上文列出的解析、链式运算、协议单位、RRULE、EXIF、Docker、CLI 展示或测试诊断场景。
- Phase 2 checkpoint `8d575317` 对 `.proto` / `*.pb.go` 的文件列表为空；本次工作树也没有协议或生成文件 diff。
- `go test ./common/carbonx ./common/crontask ./app/trigger/internal/cronjob ./app/trigger/internal/logic ./app/ispagent/internal/crontask ./app/ispagent/internal/handler ./app/ispagent/internal/svc` 通过。
- `go test ./common/... ./app/trigger/... ./app/ispagent/...` 和 `go test ./...` 通过。
- `go vet ./common/carbonx ./common/crontask ./app/trigger/internal/cronjob ./app/trigger/internal/logic ./app/ispagent/internal/crontask ./app/ispagent/internal/handler ./app/ispagent/internal/svc`、`git diff --check` 通过。
