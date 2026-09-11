# zero-service Agent Guide

## 默认行为

- 使用中文沟通；标识符、协议字段、命令和路径保留原文。
- 修改前搜索定义、直接调用方、测试、配置和生成入口，以当前源码和契约为准。
- 改动紧贴需求，沿用相邻代码的结构、错误和测试风格。
- 结构化数据使用项目已有解析器或标准库解析。
- 示例使用占位配置；日志、输出和提交内容不得包含凭据或敏感连接信息。

## 按需上下文

项目契约与规范由 OpenViking 记忆系统提供（`openviking-memory` 私库，服务地址 `http://localhost:1933`）：OpenCode 插件自动召回，需要精确规则时用 `openviking_search` / `openviking_read` 检索 `viking://resources/project/zero-service`。记忆系统说明见 [`docs/openviking-memory.md`](docs/openviking-memory.md)。

源码、测试、`.proto`、`.api`、配置和生成脚本是事实来源。记忆召回与事实冲突时，先查明当前行为，并在同一改动中修正记忆。

## 记忆同步

开发过程中**通过 OpenViking MCP 工具**形成和更新记忆，不进入容器：

- `openviking_remember` — 沉淀偏好、决策、结论
- `openviking_write` / `openviking_edit` — 写入/修改 `viking://resources/project/zero-service` 下的契约与规范
- `openviking_search` / `openviking_find` / `openviking_read` — 检索与读取

仅当项目尚无记忆（新项目/新环境初始化）时，才用项目现有 md 文档一次性导入形成初始记忆：`openviking_add_resource` 或容器内 `ov add-resource`。此后日常迭代一律通过上面的 MCP 工具更新，不再批量导入文档。

记忆变更落在 `openviking-memory` 仓库的 `data/` 中，任务收尾时整体同步到 git：

```bash
cd ~/openviking-memory
docker compose down            # 保证 SQLite/RocksDB 干净落盘
git add -A && git commit -m "chore(memory): sync"
git push
docker compose up -d
```

- 容器未运行时，先在 `openviking-memory` 仓库执行 `docker compose up -d`。

## 完成标准

- 修改 `.proto` 或 `.api` 后运行对应目录的 `gen.sh`，编辑契约源而非生成文件。
- 记忆变更后按「记忆同步」一节同步 `openviking-memory` 仓库。
- Go 文件经过 `gofmt`，并运行与改动风险匹配的目标测试、race test 或 `go vet`。
- `git diff --check` 通过；最终 diff 只包含任务范围，未验证项在交付时明确说明。
