<!--
提交前请确认：
- 标题使用 Conventional Commits 格式，例如 fix(trigger): 修正 CronJob 补触发边界
- 变更类型由标题前缀表达（feat/fix/docs/style/refactor/test/chore/ci/perf/build/revert），正文不重复勾选：
  单选语义的复选框会让 PR 列表永久显示 "x of N tasks"，稀释下方自检清单的信号
- 目标分支为 develop（feature/fix）或 master（release/hotfix）
- 已阅读 .github/CONTRIBUTING.md 的分支模型与提交规范
- 机械检查由 CI 负责：gofmt（本 PR 改动的 .go）、go vet、go test -race、
  Markdown 链接校验、git diff --check。这里只记录 CI 判断不了的信息。
-->

## 变更说明

<!-- 这个 PR 解决什么问题？用 1-3 句话说明，避免只贴 diff。 -->

## 关联 Issue

<!-- 例如：Closes #123 / Refs #456；没有则写「无」 -->

## 影响范围

<!-- 涉及的服务、接口、协议或配置；如修改了 .proto/.api，请列出受影响的服务 -->

## 破坏性变更

<!-- 接口、协议、配置或数据结构是否有不兼容改动？有则写明迁移与回滚方式；无则写「无」 -->

## 验证方式

<!-- 贴出实际执行的命令与关键输出（可截取，注意去掉敏感信息），不要只写「已测试」。
     CI 已覆盖的项无需复述结论，但本地额外做过的验证（手工调用、迁移演练、压测）要写在这里。
     例：
     go test -race ./app/trigger/...     → ok  app/trigger/...
     curl -s .../api/v1/xxx | jq .code  → 0
-->

## 截图 / 日志

<!-- 前端改动请附截图；接口改动请附请求与响应示例（去掉敏感信息） -->

## 检查清单

只保留需要人工判断的项；可自动化的检查一律交给 CI。

- [ ] 不看 diff，仅凭「变更说明 + 影响范围」就能理解这个 PR 做了什么、影响了谁，且没有夹带无关修改
- [ ] 破坏性变更已写明迁移与回滚方式（无则写「无」）
- [ ] 「验证方式」贴的是真实执行过的命令与输出，不是「已测试」这类结论
