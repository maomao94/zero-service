# 日志脱敏设计（L1-L3）

## 边界

本任务只修复三处完整 token 日志泄漏。**不改变**：
- 发送端 raw token 传播（保持现状，按需传播）；
- MCP `_meta` 内容（保持现状）；
- 接收端提取行为（first-value、空首跳过）；
- wire key/顺序、`b64:`、claim 转换、typed-key 结构。

> 设计修订（用户确认 2026-08-14）：接收端 report-only 观测为过度设计，已移除。本任务仅保留日志脱敏。

## 1. 日志脱敏 L1-L3

### L1 `facade/streamevent/internal/logic/upsocketmessagelogic.go:30-31`

现状：`Infof("token: %s", token)` 打印完整 token。

改：只记录存在性与身份（claims 不含 token）：

```go
tokenPresent := authctx.GetAuthorization(l.ctx) != ""
l.Logger.Infof("auth_present=%t user_id=%s", tokenPresent, authctx.GetUserId(l.ctx))
```

### L2 `aiapp/mcpserver/internal/tools/echo.go:25-28`

现状：`Debugf("token: %s,username: %s", auth, username)`。

改：去掉 token 值，保留 username（信息属性）：

```go
authPresent := auth != ""
logx.Debugf("auth_present=%t, username: %s", authPresent, username)
```

### L3 `common/mcpx/auth.go:61,65`

现状：line 61 把 raw token 存入 `extra` map；line 65 `%v` 打印整个 map（含 token）。

- **line 65 日志**：改为不打印 `extra` 全量，只打印键名与 userId：
  ```go
  logx.WithContext(ctx).Debugf("[mcpx-auth] jwt verified, userId=%s, extraKeys=%v", info.UserID, mapKeys(extra))
  ```
- **line 61 `extra[authctx.CtxAuthorizationKey] = token`**：**保留**（这是 P5/P6 边界契约，`auth_test.go:46-48` 锁定 `Extra[CtxAuthorizationKey] == token`；移除属 meta 策略，非日志修复）。仅日志不再打印它。

### 测试影响

- L1/L2：无现有测试锁定日志文本；`facade/streamevent` 无测试文件。
- L3：`auth_test.go:46-48` 锁定 Extra 内容 → **不改**；日志格式无测试锁定，改安全。

## 2. 禁止混合（audit §8.3）

- 不升级 `b64:`。
- 不捆绑 claim 规范化或 typed-key 语义变更。
- 不在无逐边界证据下全局删除 Authorization。
- 不改发送端传播、不改 MCP `_meta` 内容、不改提取语义。
