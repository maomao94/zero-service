# 重新梳理项目 Trellis Spec

## Goal

基于当前 `zero-service` 的真实代码、测试、配置与项目文档，重新设计并编写
`.trellis/spec/`，让未来开发者和 AI 能快速获得与改动范围匹配的编码约束、
模块边界和验证要求，而不必阅读冗长、过时或实验性的说明。

## Confirmed Facts

- 仓库是一个 Go module 下的多服务集合，核心目录包括 `app/`、`aiapp/`、
  `socketapp/`、`gtw/`、`facade/`、`common/` 和 `model/`。
- 当前 spec 全部集中在 `backend/`，约 1.1 万行；既包含全局规则，也包含公共库
  API、业务协议细节、历史迁移结论和实验性组件说明。
- `.proto` / `.api` 与各服务 `gen.sh` 构成服务契约和生成边界；业务实现主要位于
  `internal/logic`，依赖由 `internal/svc/ServiceContext` 组装。
- `common/gnetx` 的源码明确包含原型用途说明，旧 spec 也将整个组件标为实验性。
- `app/xfusionmock` 是 Demo/测试服务；`1.7.1/`、`1.9.x/` 是历史模型快照；生成代码
  和 `third_party/` 不是项目编码规则的来源。

## Requirements

- 可删除、合并、拆分、重命名现有 spec 文件，不维持旧文件名兼容。
- 目录遵循 Trellis 官方模板职责：根 `README.md`、技术层 `backend/` 和 `guides/`；本项目没有前端，不创建空 `frontend/`。
- 每条重要规则必须有当前源码、测试、配置或项目文档依据，并写出可定位的相对路径。
- 只保留能指导后续改动的稳定约束；实验性、原型、Mock、历史快照和纯生成代码不建立
  独立 spec。
- 高风险且已验证的并发、持久化、协议和身份语义必须保留，但应压缩成开发契约，
  不复制完整 API 手册或历史排障过程。
- spec 与 `docs/` 分工明确：spec 说明“修改代码时必须遵守什么”，用户/协议说明链接到
  `docs/`，避免重复维护。
- 本任务只修改 `.trellis/spec/` 和本任务规划/研究文件，不修改产品源码。
- 全部索引必须与最终文件集一致，并提供 Pre-Development Checklist 与 Quality Check。
- 文档使用中文，代码标识、路径、命令保持原文。

## Acceptance Criteria

- [x] `.trellis/spec/` 采用 `README.md + backend/ + guides/`，Code-Spec 与 Guide 职责分开。
- [x] 所有最终 spec 都有明确的适用场景、项目规则、证据路径、反模式和针对性验证方式。
- [x] 实验性/原型/Mock/历史快照没有独立 spec，也不出现在开发索引的必读项中。
- [x] `crontask`、`gormx`、ISP、IEC 104、DJI、GIS、消息与实时通信等稳定高风险契约
  经当前源码和测试复核后被保留或合理合并。
- [x] 所有 `index.md` 与实际文件一致；Markdown 相对链接和引用的源码路径均存在。
- [x] 不存在 `TBD`、占位文本、空标题、复制模板或仅凭通用经验编写的规则。
- [x] `python3 ./.trellis/scripts/get_context.py --mode packages` 只列出 `backend` 技术层。
- [x] `git diff --check` 通过，且最终 diff 不包含产品源码改动。

## Out Of Scope

- 修改产品代码、生成代码、配置或部署脚本。
- 为每个小型 `common/` 工具包逐一编写 API 参考。
- 为 `common/gnetx`、`app/xfusionmock`、历史模型快照或其他明确实验/测试用途代码
  编写独立规范。
- 替代 `docs/` 下的用户文档、部署文档或完整协议文档。

## Open Questions

- 无阻塞问题。最终目录设计以“少而高信号”为原则：稳定高风险契约保留独立文件，
  低风险包通过公共组件总则和相邻代码约束覆盖。
