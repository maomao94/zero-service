# 本地开发工具安装指南

本文记录 zero-service 项目推荐的 macOS 本地开发工具。目标是让开发者和 AI 编程助手都能稳定完成代码搜索、项目分析、Java/Maven 开发、HTTP/gRPC/WebSocket 调试、DNS 与网络连通性排查。

## 适用环境

- macOS
- Homebrew 安装路径：`/usr/local` 或 `/opt/homebrew`
- Shell：`zsh`
- 项目语言以 Go 为主，同时可能涉及 Java、Maven、Shell、Docker、HTTP、gRPC、WebSocket、MQTT 等后端开发场景

## 一键安装

如果是新机器，建议先安装这组默认工具：

```bash
brew install \
  ripgrep fd jq yq gh tree ast-grep \
  jenv shellcheck shfmt \
  httpie grpcurl \
  wget telnet nmap mtr websocat doggo
```

如果还没有 Java/Maven 环境，再按需安装：

```bash
brew install openjdk@8 openjdk@17 openjdk@21 maven gradle
```

本机当前已经安装并验证过的核心工具包括：

```text
rg fd jq yq gh tree ast-grep
jenv shellcheck shfmt
http https curl wget
dig doggo nslookup host whois
nc telnet nmap mtr traceroute ping
grpcurl websocat
java javac mvn brew
```

## 工具清单

### 代码探索

| 命令 | 用途 |
|------|------|
| `rg` | 高速全文搜索代码，优先用于查函数、配置、错误信息 |
| `fd` | 快速查找文件和目录，比 `find` 更适合日常使用 |
| `tree` | 查看项目目录结构 |
| `ast-grep` | 基于语法树搜索和重构代码，适合跨文件结构化查找 |
| `jq` | JSON 查看、格式化、过滤 |
| `yq` | YAML/TOML/XML/JSON 查看、格式化、过滤 |
| `gh` | GitHub CLI，用于查看 PR、Issue、Actions、远程仓库信息 |

### Java 与 Maven

| 命令 | 用途 |
|------|------|
| `java` / `javac` | Java 运行时和编译器 |
| `mvn` | Maven 构建、测试、依赖分析 |
| `jenv` | 管理多个 JDK，并按全局、项目或当前 shell 切换版本 |

常用检查命令：

```bash
java -version
javac -version
mvn -version
jenv versions
```

常用 Maven 命令：

```bash
mvn test
mvn -DskipTests package
mvn dependency:tree
mvn -q help:effective-pom
```

### Shell 脚本

| 命令 | 用途 |
|------|------|
| `shellcheck` | 检查 Shell 脚本常见问题 |
| `shfmt` | 格式化 Shell 脚本 |

### HTTP、gRPC 与 WebSocket

| 命令 | 用途 |
|------|------|
| `curl` | HTTP 请求和接口调试基础工具 |
| `http` / `https` | HTTPie 命令，输出更适合人工阅读 |
| `grpcurl` | 调试 gRPC 服务，查看服务列表、方法和请求响应 |
| `websocat` | 调试 WebSocket 连接 |

示例：

```bash
http GET :8080/health
grpcurl -plaintext localhost:9000 list
websocat ws://localhost:8080/ws
```

### DNS、地址与网络排查

| 命令 | 用途 |
|------|------|
| `dig` | DNS 查询，系统通常自带 |
| `doggo` | 更现代友好的 DNS 查询工具 |
| `nslookup` / `host` | DNS 查询补充工具 |
| `whois` | 域名和 IP 注册信息查询 |
| `ping` | 基础连通性检查 |
| `traceroute` | 路由追踪 |
| `mtr` | 持续路由追踪，适合排查丢包和网络抖动 |
| `nc` | TCP/UDP 连通性检查 |
| `telnet` | 简单 TCP 端口测试，兼容老系统排查习惯 |
| `nmap` | 端口扫描和服务识别 |
| `wget` | 下载文件、抓取页面，和 `curl` 互补 |

示例：

```bash
doggo example.com
dig example.com A
nc -vz localhost 8080
nmap -Pn -p 80,443 example.com
sudo mtr example.com
```

`mtr` 需要 root 权限，通常使用 `sudo mtr <host>`。

## jenv 最佳实践

本机可能同时存在多个 JDK。推荐统一交给 `jenv` 管理，避免 `java -version` 和 `mvn -version` 使用不同 JDK。

### 初始化 zsh

在 `~/.zshrc` 中加入：

```bash
export PATH="$HOME/.jenv/bin:$PATH"
eval "$(jenv init -)"
```

启用 `JAVA_HOME` 自动导出：

```bash
jenv enable-plugin export
```

### 添加 JDK

示例：

```bash
jenv add /usr/local/opt/openjdk@8
jenv add /usr/local/opt/openjdk@17
jenv add /usr/local/opt/openjdk@21
```

如果 JDK 安装在用户目录，也可以添加 `Contents/Home` 路径：

```bash
jenv add "$HOME/Library/Java/JavaVirtualMachines/corretto-17.0.14/Contents/Home"
```

### 版本切换

```bash
jenv global 1.8.0.452
jenv local 17
jenv shell 21
```

- `global`：全局默认 JDK。
- `local`：当前项目默认 JDK，会生成 `.java-version`。
- `shell`：只对当前终端会话生效。

切换后验证：

```bash
jenv version
java -version
mvn -version
echo "$JAVA_HOME"
```

## 推荐工作流

1. 进入项目后先用 `rg`、`fd`、`tree` 定位代码和结构。
2. 查看 JSON/YAML 配置时用 `jq`、`yq`，不要手动猜配置层级。
3. 修改 Shell 脚本后运行 `shellcheck` 和 `shfmt`。
4. 调接口优先用 `http` 或 `curl`，调 gRPC 用 `grpcurl`，调 WebSocket 用 `websocat`。
5. 排查域名和网络时按 `doggo/dig -> nc/telnet -> traceroute/mtr -> nmap` 的顺序逐层定位。
6. Java 项目必须先确认 `java -version`、`mvn -version`、`JAVA_HOME` 一致。
7. 每个 Java 项目建议用 `jenv local <version>` 固定版本，避免不同开发者或不同终端使用不同 JDK。

## 当前机器状态参考

本次配置后，当前机器已完成以下事项：

- 安装代码探索工具：`rg`、`fd`、`tree`、`ast-grep`、`jq`、`yq`、`gh`
- 安装 Shell 工具：`shellcheck`、`shfmt`
- 安装接口调试工具：`httpie`、`grpcurl`、`websocat`
- 安装网络排查工具：`wget`、`telnet`、`nmap`、`mtr`、`doggo`
- 安装并配置 `jenv`
- 将已有 JDK 8、17、18、23、24 加入 `jenv`
- 将全局默认 Java 设置为 JDK 8，并让 Maven 使用同一个 JDK

建议新开一个终端后再次验证：

```bash
command -v rg fd jq yq gh tree ast-grep jenv shellcheck shfmt http grpcurl wget nmap mtr websocat doggo
java -version
mvn -version
jenv version
```
