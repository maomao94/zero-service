# Implementation plan

1. 从稳定 docs、领域 Spec 和服务契约提取候选术语。
2. 删除通用编程词、实验性 AI 词和只在实现内部出现的不稳定名称。
3. 解决同义词与大小写冲突，为每项选择规范名称并列出 `_Avoid_`。
4. 按 CONTEXT-FORMAT 创建根 `CONTEXT.md`，逐项回查权威来源。
5. 检查每项定义是否只回答“是什么”，并运行 `git diff --check`。

## Rollback Point

CONTEXT 是独立新增文件，可整体回退；本任务不批量修改现有文档。
