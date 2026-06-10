# 实施计划

## 实施清单

1. 创建/调整 `cli/dockeru` 目录结构，并添加最小 `main.go`，负责执行 Cobra root command。
2. 添加/调整 `internal/cli/root.go`：定义 root command 元信息、`SilenceUsage`、`SilenceErrors`，默认 `RunE` 启动 Bubble Tea TUI。
3. 在 root/subcommand/flag/args/RunE 这些 Cobra 核心位置写中文学习型注释，解释 Cobra 用法。
4. 新增 `internal/tui`：实现 Bubble Tea `Model`、`Init`、`Update`、`View` 和 Docker action commands，并写中文学习型注释。
5. 在 `internal/docker` 添加 Docker 数据模型和 runner helper：
   - 列出容器。
   - 列出镜像。
   - 执行带终端挂接的 Docker 命令。
   - 执行并捕获输出的 Docker 命令。
6. 保留/实现辅助 Cobra 命令：`ps`、`images`、`start`、`stop`、`restart`、`logs`、`exec`、`compose up`、`image save`、`image prune`。
7. 添加 `cli/dockeru/build.sh`，支持本机和常见跨平台编译目标。
8. 添加/更新 `cli/dockeru/README.md`，用本工具代码作为示例讲清 Cobra + Bubble Tea 开发方式。
9. 对新增 Go 文件执行 `gofmt`。
10. 如果新增 Cobra/Bubble Tea 依赖，执行：
    - `go mod tidy`
11. 执行验证：
    - `go build ./cli/dockeru`
    - `go run ./cli/dockeru --help`
    - `timeout 2s go run ./cli/dockeru` 或等价方式确认 TUI 可启动但不长期阻塞
    - `go run ./cli/dockeru ps --help`
    - `go run ./cli/dockeru images --help`
    - `go run ./cli/dockeru logs --help`
    - `go run ./cli/dockeru exec --help`
    - `go run ./cli/dockeru compose up --help`
    - `go run ./cli/dockeru image save --help`
    - `go run ./cli/dockeru image prune --help`
    - `bash -n cli/dockeru/build.sh`

## 评审关卡

- 确认实现范围只包含 `cli/dockeru` 和必要依赖文件。
- 确认 `util/dockeru` 没被改动。
- 确认 `dockeru` 默认启动 Bubble Tea TUI，`dockeru --help` 仍显示 Cobra help。
- 确认没有旧式一次性 1/2/3 文本菜单。
- 确认没有实现 K8s、SSH 或远程 Docker 管理。
- 确认 Docker 操作用 `exec.Command(name, args...)`，没有拼接 shell 字符串。
- 确认 Cobra + Bubble Tea 代码能作为学习样例阅读，`README.md` 解释了结构。
- 确认 Cobra/Bubble Tea 关键使用点有中文注释，但普通业务代码不过度注释。
- 确认没有写入密钥、个人路径或本地基础设施信息。
- 确认 `image prune` 有确认保护。

## 回滚点

- 如果 Cobra 依赖导致不合适的依赖变更，停止并询问是否改为 `cli/dockeru` 下独立 Go module。
- 如果 Go 版本或 toolchain 阻塞 `go mod tidy`，保留代码改动但先报告工具链阻塞。
- 如果本机没有 Docker，验证范围限制为 build 和 help 命令。
