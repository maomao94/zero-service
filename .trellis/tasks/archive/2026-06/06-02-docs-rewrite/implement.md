# Implement: 文档体系优化

## 执行步骤

### 阶段 1：清理和准备

- [ ] 1.1 删除 `docs/mattpocock-skills-guide.md`
- [ ] 1.2 删除 `docs/ai-solo-smoke-checklist.md`
- [ ] 1.3 创建 `docs/error-codes.md`（从 `code.md` 迁移内容）

### 阶段 2：新增文档

- [ ] 2.1 创建 `docs/README.md`（文档索引页）
- [ ] 2.2 创建 `docs/architecture.md`（架构概览）
- [ ] 2.3 创建 `docs/djicloud.md`（DJI 云平台服务）
- [ ] 2.4 创建 `docs/quick-start.md`（快速开始详细版）
- [ ] 2.5 创建 `docs/development.md`（开发指南，整合 local-development-tools.md）
- [ ] 2.6 创建 `docs/deployment.md`（部署指南）
- [ ] 2.7 创建 `CONTRIBUTING.md`（贡献指南）

### 阶段 3：优化现有文档

- [ ] 3.1 重写 `README.md`（精简到 ~150 行）
- [ ] 3.2 优化 `docs/iec104.md`（统一风格）
- [ ] 3.3 优化 `docs/iec104-protocol.md`（统一风格）
- [ ] 3.4 优化 `docs/trigger.md`（统一风格）
- [ ] 3.5 重命名 `docs/socketiox-documentation.md` → `docs/socketio.md`，优化内容
- [ ] 3.6 优化 `docs/service-ports.md`（统一风格）
- [ ] 3.7 优化 `docs/kml-kmz-guide.md`（统一风格）

### 阶段 4：验证

- [ ] 4.1 检查所有文档内链接有效性
- [ ] 4.2 检查 README 行数（目标 ~150 行）
- [ ] 4.3 检查文档风格一致性

## 各步骤详细说明

### 1.1 删除 mattpocock-skills-guide.md
- 原因：外部工具指南，不属于项目文档
- 操作：`rm docs/mattpocock-skills-guide.md`

### 1.2 删除 ai-solo-smoke-checklist.md
- 原因：QA checklist，非用户文档
- 操作：`rm docs/ai-solo-smoke-checklist.md`

### 1.3 创建 error-codes.md
- 来源：`code.md` 内容
- 调整：增加标题和说明文字

### 2.1 创建 docs/README.md
- 分类索引：用户文档 / 核心服务 / 开发者
- 简短描述每个文档

### 2.2 创建 architecture.md
- 从 README 提取架构图
- 补充模块依赖、数据流、技术选型

### 2.3 创建 djicloud.md
- 从 README 提取 DJI 相关内容
- 补充 RPC 接口列表
- 补充配置说明

### 2.4 创建 quick-start.md
- 环境要求详细说明
- 单服务启动示例
- Docker Compose 启动
- 常见问题

### 2.5 创建 development.md
- 整合 local-development-tools.md 核心内容
- 补充代码生成流程
- 补充模块扩展约定

### 2.6 创建 deployment.md
- Docker 部署
- 集群部署
- 配置管理

### 2.7 创建 CONTRIBUTING.md
- 代码风格
- 提交规范
- Issue/PR 模板

### 3.1 重写 README.md
- 从 434 行精简到 ~150 行
- 保留：特性、快速开始、架构简图、核心服务表、文档导航、技术栈
- 移除：详细服务介绍（移到各服务文档）

### 3.5 重命名 socketiox-documentation.md
- 原因：简化文件名
- 同时优化内容结构

## 验证命令

```bash
# 检查文档行数
wc -l README.md

# 检查链接有效性
grep -r '\[.*\](.*\.md)' docs/ | while read line; do
  file=$(echo $line | grep -o '\[.*\]' | tr -d '[]')
  path=$(echo $line | grep -o '(.*\.md)' | tr -d '()')
  [ -f "$path" ] || echo "Broken: $path"
done

# 检查文件存在
ls -la docs/*.md
```

## 预估工作量

| 阶段 | 文件数 | 预估时间 |
|------|--------|----------|
| 清理 | 3 | 10 分钟 |
| 新增 | 7 | 60 分钟 |
| 优化 | 7 | 40 分钟 |
| 验证 | - | 10 分钟 |
| **总计** | **17** | **~2 小时** |
