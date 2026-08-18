# Docs restructure design

## Target Paths

现有 10 个服务文档移动到父任务规定的 5 个服务目录；`docs/images/` 保持共享资源目录。新增服务目录由后续缺失服务子任务创建。

## Link Migration Contract

- 同一服务目录内文档链接保持同层相对路径。
- 从服务目录指向 `docs/images/` 或 docs 根文档使用 `../`。
- 从服务目录指向仓库源码通常由 `../app/...` 改为 `../../app/...`，其他顶层目录同理。
- 扫描范围是全仓库 Markdown 和内嵌 HTML 资源，不限于 `docs/`。

## Ownership

本任务拥有现有路径迁移以及根 README、docs 索引的基线更新。它不添加 file/gis/podengine/bridge 入口，不创建 ADR 或 CONTEXT。
