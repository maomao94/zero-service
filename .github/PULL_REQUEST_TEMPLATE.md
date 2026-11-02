<!--
提交前请确认：
- 标题使用 Conventional Commits 格式，例如 fix(trigger): 修正 CronJob 补触发边界
- 目标分支为 develop（feature/fix）或 master（release/hotfix）
- 已阅读 CONTRIBUTING.md 的分支模型与提交规范
-->

## 变更说明

<!-- 这个 PR 解决什么问题？用 1-3 句话说明，避免只贴 diff。 -->

## 关联 Issue

<!-- 例如：Closes #123 / Refs #456；没有则写「无」 -->

## 变更类型

- [ ] 新功能（feat）
- [ ] 缺陷修复（fix）
- [ ] 重构（refactor）
- [ ] 文档（docs）
- [ ] 构建 / 依赖（chore）
- [ ] 其他：

## 影响范围

<!-- 涉及的服务、接口、协议或配置；如修改了 .proto/.api，请列出受影响的服务 -->

## 验证方式

<!-- 贴出实际执行的命令与结果，不要只写「已测试」 -->

```bash
gofmt -l .            # 应无输出
go vet ./...          # 
go test -race ./...   # 或缩小到受影响的包
python3 scripts/check_docs.py  # 改动了 Markdown 时
git diff --check      # 应无输出
```

结果：

## 截图 / 日志

<!-- 前端改动请附截图；接口改动请附请求与响应示例（去掉敏感信息） -->

## 检查清单

- [ ] 改动范围与 Issue 一致，未夹带无关修改
- [ ] 修改 `.proto` / `.api` 后已运行对应目录的 `gen.sh`，未手改生成文件
- [ ] 新增服务或接口已补充文档，并在 `docs/README.md` 登记
- [ ] 配置示例只包含占位值，无真实凭据
- [ ] 破坏性变更已在「变更说明」中标注
