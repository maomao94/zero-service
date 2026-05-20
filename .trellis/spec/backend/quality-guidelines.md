# 质量规范

> 质量目标是最小影响、契约清晰、生成流程完整、验证真实执行，而不是扩大重构范围。

## 禁止模式

- 跳过 `gen.sh` 手写 go-zero Handler、Server、Routes、Types、pb 等生成代码。
- 在 Handler/Server 中堆业务编排。
- 套用 Java 风格命名、无意义 getter/setter、过度封装或 Builder 模式。
- 为一次性逻辑创建过度抽象，或在 `common/` 中沉淀尚未证明复用价值的能力。
- 新增功能相近依赖前不检查 `go.mod`、相邻模块和既有封装。
- 无关格式化、无关重构、大范围重排 import 或改动生成文件。
- 忽略错误、硬编码配置、提交真实密钥或在日志中打印敏感信息。

## 必须遵循

- `.api` / `.proto` 是契约源头；变更后执行对应模块 `gen.sh`，再实现 Logic。
- API 请求响应命名使用 `xxxRequest` / `xxxResponse`；gRPC 请求响应命名使用 `xxxReq` / `xxxRes`。
- `.api` / `.proto` 注释必须完整，并与实现行为保持一致。
- 工具函数、复杂协议转换和关键业务分支需要单元测试。
- gRPC Logic 层以业务编排为主，非必要不写单元测试；测试重心放在工具函数、model 方法和复杂协议转换上。
- 先搜索相邻实现和 `common/`，再新增工具函数、SDK、client 或依赖。
- 修改生成代码后必须检查 diff，确认生成结果符合预期。

## 验证策略

- 优先运行与变更相关的包或模块测试，不盲目全量测试。
- 修改依赖时执行 `go mod tidy` 并检查 `go.mod` / `go.sum` diff。
- 修改 `.api` / `.proto` 后执行对应 `gen.sh`，再检查生成代码 diff。
- 跨模块或公共组件变更后扩大到 `go build ./...`、`go test ./...` 或 `go vet ./...`。
- 未执行的验证必须说明原因，例如缺少外部服务、环境变量、数据库或部署凭据。

## 代码审查清单

- [ ] 变更范围是否只覆盖用户需求。
- [ ] 是否遵循 Handler/Server → Logic → Model/SDK 分层。
- [ ] 是否复用已有 model/client/cache/config/common 封装。
- [ ] 是否需要更新 `.api`、`.proto`、Swagger、文档或 SQL。
- [ ] 是否执行了必要生成脚本并检查 diff。
- [ ] 是否存在敏感信息、硬编码配置或本地绝对路径。
- [ ] 是否有必要单测、构建或手工验证结果。
- [ ] 是否需要把稳定规则回填到 `.trellis/spec/**`。
