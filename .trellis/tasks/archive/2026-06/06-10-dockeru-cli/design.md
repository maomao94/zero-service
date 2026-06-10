# Dockeru CLI 技术设计

## 架构

`cli/dockeru` 是主仓库模块内的独立 Go CLI + TUI 入口：

```text
cli/dockeru/
  main.go
  README.md
  build.sh
  internal/
    cli/
      root.go
      tui.go
      ps.go
      images.go
      lifecycle.go
      logs.go
      exec.go
      compose.go
      image.go
    docker/
      runner.go
      container.go
      image.go
    ui/
      table.go
      select.go
    tui/
      model.go
      view.go
      actions.go
```

整体参考飞书 CLI 的 Cobra command constructor 风格，但默认 root command 会启动 Bubble Tea TUI。Cobra 负责入口、help、version、flags 和辅助命令；Bubble Tea 负责进入后的终端交互管理。

## Cobra 学习型结构

因为本任务也是 Cobra 学习练习，每个命令文件要保持直观：

- `NewRootCommand()` 组装整棵命令树。
- 每个功能文件暴露 `NewXCommand()` 或小型 group constructor。
- flag 定义放在拥有该 flag 的 command 附近。
- `Args` validator 用来说明位置参数要求。
- `RunE` 只负责连接 Cobra 和具体执行逻辑，复杂逻辑放到小 helper，方便区分“Cobra 接线”和“业务逻辑”。
- 在 root command、subcommand 注册、flag 绑定、args 校验、`RunE` 处添加中文学习型注释，解释为什么这么写。

`cli/dockeru/README.md` 作为本工具的短教程，不写成泛泛文档，而是用当前代码解释如何开发 Cobra + Bubble Tea CLI/TUI。

## Bubble Tea TUI 结构

TUI 使用 Bubble Tea 的 Model-Update-View 架构：

- `model.go`：保存当前 tab、光标、容器列表、镜像列表、状态消息和错误信息。
- `Init()`：启动后加载容器和镜像。
- `Update(msg tea.Msg)`：处理按键、窗口变化和 Docker 操作返回消息。
- `View()`：把当前状态渲染为终端字符串。
- `actions.go`：把 Docker 查询/操作包装成 `tea.Cmd`，让耗时操作由 Bubble Tea runtime 调度。

默认按键：

- `tab`：切换容器/镜像视图。
- `up/down` 或 `k/j`：移动光标。
- `r`：刷新当前视图。
- 容器视图：`s` start/stop、`R` restart、`l` logs、`e` exec。
- 镜像视图：`S` save、`p` prune。
- `q` 或 `ctrl+c`：退出。

## 命令契约

Root command：

```text
dockeru
```

子命令：

```text
dockeru ps [-f filter]
dockeru images [-f filter]
dockeru start [container] [-f filter]
dockeru stop [container] [-f filter]
dockeru restart [container] [-f filter]
dockeru logs [container] [-f filter] [--tail 1000]
dockeru exec [container] [-f filter] [--shell /bin/bash]
dockeru compose up <service> [-f docker-compose.yml]
dockeru image save [image] [-f filter] [-o file.tar]
dockeru image prune [-y]
```

CLI/TUI 交互策略：

- 主路径是 `dockeru` 进入 Bubble Tea TUI，在 TUI 中选择容器/镜像并执行操作。
- `dockeru --help`、`dockeru --version` 和辅助子命令仍由 Cobra 提供，方便学习 Cobra 和脚本调用。
- 不实现旧式一次性 1/2/3 文本菜单。
- TUI 操作必须只针对本机 Docker，不回退到 K8s 或远程 SSH。

选择行为：

- 如果显式提供 container/image 参数，直接使用该参数。
- 如果没有显式参数，列出匹配候选并让用户输入序号。
- 如果只匹配到一个候选，自动选中。
- filter 匹配容器名称、容器镜像、镜像 repository 或镜像 tag。
- 如果没有候选，返回清晰错误，不回退到 K8s 或远程 SSH。

## Docker 边界

使用 `os/exec` 参数数组调用 Docker，不拼接 shell 字符串。需要交互的命令直接挂接 `os.Stdin`、`os.Stdout`、`os.Stderr`。

列表命令解析 Docker format 输出：

```text
docker ps -a --format {{.ID}}|{{.Image}}|{{.Command}}|{{.CreatedAt}}|{{.Status}}|{{.Ports}}|{{.Names}}
docker images --format {{.Repository}}|{{.Tag}}|{{.ID}}|{{.CreatedAt}}|{{.Size}}
```

运行命令：

```text
docker start <container>
docker stop <container>
docker restart <container>
docker logs --tail <n> -f <container>
docker exec -it <container> <shell>
docker compose -f <compose-file> up -d <service>
docker image save -o <file> <image>
docker image prune
```

## 编译脚本

`cli/dockeru/build.sh` 从 `cli/dockeru` 编译到 `cli/dockeru/bin`。

预期目标：

- 本机平台：`bin/dockeru`
- Darwin amd64/arm64
- Linux amd64/arm64

脚本使用 `go build`，遵循正常 Go module 行为，不嵌入宿主机特定路径。

## README 学习内容

README 需要用 dockeru 示例覆盖这些 Cobra 概念：

- Root command：`Use`、`Short`、`Long`、`Version`、`SilenceUsage`、`SilenceErrors`。
- Subcommands：`root.AddCommand(...)` 和 `image save` 这种分组命令。
- Flags：例如 `--filter`、`--tail`、`--shell`、`--file`、`--output`、`--yes`。
- Args：可选容器/镜像参数，以及必填 compose service 参数。
- Handlers：为什么执行逻辑放在 `RunE`，并通过返回 error 交给 Cobra/入口处理。
- 扩展路径：如何新增一个命令文件，并在 root 或 group command 中注册。
- Bubble Tea：`tea.Model`、`Init`、`Update`、`View`、`tea.Cmd`、按键消息和 Cobra 启动 TUI 的方式。

## 兼容性和风险

- help/build 验证不需要 Docker daemon；TUI 启动后的真实 Docker 查询/操作需要本机安装 Docker 且当前用户有权限访问。
- 新 CLI 优先使用 `docker compose`，不主动兼容旧 `docker-compose`，除非用户后续明确要求。
- `image prune` 有破坏性，必须要求 `-y/--yes` 或交互确认。
- `exec -it` 需要真实 TTY；验证时只跑 help，不执行真实进入容器。
- 新增 Cobra 会修改 `go.mod` / `go.sum`，需要执行 `go mod tidy` 并检查依赖 diff。
