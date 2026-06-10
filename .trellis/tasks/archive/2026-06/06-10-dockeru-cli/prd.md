# Docker 宿主机管理 CLI

## 目标

在 `cli/dockeru` 下创建一个轻量 Go + Cobra + Bubble Tea 终端 UI 工具，命名为 `dockeru`，用于管理宿主机 Docker。新工具需要保留现有 `util/dockeru` 原型里的核心工作流，但改造成“进入后可直接管理”的 TUI，而不是一次性脚本菜单。

这个 CLI 面向本机运维使用：查看容器和镜像、按关键字过滤、启动/停止/重启容器、跟随日志、进入容器 Shell、启动 compose 服务、导出镜像、清理悬空镜像。

这个实现同时也是 Cobra + Bubble Tea 学习样例。用户不熟悉 Cobra，所以代码结构、关键 Cobra 使用点注释和 `README.md` 都要说明 root command、subcommand、flag、args、`RunE` handler 是如何协作的；同时解释 Bubble Tea 的 `Model`、`Init`、`Update`、`View` 和按键处理。

## 已确认事实

- 用户要求使用 Go + Cobra。
- 用户要求工具目录直接放在 `cli/dockeru`。
- 用户要求提供编译脚本。
- 用户希望通过这次 `dockeru` 二次改造学习 Cobra CLI 开发，并使用 Bubble Tea 做终端 UI。
- 现有原型在 `util/dockeru/main.go`。
- 参考 CLI 设计在本机 `/Users/hehanpeng/GolandProjects/cli`，重点参考 Cobra root/subcommand 组装方式。
- 当前仓库 `go.mod` 已有 Docker 相关依赖和 `golang.org/x/term`，但还没有 `github.com/spf13/cobra`。
- 本任务目标是通过本机 Docker CLI 管理宿主机 Docker，不做远程 SSH 或 Kubernetes 管理。
- 交互方式调整为“TUI 主模式 + Cobra 辅助入口”：直接运行 `dockeru` 进入终端 UI，在 UI 中用键盘选择容器/镜像并执行操作；Cobra 仍负责 help、version、入口命令和可选的快速子命令。

## 需求

- 新增 Go 命令工具，根目录为 `cli/dockeru`。
- 使用 Cobra 构建 root command 和必要 subcommands，默认 root command 启动 Bubble Tea TUI。
- 使用 `github.com/charmbracelet/bubbletea` 构建终端 UI。
- 工具保持轻量、自包含，不接入 go-zero 服务。
- 提供以下命令：
  - `ps`：查看容器，等价于 `docker ps -a`，支持按名称或镜像过滤。
  - `images`：查看 Docker 镜像，支持按 repository 或 tag 过滤。
  - `start`、`stop`、`restart`：对选中的容器执行生命周期操作。
  - `logs`：跟随容器日志，支持配置 tail 行数。
  - `exec`：进入容器 Shell，默认 `/bin/bash`，支持通过 flag 覆盖。
  - `compose up`：基于 compose 文件执行 `docker compose up -d <service>`。
  - `image save`：导出镜像为 `.tar` 文件。
  - `image prune`：清理悬空镜像。
- 默认交互形态以 `dockeru` 启动 TUI 为主，在终端 UI 内直接管理 Docker；保留 help/version 和必要快速命令作为辅助入口。
- TUI 至少提供两个视图：容器列表和镜像列表。
- TUI 容器视图支持刷新、启动、停止、重启、查看日志、进入 shell。
- TUI 镜像视图支持刷新、导出镜像、清理悬空镜像。
- TUI 需要展示快捷键帮助，例如 `tab` 切换视图、`r` 刷新、`s` start/stop、`R` restart、`l` logs、`e` exec、`S` save、`p` prune、`q` 退出。
- 支持非交互式参数，方便脚本调用：
  - 列表和选择流程支持 filter flag。
  - 容器、镜像、服务支持显式参数。
  - 未提供精确目标参数时，才使用交互式候选选择作为便利 fallback，例如 `dockeru image save -f nginx` 先列出匹配镜像，再输入序号选择。
