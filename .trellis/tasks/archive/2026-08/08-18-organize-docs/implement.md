# Parent integration plan

## Execution Order

1. 完成并检查 `08-18-docs-restructure`。
2. 基于最终目录，可并行完成 `08-18-docs-context`；随后完成 `08-18-docs-missing-services` 和 `08-18-docs-adr`。
3. 返回父任务执行全量集成检查，不直接新增功能文档。

## Integration Checklist

- [ ] 目标目录树与父 PRD 一致。
- [ ] 根 README、docs 索引、新增服务入口和 ADR 索引职责不重叠。
- [ ] 全仓库 Markdown/HTML 相对链接目标存在，移动后的源码链接路径深度正确。
- [ ] `docs/images/` 资源可从移动后的文档访问。
- [ ] ADR 与普通文档没有重复维护同一决策论证。
- [ ] CONTEXT 没有通用编程术语或实现细节。
- [ ] 排除范围未扩张。

## Validation

```bash
git diff --check
git status --short
```

另用确定性链接检查脚本扫描全仓库 Markdown 和内嵌 HTML 资源；脚本若需临时创建，放在批准的临时目录，不提交到仓库。人工复核 GitHub 锚点、外部 URL、索引完整性和敏感信息。

## Rollback Points

- 每个子任务检查通过后形成一个逻辑回滚点。
- 集成问题回到对应文件所有者子任务修复，不在父任务复制或恢复旧路径文件。
