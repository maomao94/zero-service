# Authorization 传播审计实施计划

## 执行清单

- [x] 用户审查并明确批准本规划后，才将子任务切换为 `in_progress`。（任务状态已为 in_progress，规划门禁已通过）
- [x] 复核传播矩阵中的 HTTP、Socket.IO、gRPC、MCP 源、验证点、接收端和消费者证据。（audit-report.md §2：所有抽查主张可代码证实；R1/R2 两处细化）
- [x] 按默认拒绝基线分类每条 raw token 链路；没有明确委托证据的链路不得标记为 `user-token`。（audit-report.md §3：P1–P8 均无 `user-token` 归类）
- [x] 标记需要业务/deployment owner 回答的 unresolved 项，不用推测替代证据。（audit-report.md §9：U1–U11）
- [x] 复核 `user-id` 等 claims 是否参与授权、数据隔离、审计或仅展示。（audit-report.md §6：user-id 安全属性，user-name 信息属性，dept-code/auth-type 未证实/无消费者）
- [x] 核对 gRPC duplicate、empty-first、context overwrite、身份冲突和 `b64:` 当前契约。（audit-report.md §4；metadata_test.go 契约测试复核）
- [x] 核对 MCP service token 与 user token 双层传播、raw `_meta` 生命周期和嵌套 gRPC 路径。（audit-report.md §5；client.go/wrapper.go/context_meta.go/servicecontext.go 复核）
- [x] 核查日志、trace、error、metrics、DB/cache/event/file 中的 token 泄漏或负面搜索结果。（audit-report.md §1.1、§5.2；L1–L3 确认 + 负面搜索）
- [x] 形成 receiver-first 迁移、无内容观测、灰度和回滚建议。（audit-report.md §7）
- [x] 输出后续 typed key、claim normalization、policy enforcement 的明确输入和边界。（audit-report.md §8）
- [x] 运行任务文档路径/证据检查、Trellis validate 和 `git diff --check`。（见下方验证结果）
- [x] 向用户提交审计结果并暂停；用户确认前不提交、不归档、不进入下一子任务。（本报告已提交，等待用户确认）

## 验证命令

```bash
python3 ./.trellis/scripts/task.py validate .trellis/tasks/08-14-audit-authorization-propagation
git diff --check
git status --short
```

本任务不修改 Go 代码，因此不以测试命令替代证据复核。若执行期意外出现应用代码或配置 diff，立即停止并回到规划边界。

## 人工门禁

- 开发/执行前：用户批准本 PRD、design、implement 后才能 `task.py start`。
- 交付后：审计结果、候选分类和残余未知提交用户确认；确认后才提交、归档，并开始 `typed-auth-context-keys` 的规划。

## Rollback Points

- 审计结论与代码证据不符时，仅修正文档，不修改代码以“匹配结论”。
- 发现需要立即修复的 token 泄漏时，记录为独立实施项并请求用户批准，不在审计任务越界修改。
- `b64:`、wire key、claim conversion、metadata duplicate 行为均保持原状。
