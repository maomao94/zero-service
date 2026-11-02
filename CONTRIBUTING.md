# 贡献指南

感谢你对 Zero-Service 项目的关注！

提交代码前请先阅读 [README](./README.md)（环境要求 Go 1.26+）和 [开发指南](./docs/development.md)。参与本项目即表示同意遵守[贡献者公约](./CODE_OF_CONDUCT.md)；安全问题请按[安全策略](./SECURITY.md)私密报告，不要公开提 Issue。

## 分支模型（Git Flow）

本仓库使用 Git Flow（见 `.gitflow`），主分支为 `master`（稳定可发布）与 `develop`（集成开发）：

- `feature/*`：从 `develop` 切出，完成后提 PR 合回 `develop`
- `release/*` / `hotfix/*`：按需使用，前缀见 `.gitflow`
- 版本标签前缀为 `v`

```bash
git checkout develop
git pull origin develop
git checkout -b feature/your-feature
# ... 修改并提交 ...
git push origin feature/your-feature
# 在 GitHub 提 PR：feature/your-feature -> develop
# 发布时再由 develop -> master 提 PR
```

不使用 Fork 流程，直接在本仓库按上述分支提 PR 即可（Fork 仅在无写权限时使用）。

## 代码规范

### Go 代码

- 遵循 Go 官方编码规范，驼峰命名，首字母大写导出。
- 命名约定（见 `docs/development.md`）：
  - API 请求/响应：`XxxRequest` / `XxxResponse`
  - gRPC 请求/响应：`XxxReq` / `XxxRes`
- 分层：Handler/Server -> Logic -> Model/SDK，跨服务复用能力放入 `common/`。
- 错误码使用 `google.rpc.Code` 标准，业务码按 `docs/error-codes.md` 经 `detail.reason` 扩展。
- 配置安全：示例配置只保留占位值，不得提交真实密钥与敏感连接信息。

### 提交前检查（与 CI 对齐）

```bash
gofmt -l .            # 应无输出
go vet ./...          # 对应 .github/workflows/go.yml 的 Vet 步骤
go test -race ./...   # 对应 go.yml 的 Test 步骤，按需加包路径缩小范围
python3 scripts/check_docs.py  # 改动了 Markdown 时，对应 docs.yml
git diff --check      # 应无输出
```

说明：

- CI 触发分支为 `master` 与 `develop`（`go.yml`：push/PR 到这两个分支都会跑）。
- PR 还会跑 `reviewdog.yml`（staticcheck，`fail_level: any`，见文件内注释的排除项）。
- 系统依赖：用到 `common/gisx/geos` 时 CI 需装 `libgeos-dev`，本地 macOS 用 `brew install geos pkg-config` 对齐。

### 提交规范

使用 Conventional Commits 格式：

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

类型：
- `feat`：新功能
- `fix`：修复
- `docs`：文档
- `style`：格式（不影响代码运行）
- `refactor`：重构
- `test`：测试
- `chore`：构建/工具依赖（如 `chore(deps)`）

示例（格式示意，勿照抄业务内容）：
```
feat(<scope>): 新功能一句话描述
fix(<scope>): 修复一句话描述
```

### 代码生成

- 修改 `.proto` 或 `.api` 后，必须运行**对应服务目录**的 `gen.sh`（如 `app/trigger/gen.sh`），不要只在仓库根目录找统一入口。
- 只编辑契约源文件（`.proto`/`.api`），不要手写或修改生成的 handler、server、routes、`*.pb*.go` 文件。
- 数据库模型脚本在 `model/` 下（`genModel.sh` / `genPgModel.sh` / `genModelSql.sh`），按需使用。
- 提交前检查生成文件的 diff，只提交任务范围内的改动。

## Pull Request 要求

- 使用仓库的 [PR 模板](./.github/PULL_REQUEST_TEMPLATE.md)填写变更说明、影响范围与验证结果（贴命令输出，不要只写「已测试」）。
- PR 标题使用与提交相同的 Conventional Commits 格式，并写清目标分支（`-> develop` 或 `-> master`）。
- 关联相关 Issue，包含变更说明与验证方式（跑了哪些 `go test` / `go vet`）。
- 通过 CI 检查（`go vet`、`go test -race`、reviewdog staticcheck、`Docs` 文档校验）。
- 等待 Reviewer 批准；CI 变红先本地复现再推新提交。

## 版本与变更记录

- 版本号遵循[语义化版本](https://semver.org/lang/zh-CN/)，标签前缀为 `v`（见 `.gitflow`）。
- 面向使用者的改动（新增能力、行为变更、弃用、修复）需在 [`CHANGELOG.md`](./CHANGELOG.md) 的「未发布」下补一条，按 `新增` / `变更` / `弃用` / `移除` / `修复` / `文档` / `依赖` 分类。
- `release/*` 合并到 `master` 前，把「未发布」改为目标版本号与发布日期，再由 tag 触发发布说明。

## Issue 规范

### Bug 报告

- 描述问题现象
- 复现步骤
- 期望行为
- 实际行为
- 环境信息（Go 版本、OS 等）

### 功能请求

- 描述需求背景
- 期望的解决方案
- 替代方案（如有）

## 文档贡献

- 文档使用中文
- 保持格式一致
- 代码示例使用 fenced code block
- 新增服务必须补充文档
- 新增文档先在 [`docs/README.md`](./docs/README.md) 的对应分区登记，再补交叉链接
- 架构图与流程图放在 `docs/images/diagrams/`，提交同名 `.mmd` 源文件与导出的 PNG，正文用 `<img>` 引用，约定见该目录的 [`README.md`](./docs/images/diagrams/README.md)
- 提交前执行 `python3 scripts/check_docs.py`：校验相对链接、HTML `<img>`/`<a href>`、页内锚点（含重复标题的 `-1` 后缀）、代码块闭合与 `.mmd`/`.png` 配对

### 文档体量与拆分

- 单篇超过 **600 行**时按语义拆分，不要靠目录硬撑：规则与示例（如 `trigger-rrule-syntax.md` 与 `trigger-rrule-api-guide.md`）、正文与附录（如 `iec104-message.md` 与 `iec104-reference.md`）。
- 拆分后父文档保留入口表格，子文档互相链接，并在分区 `README.md` 登记；不复制正文。
- 包级 API 说明放组件目录的 `README.md`，跨服务的选型与实践放 `docs/`，避免同一份 API 在两处各讲一遍。
- `docs/` 的分区索引（`docs/<topic>/README.md`）只做导航，不承载正文。

### 不收录的内容

- 不在仓库内抄录外部工具、依赖库或 AI 技能集的目录与全量说明，只保留上游链接、安装位置和版本核对方式（参见 [`docs/ai-collaboration.md`](./docs/ai-collaboration.md)）。
- 端口、字段、错误码、方法数等易变事实，以源码、`.proto`/`.api` 与 `etc/*.yaml` 为准，文档只给结论并标明出处。


## 问题反馈

- GitHub Issues：[https://github.com/maomao94/zero-service/issues](https://github.com/maomao94/zero-service/issues)
