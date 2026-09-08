# Carbonx 日期时间处理规范

> 编写或修改涉及日期格式化、解析、当前时间、时间运算的业务代码时读取。

## 核心规则

业务代码日期处理统一走 `common/carbonx`（或其底座 `github.com/dromara/carbon/v2`），不混用 `time` 包的 Format/Parse/Now：

| 场景 | 正确做法 | 错误做法 |
|------|---------|---------|
| 输出 `yyyy-MM-dd HH:mm:ss` | `carbonx.FormatDateTime` / `OrEmpty` / `FormatNullDateTime` | `t.Format("2006-01-02 15:04:05")` |
| 输出可空时间 | `carbonx.FormatDateTimeOrEmpty`（Go 零时间）/ `FormatNullDateTime`（sql.NullTime） | 手写 `IsZero()` 判断后再 Format |
| 解析时间文本 | `carbon.Parse(s)` + `IsValid()` 守卫 | `time.ParseInLocation("2006-01-02 15:04:05", ...)` |
| 取当前时间落库 | `carbonx.NowStartOfSecond().StdTime()` | `time.Now()` |
| 时间运算 | `carbon.Now().AddSeconds(n)` / `AddDay()` 等链式 | `time.Now().Add(...)` 后再 Format |
| 非标准格式输出 | `carbon.Now().Format("Ymd")`（PHP 风格 token） | `time.Format("20060102")` |
| Carbon 转回 `time.Time` | `c.StdTime()` | — |

**Why**：carbonx 的 init 统一了全局默认（上海时区、`DateTimeLayout`、zh-CN），格式化、解析、运算共用同一套语义；混写 Go layout（`2006-01-02`）与 PHP token（`Ymd`）容易出错，`time.Now()` 的亚秒精度写秒级 datetime 列时被静默截断。

依据：`common/carbonx/carbonx.go`、`app/trigger/internal/logic/helper.go`、`app/live/internal/logic/generatemeetingticketlogic.go`、`app/live/internal/logic/joinmeetingbyticketlogic.go`

## Convention: 当前时间落库用秒级精度

**What**：Logic 层填充 `time.Time` 业务字段（start_time、join_time、create_time 等）时用 `carbonx.NowStartOfSecond().StdTime()`。

**Why**：业务 datetime 列为秒级精度，`time.Now()` 的亚秒部分写库被截断，导致落库值与日志/响应文本不一致；秒级取值也使同秒排序行为可预期。Repo/Store 层签名保持 `time.Time`/`sql.NullTime`，carbonx 构造发生在 Logic 层。

## Convention: 格式化与解析互为逆过程

**What**：序列化侧（写 Redis/落库文本/出参）与解析侧必须用同一套 carbonx/carbon API 迁移。`carbon.ToDateTimeString()` 输出 `yyyy-MM-dd HH:mm:ss`，对应解析侧 `carbon.Parse(s)`（默认 layout 即 DateTimeLayout）。

**Why**：只迁移一侧会造成新旧格式不兼容（例：Redis 中票据 expireTime 由旧代码写入、新代码解析）。

## Wrong vs Correct

### Wrong

```go
ticket := fmt.Sprintf("%s-%s", time.Now().Format("20060102"), uuid)
expire := time.Now().Add(time.Duration(sec) * time.Second)
data["expireTime"] = expire.Format("2006-01-02 15:04:05")
expireT, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
if err == nil && time.Now().After(expireT) { ... }
now := time.Now()
```

### Correct

```go
ticket := fmt.Sprintf("%s-%s", carbon.Now().Format("Ymd"), uuid)
expire := carbon.Now().AddSeconds(int(sec))
data["expireTime"] = expire.ToDateTimeString()
if expireAt := carbon.Parse(s); expireAt.IsValid() && expireAt.Lt(carbon.Now()) { ... }
now := carbonx.NowStartOfSecond().StdTime()
```

## 边界与例外

- `carbon.Parse` 对非法输入返回 invalid Carbon 而非 error，必须显式 `IsValid()`/`IsInvalid()` 判断后再比较，等价于原先 `err == nil` 守卫。
- 输出格式化的签名契约（时区保留、零值语义）见 [shared-packages.md](./shared-packages.md) 的 Carbon 格式化场景；不要给 `FormatDateTime*` 隐式加 timezone。
- 严格解析、RRULE、协议 Unix 单位、日期/时间-only 输入仍由原领域包负责。
- 依赖 carbon 默认配置（时区/layout）的包需确保 carbonx 的 init 已执行：直接 import `zero-service/common/carbonx`，或副作用导入 `_ "zero-service/common/carbonx"`。

## 验证

- 迁移后检索残留：`rg 'Format\("2006|ParseInLocation|time\.Now\(\)' app/<svc> --glob '!*_test.go' --glob '!*.pb.go'` 应为空。
- 同一时间值的格式化输出与解析输入做 round-trip 测试；涉及持久化/缓存文本的迁移要覆盖旧格式数据兼容。
