# 日期时间工具统一实施计划

## 实施顺序

- [x] 读取公共包、编码、质量、Trigger、crontask 规范及日期时间研究报告，复核当前工作树，避免修改 `.opencode/package.json`。

### 阶段一：编写工具

- [x] 扩展 `common/carbonx`：加入 Carbon 构造/秒精度、标准秒/毫秒/微秒格式、零值/SQL NULL 和 `NowUnix*` API。
- [x] 为 `common/carbonx` 添加表驱动测试，先锁定 location、timezone、precision、zero/null 和 Unix 单位契约。
- [x] 建立秒/Carbon 毫秒/Carbon 微秒调用清单，使用含尾零样例验证与迁移前输出完全一致，再逐类迁移，禁止跨精度替换。
- [x] 仅运行 `common/carbonx` 定向测试、vet、diff 检查，确认未修改任何现有调用方或 `common/tool/timeutil.go`。
- [x] 暂停并向用户提交阶段一 API、测试结果和 diff；用户已确认 API，本次完成审计会话再次确认 Phase 1 与 Phase 2 已完成。

### 阶段二：确认后迁移

- [x] 仅在用户明确确认阶段一 API 后继续。
- [x] 将 `common/tool/timeutil.go` 全部调用迁移到 `common/carbonx`，移动对应测试后删除旧源码和旧测试。
- [x] 分类型替换全仓语义等价的 `CreateFromStdTime(...).ToDateTime*String()`、当前标准格式输出及局部 formatter；每批替换后执行目标包编译/测试。
- [x] 检查标准布局常量和直接 Carbon 格式化残留，删除无用重复项，并为每类保留调用核对领域/协议原因。
- [x] 更新因旧 API 示例而过时的 Trellis Trigger 规范代码片段；不改变其中业务契约。
- [x] 运行完整质量检查和 diff 审查，确认无精度、时区、空值、协议或无关文件变化。

## 验证命令

```bash
gofmt -w <changed-go-files>
go test ./common/carbonx
go test ./common/crontask ./app/trigger/internal/cronjob ./app/trigger/internal/logic ./app/ispagent/internal/crontask ./app/ispagent/internal/handler ./app/ispagent/internal/svc
go test ./<other-direct-caller-packages>
go test ./...
go vet ./...
git diff --check
git status --short
```

## Review Gates

- 阶段一 gate：只包含新工具与测试；用户确认前不迁移调用、不删除旧工具。
- 公共 API gate：没有万能 option/builder，没有业务依赖，没有逐方法复制 Carbon API。
- 语义 gate：每个替换点的 location、timezone、zero/null、precision 和错误行为与迁移前一致。
- 调度 gate：RRULE、`ScheduledTime`、`NextRun`、严格上海输入和秒精度归一契约不变。
- 协议 gate：proto/string、JSON、EXIF、Docker、DJI、IEC 和 Unix 单位不变。
- 精度 gate：秒/毫秒/微秒调用保持迁移前 Carbon 输出；Proto 与生成文件无 diff；LAL/BridgeDump 等接入协议格式不被公共工具覆盖。
- 清理 gate：旧 `common/tool/timeutil.go`、旧 API 引用和无用局部 helper 均不存在。

## Rollback Points

- 公共 API 与测试完成后先验证 `common/carbonx`，再开始调用迁移。
- 每类调用迁移独立检查，若发现语义不一致，仅撤销该类迁移，不扩大 helper 兼容行为。
- 不通过旧包 wrapper 临时维持编译；最终变更必须原子完成定义与调用迁移。
