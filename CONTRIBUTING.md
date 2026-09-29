# 贡献指南

感谢你对 Zero-Service 项目的关注！

提交代码前请先阅读 [README](./README.md)（环境要求 Go 1.26+）和 [开发指南](./docs/development.md)。

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
git diff --check      # 应无输出
```

说明：

- CI 触发分支为 `master`（`go.yml`：push/PR 到 `master` 才跑）。
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

- PR 标题使用与提交相同的 Conventional Commits 格式，并写清目标分支（`-> develop` 或 `-> master`）。
- 关联相关 Issue，包含变更说明与验证方式（跑了哪些 `go test` / `go vet`）。
- 通过 CI 检查（`go vet`、`go test -race`、reviewdog staticcheck）。
- 等待 Reviewer 批准；CI 变红先本地复现再推新提交。

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

## 问题反馈

- GitHub Issues：[https://github.com/maomao94/zero-service/issues](https://github.com/maomao94/zero-service/issues)
