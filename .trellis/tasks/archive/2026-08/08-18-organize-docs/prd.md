# Organize project documentation

## Goal

把 Zero-Service 文档整理为可导航、可验证、权威来源清晰的知识体系：先重排现有服务文档，再补齐选定服务文档、建立项目术语表，并仅为证据充分的关键决策创建 ADR。

## Background

- `docs/` 当前平铺 19 个 Markdown 文件和 `images/` 资源目录；其中 10 个现有服务文档计划移动。
- `service-ports.md` 列出大量服务，但 file、gis、podengine 和 bridge 系列没有专项文档。
- 仓库没有 `CONTEXT.md` 和 `docs/adr/`。
- `.trellis/spec/guides/documentation-guide.md` 要求同一事实只有一个权威来源，README 负责入口，`docs/` 负责完整流程，协议源码负责字段和接口契约。
- AI 服务属于实验性内容，不纳入本任务。

## Requirements

### 子任务与顺序

1. `08-18-docs-restructure`：先完成现有 10 个服务文档的目录重排、资源路径迁移和基线索引更新。
2. `08-18-docs-missing-services`：在重排完成后新增 file、gis、podengine、bridge 文档，并更新新增入口。
3. `08-18-docs-context`：可独立实施；建立规范术语和别名映射，不承诺修改所有现有文档。
4. `08-18-docs-adr`：在重排完成后审查 3 个 ADR 候选；只创建满足 ADR 门槛且证据充分的记录。
5. 父任务仅负责最终集成检查，不作为直接实现目标。

### 公共文件所有权

- `docs-restructure` 拥有根 `README.md`、`docs/README.md` 的目录迁移基线更新。
- `docs-missing-services` 只追加新服务入口，并按需更新 `docs/service-ports.md` 中与新增文档直接相关的链接。
- `docs-adr` 只创建 `docs/adr/`、ADR 索引，并在素材文档中添加权威来源链接或删除重复决策论证。
- `docs-context` 只创建根 `CONTEXT.md`；现有文档术语批量统一不在本任务内。

## Acceptance Criteria

- [ ] 四个子任务按依赖顺序完成并各自通过质量检查。
- [ ] 全仓库 Markdown 中指向移动文档、图片、附件和仓库源码的相对路径有效。
- [ ] 根 `README.md` 保持入口性质；详细内容仍以 `docs/` 和契约源码为权威来源。
- [ ] 新服务文档只覆盖选定的稳定服务，AI 和明确排除的内部服务未被扩入范围。
- [ ] `CONTEXT.md` 仅包含稳定、项目特有的规范术语和别名，不含配置、端口、字段、算法或执行步骤。
- [ ] 每个 ADR 候选都有门槛判定；未满足门槛的候选不会被强行写成 ADR。
- [ ] 没有因新增 ADR 或索引而形成重复权威来源。
- [ ] `git diff --check` 通过，最终差异不包含真实凭据、内网地址或个人路径。

## Target Layout

```text
CONTEXT.md
docs/
  README.md
  architecture.md
  deployment.md
  development.md
  error-codes.md
  quick-start.md
  service-ports.md
  images/
  adr/
  bridge/README.md
  djicloud/{README.md,djicloud.md,kml-kmz-guide.md}
  file/README.md
  gis/README.md
  iec104/{README.md,iec104.md,iec104-message.md,iec104-command.md}
  isp/{README.md,isp.md}
  podengine/README.md
  socketio/{README.md,socketio.md}
  trigger/{README.md,trigger.md,trigger-rrule-api-guide.md,trigger-plan-opengauss-migration.md}
  antsx-vs-reactive.md
```

## Out of Scope

- AI 服务和术语。
- alarm、logdump、xfusionmock、lalproxy 等未选定内部服务的专项文档。
- Mermaid/ASCII 架构图重绘和全量文案润色。
- 修改项目代码行为或协议契约。
- 为达到词条数量而收录通用编程概念。

## Key Decisions

1. 现有服务文档按服务目录归档；单篇新增服务直接以子目录 `README.md` 作为专项文档。
2. KML/KMZ 指南归入 `djicloud/`；Trigger openGauss 迁移指南归入 `trigger/`。
3. AI 服务整体排除。
4. CONTEXT 使用根目录单一词汇表，范围由稳定术语清单决定，不使用 `60+` 数量指标。
5. ADR 采用证据门控，不预先保证创建 3 篇。

## Open Questions

- 无。