- 列表命令输出可读表格。
- `exec`、`logs`、`compose up`、`image save`、`image prune` 等真实 Docker 操作需要透传 stdin/stdout/stderr。
- 在 `cli/dockeru` 提供编译脚本，支持本机平台和常见 Darwin/Linux amd64/arm64 目标。
- 在 `cli/dockeru/README.md` 提供简洁 Cobra 学习说明，解释：
  - root command 如何创建。
  - subcommand 如何注册。
  - flag 和 positional args 如何定义。
  - 为什么使用返回 error 的 `RunE`。
  - 如何按现有模式新增一个 dockeru 命令。
- README 还要解释 Bubble Tea 学习内容：
  - `Model` 保存 UI 状态。
  - `Init` 如何触发初始加载。
  - `Update` 如何处理按键和 Docker 操作结果。
  - `View` 如何渲染终端界面。
  - Cobra 如何启动 Bubble Tea program。
- 在 Cobra 关键使用点写学习型注释，重点解释 Cobra 机制；不要给普通业务代码写无意义注释。
- Cobra 代码保持易读：优先使用明确的 command constructor 函数，不做过度抽象。
- 不写入个人路径、命名空间、主机、密码或项目特定容器名。
- 除依赖更新必须修改根 `go.mod` / `go.sum` 外，代码范围保持在 `cli/dockeru`。

## 验收标准

- [ ] `go run ./cli/dockeru --help` 能打印 Cobra help 并成功退出。
- [ ] `go run ./cli/dockeru` 会启动 Bubble Tea TUI，而不是只打印 help。
- [ ] TUI 能在没有 K8s/SSH 的情况下展示本机 Docker 容器/镜像管理界面。
- [ ] TUI 有清晰的键盘快捷键说明，并支持退出、刷新、切换容器/镜像视图。
- [ ] `go run ./cli/dockeru ps --help`、`images --help`、`logs --help`、`exec --help`、`compose up --help`、`image save --help`、`image prune --help` 都能运行，且不要求 Docker daemon 正在运行。
- [ ] `cli/dockeru/build.sh` 存在、可执行，并清楚实现编译产物输出路径。
- [ ] `cli/dockeru/README.md` 用本工具示例解释 Cobra 和 Bubble Tea 开发模式，并包含运行和扩展示例。
- [ ] 代码结构能让用户看懂 Cobra 概念和 Bubble Tea 概念：例如 `root.go`、命令文件、flags、args、`RunE` handler、TUI model/update/view。
- [ ] Cobra 和 Bubble Tea 关键使用点有中文学习型注释，能帮助理解 root/subcommand/flag/args/RunE/model/update/view。
- [ ] `go build ./cli/dockeru` 成功。
- [ ] 如果新增 Cobra 依赖，执行 `go mod tidy` 并检查依赖 diff。
- [ ] 运行 Docker 操作时使用 `exec.Command(name, args...)` 参数数组，不拼接 shell 命令字符串。
- [ ] 不写入密钥、个人路径或本地基础设施信息。
- [ ] 现有 `util/dockeru` 保持不变，除非后续明确要求替换。

## 非目标

- 远程 SSH Docker 管理。
- `pod-enter-app.sh` / `pod-log-app.sh` 里的 Kubernetes pod 管理。
- 旧式一次性 1/2/3 文本菜单；新的交互方式是 Bubble Tea TUI。
- Docker SDK 集成；第一版优先包装宿主机 Docker CLI，以贴近现有原型。
- 配置 profile 或持久化状态。
- 除编译脚本以外的安装包、发布包或安装器。

## 开放问题

- 无阻塞问题。默认范围是：本机 Docker CLI 管理工具 + 可学习的 Cobra 示例代码和说明。
