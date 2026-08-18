# Record architecture decision records

## Goal

建立 docs/adr/ 目录，对候选决策进行证据审查，只为同时满足"难以逆转、缺少上下文会令人困惑、存在真实取舍"的候选创建 ADR。

## Background

父任务 `08-18-organize-docs` 的一部分。当前无 ADR 目录，决策型内容散落在普通文档中。需将已有充分论证的决策沉淀为 ADR，供未来读者理解"为什么这么做"。

用户指示：openGauss 迁移已全部完成，Trigger Plan 迁移指南（`docs/trigger/trigger-plan-opengauss-migration.md`）与对应 ADR 候选不再需要，已删除；ADR 重新编号为 0001（antsx）、0002（IEC 104 三通道）。

## Requirements

1. 创建 `docs/adr/` 目录
2. 审查并酌情写 ADR-0001：antsx 选择（vs Java WebFlux/RxJava），素材来自 `docs/antsx-vs-reactive.md`
3. 审查并酌情写 ADR-0002：IEC 104 三通道分发（Kafka/MQTT/gRPC），素材来自 `docs/iec104/iec104.md`
4. 遵循 domain-modeling 的 ADR-FORMAT
5. 仅记录文档中已有充分论证的决策，不编造

## Acceptance Criteria

- [ ] docs/adr/ 目录已创建
- [ ] 每个通过证据门槛的候选已创建 ADR；不满足者记录不创建及原因
- [ ] 每篇 ADR 使用 `000N-slug.md` 顺序编号和一至三句核心决策说明；仅在有价值时加入状态、备选方案、后果
- [ ] 内容基于现有文档和源码事实
- [ ] 原文档保留其操作/API 内容；决策理由只保留一个权威来源，相关文档互相链接而不复制整段
- [ ] 无 openGauss 迁移相关文档和引用残留

## Out of Scope

- 新增文档中没有论证依据的决策
- Socket.IO 网关、DJI 双通道等论证不完整的决策（留空位后续补充）
- AI 相关决策（实验性内容）
- openGauss 迁移相关（用户指示已完成，删除）

## Technical Notes

- ADR 目录可放在 docs/adr/，随 docs 目录结构存在
- 必须在 `08-18-docs-restructure` 通过检查并提供最终目录清单后开始
- 素材文档在重排后会移动到子目录，ADR 中引用需用最终路径

## Key Decisions

- 候选范围：antsx、IEC 104 三通道（Trigger 主键迁移已按用户指示删除）
- 每篇 ADR 均通过证据审查（antsx 与三通道均满足三项门槛）

## Open Questions

- 无
