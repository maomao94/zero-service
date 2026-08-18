# Implementation plan

1. 确认重排子任务完成，读取候选素材的最终路径。
2. 对 antsx、Trigger 主键迁移、IEC 104 三通道逐项记录 hard-to-reverse、surprising、trade-off 证据。
3. 对通过门槛的候选按顺序编号创建 ADR；未通过者记录不创建原因，不占编号。
4. 从普通文档删除重复决策论证或改为摘要，并双向链接；保留操作指南、API 和迁移步骤。
5. 创建 ADR 索引，运行链接检查和 `git diff --check`。

## Rollback Point

每篇 ADR 与素材文档中的对应去重修改必须一起回退，避免权威来源缺失或重复。
