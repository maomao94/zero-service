---
name: "module-deploy"
description: "部署包含 Dockerfile 和 env 目录的模块到指定环境。当用户需要部署一个模块到 test、dev 或其他环境时调用。"
---

# 模块部署技能

## 功能

本技能提供了一种通用的方法，用于部署包含以下条件的模块到指定环境：
- 包含 `Dockerfile` 文件
- 包含 `env` 目录，且目录下有对应环境的配置文件（如 `test.env`）
- 包含 `deploy.sh` 部署脚本

## 适用场景

当用户需要：
- 部署一个模块到 test 环境
- 部署一个模块到 dev 环境
- 部署一个模块到其他自定义环境

## 使用方法

### 前提条件

1. 确保模块目录下存在以下文件和目录：
   - `Dockerfile`：用于构建 Docker 镜像
   - `deploy.sh`：部署脚本
   - `env/` 目录：包含环境配置文件
   - `env/<环境名>.env`：对应环境的配置文件，如 `env/test.env`

2. 确保 `env/<环境名>.env` 配置文件中包含以下必要的环境变量：
   - `REMOTE_USER`：远程服务器用户名
   - `REMOTE_PASSWD`：远程服务器密码
   - `REMOTE_HOST`：远程服务器主机地址
   - `REMOTE_PORT`：远程服务器 SSH 端口
   - `REMOTE_PATH`：远程服务器镜像存储路径
   - `REMOTE_COMPOSE_PATH`：远程服务器 docker-compose 文件路径
   - `IMAGE_NAME`：Docker 镜像名称
   - `SERVICE_NAME`：服务名称

### 部署命令

在模块目录下执行以下命令：

```bash
bash deploy.sh <环境名>
```

例如，部署到 test 环境：

```bash
bash deploy.sh test
```

### 部署流程

1. 加载指定环境的配置文件
2. 编译模块（如果需要）
3. 本地构建 Docker 镜像
4. 将镜像保存为 tar 文件
5. 上传镜像到远程服务器
6. 在远程服务器上加载镜像
7. 备份旧镜像（如果存在）
8. 为新镜像打标签
9. 清理旧备份镜像
10. 启动服务
11. 清理临时文件

## 示例

### 示例 1：部署 podengine 模块到 test 环境

```bash
cd app/podengine
bash deploy.sh test
```

### 示例 2：部署其他模块到 dev 环境

```bash
cd app/other-module
bash deploy.sh dev
```

## 注意事项

1. 确保本地已经安装 Docker 和 Go（如果模块需要编译）
2. 确保远程服务器已经安装 Docker 和 docker-compose
3. 确保本地和远程服务器之间网络通畅
4. 确保远程服务器有足够的磁盘空间存储 Docker 镜像
5. 确保配置文件中的环境变量正确无误

## 故障排查

### 常见错误

1. **环境文件不存在**：确保 `env/<环境名>.env` 文件存在
2. **缺少必要环境变量**：检查配置文件中是否包含所有必要的环境变量
3. **编译失败**：确保本地已经安装 Go，且代码可以正常编译
4. **镜像构建失败**：检查 Dockerfile 是否正确
5. **上传失败**：检查网络连接和远程服务器的 SSH 配置
6. **启动失败**：检查远程服务器的 docker-compose 配置和服务依赖

### 解决方法

1. 检查错误信息，确定具体问题
2. 验证配置文件中的环境变量
3. 验证本地和远程服务器的环境
4. 验证网络连接
5. 查看远程服务器的日志

如果问题仍然无法解决，请联系系统管理员或 DevOps 团队寻求帮助。