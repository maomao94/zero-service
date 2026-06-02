# 贡献指南

感谢你对 Zero-Service 项目的关注！

## 开发流程

1. Fork 本仓库
2. 创建特性分支：`git checkout -b feature/your-feature`
3. 提交更改：`git commit -m 'feat: add your feature'`
4. 推送分支：`git push origin feature/your-feature`
5. 创建 Pull Request

## 代码规范

### Go 代码

- 遵循 Go 官方编码规范
- 使用 `gofmt` 格式化代码
- 使用 `go vet` 静态检查
- 命名规范：
  - API 请求/响应：`XxxRequest` / `XxxResponse`
  - gRPC 请求/响应：`XxxReq` / `XxxRes`
  - 驼峰命名，首字母大写导出

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
- `chore`：构建/工具

示例：
```
feat(trigger): add scheduled task support
fix(ieccaller): reconnect logic for unstable network
docs(readme): update quick start guide
```

### 代码生成

- 修改 `.proto` 或 `.api` 文件后，必须执行 `gen.sh`
- 不要手写或修改生成的代码
- 提交前检查生成文件的 diff

## Pull Request 要求

- PR 标题清晰描述变更
- 关联相关 Issue
- 包含变更说明
- 通过 CI 检查
- 至少一个 Reviewer 批准

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
