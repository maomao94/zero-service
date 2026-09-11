# OpenViking 记忆系统

本项目的稳定工程契约与领域规范由 OpenViking 记忆系统提供，替代已移除的 `docs/agent/` 本地知识库。本文档描述记忆系统的部署、MCP 工具与使用指导。

## 概述

| 项 | 值 |
| --- | --- |
| 部署仓库 | `openviking-memory`（GitHub 私库，自包含：compose + 配置 + data） |
| 服务地址 | `http://localhost:1933`（API + Web Studio `/studio`） |
| 账号 / 用户 | `developer` / `admin`（admin 角色） |
| 本项目的知识 | `viking://resources/project/zero-service/`（standards / domains / guides） |
| 存储 | `~/.openviking` 由容器挂载到 `/app/.openviking`，workspace 指向 `/app/.openviking/data` |

启动与验证：

```bash
cd ~/openviking-memory
docker compose up -d
curl http://localhost:1933/health    # {"status":"ok"}
curl http://localhost:1933/ready     # 就绪检查（AGFS/VectorDB/Embedding）
```

## 接入方式

**OpenCode 插件**（开发时的主要通道）：凭据通过环境变量注入，已在 `~/.zshrc` 配置：

```bash
export OPENVIKING_URL="http://localhost:1933"
export OPENVIKING_API_KEY="<user key>"
```

插件每轮自动召回相关记忆（`<openviking-context>`），并注册 MCP 工具；修改凭据后需重启 OpenCode。

**Web Studio**：<http://localhost:1933/studio>，浏览器查看/管理记忆，连接时填 Server URL 与 user key。

## MCP 工具（15 个，OpenCode 中带 `openviking_` 前缀）

### 检索

| 工具 | 用途 |
| --- | --- |
| `openviking_find` | 快速语义检索，返回带 URI/摘要/得分的排序结果 |
| `openviking_search` | 深度语义检索；`mode="context"` 返回注入就绪的上下文块（可按类别配额、token 预算），替代原 recall 工具 |
| `openviking_read` | 读取一个或多个 `viking://` 文件 |
| `openviking_list` | 列目录（支持递归、排序、分页） |
| `openviking_tree` | 递归目录树（可带摘要） |
| `openviking_grep` | 精确文本/正则搜索 |
| `openviking_glob` | 文件名模式匹配 |

检索可以限定范围（`target_uri`），例如只在 zero-service 契约内搜：

```
openviking_find query="handler 分层职责" target_uri="viking://resources/project/zero-service"
```

### 写入

| 工具 | 用途 |
| --- | --- |
| `openviking_remember` | 沉淀偏好、决策、结论（进入用户长期记忆） |
| `openviking_write` | 写入 `viking://` 文件（replace / create / append） |
| `openviking_edit` | 对已有 `viking://` 文件做精确字符串替换 |
| `openviking_add_resource` | 导入 URL / 本地文件 / 目录（生成摘要与向量） |
| `openviking_forget` | 永久删除一个 URI（需确认） |

### 运维

| 工具 | 用途 |
| --- | --- |
| `openviking_health` | 服务健康检查 |
| `openviking_list_watches` / `openviking_cancel_watch` | 查看/取消资源自动刷新订阅 |

## 记忆层级

```
viking://resources/                          # 共享知识资源（团队级）
└── project/zero-service/                    # 本项目的规范与契约（tags: lang=go, framework=go-zero, project=zero-service）
    ├── standards/                           # 工程规范
    ├── domains/                             # 领域契约
    └── guides/                              # 分析指南
viking://user/admin/                         # 用户私有空间（memories / skills / peers / privacy）
```

命名约定：AI 编程记忆按 **语言/框架/项目** 三层组织（`lang/`、`framework/`、`project/`），检索时按当前任务组合限定 `target_uri`，避免跨项目污染。

## 使用指导

1. **开发中查询契约**：直接问或调 `openviking_find` / `openviking_search`；需要精确规则时 `openviking_read` 对应 URI。插件自动召回是近似提醒，落地前以源码、测试、`.proto`、`.api` 为准。
2. **沉淀新结论**：任务中形成的稳定决策用 `openviking_remember` 记录；契约/规范更新用 `openviking_write` / `openviking_edit` 修改 `viking://resources/project/zero-service` 下对应文档。
3. **新项目初始化**（记忆为空时的一次性操作）：用 `openviking_add_resource` 或容器内 `ov add-resource` 导入项目现有 md 文档，此后不再批量导入。
4. **冲突处理**：记忆与源码/事实冲突时，查明当前行为，并在同一改动中修正记忆。

## 同步到 git

记忆变更落在 `openviking-memory` 仓库的 `data/` 中，任务收尾时整体同步：

```bash
cd ~/openviking-memory
docker compose down            # 保证 SQLite/RocksDB 干净落盘
git add -A && git commit -m "chore(memory): sync"
git push
docker compose up -d
```

其他机器同步：`docker compose down && git pull && docker compose up -d`。

## 排障

| 现象 | 处理 |
| --- | --- |
| 插件 401 / Invalid API Key | 检查 `~/.zshrc` 的 `OPENVIKING_API_KEY`；user key 变更后重启 OpenCode |
| 服务不可用 | `cd ~/openviking-memory && docker compose up -d`，再看 `curl http://localhost:1933/health` |
| recall 为空 | 确认 `autoRecall.enabled` 为 true、服务端已有记忆、query 长度足够 |
| 导入本地目录失败 | 当前 MCP 通道只支持单文件上传；目录导入用容器内 `ov add-resource` |
| 插件日志 | `~/.config/opencode/openviking/openviking-memory.log`、`openviking-session-state.json` |

## 参考

- 官方文档：<https://docs.openviking.ai/zh/getting-started/02-quickstart>
- OpenCode 插件：<https://docs.openviking.ai/zh/agent-integrations/10-opencode>
- 部署与迁移：`openviking-memory` 仓库 README
