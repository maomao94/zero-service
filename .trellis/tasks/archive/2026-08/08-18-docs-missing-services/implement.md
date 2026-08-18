# Implementation plan

1. 确认重排子任务已通过并读取最终目录清单。
2. 分别读取 file、gis、podengine、bridge 的领域规范、契约源码、配置和主要入口。
3. 创建四个服务目录 README，每篇独立核对职责、配置、端口、关键能力和权威契约链接。
4. 由本任务追加 `docs/README.md` 新入口；仅在有直接价值时为 `service-ports.md` 增加文档链接。
5. 运行链接检查、敏感信息检查和 `git diff --check`。

## Rollback Point

四篇文档可按服务独立回退；索引入口必须随对应文档一同回退。
