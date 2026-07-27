# 执行计划

- [x] 审计旧 Spec、当前源码/测试/配置和实验/历史排除范围。
- [x] 对照 Trellis 官方模板，确认本项目采用 `README.md + backend/ + guides/`。
- [x] 重建根 `README.md`、`backend/index.md` 和 `guides/index.md`。
- [x] 将基础、公共基础设施和稳定领域契约收敛到 `backend/`。
- [x] 将跨层、复用和文档方法收敛为 `guides/*-guide.md`。
- [x] 逐份复核 Code-Spec 的契约深度、证据、反模式和验证方式。
- [x] 校验索引、Markdown 链接、证据路径、占位文本和排除项。
- [x] 运行 Trellis 上下文检查与 Git 文本/范围检查，记录最终结果。

## 验证命令

```bash
python3 ./.trellis/scripts/get_context.py --mode packages
rg -n 'TBD|TODO: fill|To be filled|placeholder' .trellis/spec
git diff --check
git status --short
```

本任务只改文档，不运行全仓 Go 测试。源码契约真实性通过定向读取现有实现和测试验证。
