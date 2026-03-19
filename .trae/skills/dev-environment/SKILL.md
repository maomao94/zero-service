---
name: dev-environment
description: |
  开发者本机环境信息和常用工作流。包含终端配置、工具链、常用命令、
  部署流程、代码生成流程等。在执行终端命令、部署、代码生成时参考此文档。
---

# 开发环境配置

## 开发者信息

- IDE: GoLand
- Shell: zsh + starship 提示符

## 终端环境 (.zshrc)

### PATH 配置

```
/usr/local/opt/openjdk@8/bin   # Java 8
~/dockerext                     # 自定义 Docker 工具 (dk, caddy)
~/go/bin                        # Go 工具链 (goctl, depu, protoc-gen-* 等)
```

### 已安装的终端增强工具

| 工具 | 用途 | 触发方式 |
|------|------|---------|
| starship | Shell 提示符美化 | 自动加载 |
| zsh-autosuggestions | 命令自动建议 | 输入时自动触发 |
| zsh-syntax-highlighting | 命令语法高亮 | 自动加载 |
| fzf | 模糊搜索 | Ctrl+R 搜索历史 |
| thefuck | 自动修正错误命令 | `fuck` |
| zoxide | 快速跳转目录 | `z <目录名>` |

### 常用别名

```bash
ll    # ls -alF
la    # ls -A
l     # ls -CF
```

### 历史记录配置

- HISTSIZE=10000, SAVEHIST=10000
- 实时写入 (inc_append_history)
- 多终端共享 (share_history)

## 开发工具链

### 已安装工具

| 工具 | 版本 | 用途 |
|------|------|------|
| go | Go 1.25 | Go 编译器 |
| goctl | v1.9.2 | go-zero 代码生成 |
| protoc | v29.3 (libprotoc) | Protocol Buffers 编译器 |
| docker | v29.2.1 | 容器管理 |
| dk | - | 自定义 Docker 管理工具 (Go 编译的二进制) |
| depu | - | Go 依赖更新工具 (Go 编译的二进制) |
| brew | Homebrew | macOS 包管理 |
| sshpass | - | SSH 密码自动输入 (部署用) |

## Git 推送习惯

项目配置了多个远程仓库（通过 `git remote -v` 查看），日常需要保持所有远程仓库同步。

## 常用工作流

### 1. Proto 代码生成

每个服务目录下都有 `gen.sh`，用于从 .proto 生成 Go 代码：

```bash
# 在服务目录下执行
/usr/bin/env bash gen.sh
```

gen.sh 内部执行:
```bash
goctl rpc protoc {service}.proto --go_out=. --go-grpc_out=. --zrpc_out=. --client=false
protoc --proto_path=. --descriptor_set_out=./{service}.pb {service}.proto
```

### 2. 模型代码生成

PostgreSQL 模型生成脚本位于 model/ 目录:
```bash
sh genPgModel.sh postgres <table_name>
```

### 3. 部署流程

#### 3a. 自动化部署 (deploy.sh，用于 test/dev 环境)

```bash
cd app/{service_name}
sh deploy.sh test
```

deploy.sh 流程:
1. 加载 `env/test.env` 环境变量
2. `GOARCH=amd64 GOOS=linux go build` 交叉编译
3. `docker build` 构建镜像
4. `docker save` 导出 tar
5. `sshpass + scp` 上传到远程服务器
6. 远程 `docker load` + 打标签 + 备份旧镜像
7. `docker-compose up -d` 启动服务
8. 清理临时文件

#### 3b. 手动编译部署 (用于生产环境，服务器架构可能不同)

生产环境需要根据目标服务器架构手动编译，步骤如下：

**第一步：交叉编译**

根据目标架构选择 GOARCH：

```bash
# arm64 架构 (如: 华为鲲鹏、国产化服务器)
cd app/{service_name}
GOOS=linux GOARCH=arm64 go build -o app/{service_name} {service_name}.go

# amd64 架构 (常规 x86 服务器)
cd app/{service_name}
GOOS=linux GOARCH=amd64 go build -o app/{service_name} {service_name}.go
```

常见的手动编译组合：
```bash
# 数采平台三件套
GOOS=linux GOARCH=arm64 go build -o app/ieccaller ieccaller.go
GOOS=linux GOARCH=arm64 go build -o app/iecstash iecstash.go
GOOS=linux GOARCH=arm64 go build -o app/trigger trigger.go

# 桥接服务
GOOS=linux GOARCH=arm64 go build -o app/bridgedump bridgedump.go
GOOS=linux GOARCH=arm64 go build -o app/bridgegtw bridgegtw.go
```

**第二步：构建 Docker 镜像 (指定平台)**

```bash
# 使用 buildx 构建指定平台的镜像
docker buildx build --pull=false --platform linux/arm64 -t <registry>/<project>/{service_name}:<version> .
docker buildx build --pull=false --platform linux/amd64 -t <registry>/<project>/{service_name}:<version> .
```

**第三步：导出镜像文件**

使用 `dk` 工具的 image-save 功能导出镜像为 tar 文件：

```bash
dk
# 选择 9 (image-save)
# 输入镜像名称过滤条件
# 选择镜像序号
# 导出为 tar 文件
```

或直接用 docker 命令：
```bash
docker image save -o {service_name}:{version}-{image_id}.tar <registry>/<project>/{service_name}:{version}
```

**第四步：手动上传并加载**

将 tar 文件传输到目标服务器后：
```bash
docker load -i {service_name}:{version}.tar
docker-compose up -d {service_name}
```

### dk 工具功能列表

`dk` 是自定义的 Docker 管理 CLI 工具，支持以下操作：

| 序号 | 功能 | 说明 |
|------|------|------|
| 1 | log | 查看容器日志 |
| 2 | ps | 列出容器状态 |
| 3 | start | 启动容器 |
| 4 | stop | 停止容器 |
| 5 | restart | 重启容器 |
| 6 | up | docker compose 启动 |
| 7 | exec | 进入容器 shell |
| 8 | images | 列出镜像 |
| 9 | image-save | 导出镜像为 tar (支持名称过滤) |
| 10 | image-prune | 清理悬空镜像 |

### 4. 依赖更新

```bash
# 在项目根目录执行
sh update_deps.sh

# 或使用 depu 工具
depu
```

### 5. Go 构建和测试

```bash
# 构建整个项目
go build ./...

# 构建特定服务
go build ./app/trigger/...

# 运行测试
go test -v ./model/...
go test -v ./app/trigger/...

# 运行特定测试
go test ./model -run TestPlanModel_Insert -v

# 整理依赖
go mod tidy
```

## 有 deploy.sh 的服务列表

以下服务支持 `sh deploy.sh test` 部署:

- app/bridgemodbus
- app/bridgemqtt
- app/file
- app/gis
- app/ieccaller
- app/iecstash
- app/logdump
- app/podengine
- app/trigger
- app/xfusionmock
- facade/streamevent
- gtw
- socketapp/socketgtw
- socketapp/socketpush

## 有 gen.sh 的服务列表

以下服务支持 proto 代码生成:

- app/ 下全部 16 个服务
- facade/streamevent
- gtw
- socketapp/socketgtw
- socketapp/socketpush
- third_party (公共 proto)
- zerorpc
