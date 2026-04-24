# 编码规范

> 本项目基于 Go / go-zero 框架开发，AI 相关业务基于字节跳动 eino 框架。

## 技术栈

- 语言：Go
- 框架：go-zero
- AI 框架：字节跳动 eino
- 代码生成：goctl（go-zero CLI）

## 代码生成流程

每个业务服务都提供 `gen.sh` 脚本用于基础代码生成：

- **网关接口**：先修改 `.api` 文件定义接口，再执行 `gen.sh`
- **gRPC 服务**：先修改 `.proto` 文件定义服务，再执行 `gen.sh`

**禁止**：跳过 `gen.sh` 直接手写 Handler/Types 代码。

## 命名约定

### API（网关）

- 请求结构命名：`xxxRequest`
- 响应结构命名：`xxxResponse`
- 请求和响应必须成对出现

### gRPC

- 请求结构命名：`xxxReq`
- 响应结构命名：`xxxRes`
- 请求和响应必须成对出现，例如：`chatReq` + `chatRes`

## 编码规范

- 遵循 Go / go-zero / Google 开发规范
- **禁止 Java 编程风格**（不必要的 getter/setter、过度封装、Builder 模式滥用等）
- 工具类、结构类等流程代码注释清晰，但避免口语化注释
- 工具类函数必须有单元测试
- proto、api 文件必须有清晰的注释对照
- API 转 gRPC 时注释保持一致

## Git 规范

- 提交信息使用中文
- AI 不执行 `git commit`，代码由人工测试后手动提交
