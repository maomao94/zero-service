# Reorganize docs directory structure

## Goal

将 docs 目录按服务分类重排，更新全部交叉引用链接和索引，为其他子任务（缺失文档、CONTEXT.md、ADR）提供清晰的目录基础。

## Background

父任务 `08-18-organize-docs` 的一部分。docs 目录当前 19 个文件平铺，核心服务文档需要归入子目录。该子任务是父任务的关键路径，其他子任务依赖其完成后的目录结构和链接。

## Requirements

1. 创建服务子目录并移动文档：
   - `docs/iec104/`: iec104.md, iec104-message.md, iec104-command.md
   - `docs/trigger/`: trigger.md, trigger-rrule-api-guide.md, trigger-plan-opengauss-migration.md
   - `docs/socketio/`: socketio.md
   - `docs/djicloud/`: djicloud.md, kml-kmz-guide.md
   - `docs/isp/`: isp.md
2. 通用文档保留在 docs 根目录：architecture.md, development.md, deployment.md, error-codes.md, service-ports.md, quick-start.md, antsx-vs-reactive.md
3. 新增服务文档的最终位置由父任务固定：`docs/file/README.md`、`docs/gis/README.md`、`docs/podengine/README.md`、`docs/bridge/README.md`；这些目录由缺失服务子任务创建
4. 为 5 个现有服务子目录创建 README.md（服务概述 + 文档链接）
5. 更新全仓库 Markdown 和 HTML 中的交叉引用、图片、附件、目录和源码相对路径
6. 以本任务独占的方式更新 docs/README.md 和根 README.md 的现有目录路径；新增文档入口由缺失服务子任务追加
7. 保持现有文档内容语义不变，仅允许必要的路径修正和导航链接

## Acceptance Criteria

- [ ] 5 个服务子目录创建，10 个明确列出的核心服务文档已移动
- [ ] 通用文档保留在 docs 根目录
- [ ] 每个服务子目录有 README.md
- [ ] 所有交叉引用链接已更新且可正常访问
- [ ] docs/README.md 已更新，反映新目录结构
- [ ] 根 README.md 链接已更新
- [ ] `docs/images/iec-architecture.png` 和 `docs/images/trigger-flow.png` 等资源从移动后的文档可访问
- [ ] 移动的文档内容语义保持不变

## Out of Scope

- 修改文档内容语义
- 新增/删除文档（由其他子任务负责）
- 更新 .trellis/spec 索引（不受影响）

## Technical Notes

- docs/README.md 引用 `../app/lalhook/README.md`，该文件在 app 目录下，不随 docs 重排移动
- 移动后同子目录内相对链接保持不变；跨目录链接需加 `../` 前缀
- `docs/iec104.md` 和 `docs/trigger.md` 当前使用 `images/...` 资源，移动后必须改为 `../images/...`
- 完成后由本任务输出最终目录清单，后续子任务以该清单为输入
- 文档间引用关系已在父任务 prd.md 中梳理

## Key Decisions

- 按服务分类，通用文档保留根目录
- 每个服务子目录创建 README.md
- KML/KMZ 归入 djicloud/，openGauss 迁移归入 trigger/

## Open Questions

- 无
