# 开发指南

## 环境搭建

### macOS 一键安装

```bash
# 核心工具
brew install ripgrep fd jq yq gh tree ast-grep \
  shellcheck shfmt \
  httpie grpcurl \
  wget telnet nmap mtr websocat doggo

# Go 环境
brew install go
```

### 工具说明

| 工具 | 用途 |
|------|------|
| `rg` (ripgrep) | 高速全文搜索代码 |
| `fd` | 快速查找文件和目录 |
| `jq` / `yq` | JSON / YAML 处理 |
| `grpcurl` | 调试 gRPC 服务 |
| `httpie` | HTTP 接口调试 |
| `websocat` | WebSocket 调试 |
| `ast-grep` | 基于语法树搜索和重构代码 |

## 代码生成

### go-zero 服务

```bash
# 进入服务目录
cd app/{service}

# 修改 .proto 或 .api 文件
vim {service}.proto

# 执行代码生成
./gen.sh
```

**重要**：不要手写或随意修改生成的 handler、server、routes、pb.go 文件。

### 数据库模型

```bash
# 通用模型生成
model/genModel.sh

# PostgreSQL 专用
model/genPgModel.sh

# SQL 脚本生成
model/genModelSql.sh
```

## 新增服务流程

1. 在 `app/` 或业务对应目录下创建服务目录
2. 编写 `.proto` 或 `.api` 文件定义服务接口
3. 运行 `gen.sh` 生成代码框架
4. 在 `internal/logic/` 实现业务逻辑
5. 在 `etc/` 下创建配置文件
6. 编写入口 `main` 文件启动服务

## 模块扩展约定

### go-zero 服务

- 先改 `.api`/`.proto`，再执行 `gen.sh`
- 保持 Handler/Server -> Logic -> Model/SDK 的分层

### DJI 云平台

- 新增能力先改 `app/djicloud/djicloud.proto`
- 执行 `app/djicloud/gen.sh`
- 在 `internal/logic/` 和 `common/djisdk/` 补实现
- 业务 Logic 调用 `common/djisdk.Client`，不要直接拼 MQTT Topic

### 公共组件

- 跨服务复用能力放入 `common/`
- 业务有状态逻辑保留在具体服务的 `internal/`

## 调试技巧

### gRPC 调试

```bash
# 列出服务
grpcurl -plaintext localhost:21006 list

# 查看方法详情
grpcurl -plaintext localhost:21006 describe trigger.Trigger.SendTrigger

# 调用方法
grpcurl -plaintext -d '{"taskType":"test","payload":"hello"}' \
  localhost:21006 trigger.Trigger/SendTrigger
```

### HTTP 调试

```bash
# 使用 httpie
http GET localhost:11001/health

# 使用 curl
curl -X POST http://localhost:11001/api/trigger/send \
  -H "Content-Type: application/json" \
  -d '{"taskType":"test","payload":"hello"}'
```

### WebSocket 调试

```bash
# 连接 SocketIO
websocat ws://localhost:11003/socket.io/?EIO=4&transport=websocket
```

## 代码规范

- 命名遵循 Go 规范（驼峰命名、首字母大写导出）
- API 请求/响应：`XxxRequest` / `XxxResponse`
- gRPC 请求/响应：`XxxReq` / `XxxRes`
- 错误码使用 `google.rpc.Code` 标准
- 配置安全：不要提交真实密钥，示例配置仅保留占位值
