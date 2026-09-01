# 技术设计：Spec 分层加载

## 架构

```
加载策略：
  始终加载 → core-rules.md (~100行)
  按需加载 → index.md路由 → 1-2个相关spec
  跨层时   → guides/index.md → 1个相关guide
```

## core-rules.md 内容设计

从以下spec提取精华规则，按主题组织：

### 来源映射

| core-rules章节 | 来源spec | 提取内容 |
|---------------|----------|---------|
| 命名与边界 | coding-standards.md | API/gRPC命名、身份ID语义、option模式 |
| Go核心规则 | coding-standards.md | context传播、错误包装、goroutine退出、锁外慢操作 |
| 安全基线 | coding-standards.md | 不泄露敏感信息、日志脱敏 |
| 错误处理 | error-handling.md | 领域错误vs传输边界、errors.Is/As |
| go-zero分层 | go-zero-conventions.md | Handler/Logic/ServiceContext职责、依赖方向 |
| GORM核心 | gormx-guidelines.md | 模型组合、查询写入、RowsAffected、租户 |
| 并发 | concurrency-guidelines.md | goroutine退出策略、共享状态保护 |

### 不纳入core-rules的内容

- 领域特有规则（DJI、ISP、IEC104、GIS等）
- 场景示例（GaussDB空值、字段所有权等）
- 完整反模式列表（只保留最核心的）
- 代码示例（保留在原spec中）

## 现有spec精简策略

每个spec的修改模式：
1. 删除已在core-rules.md中的通用规则段落
2. 保留"适用范围"（用于index路由判断）
3. 保留领域独有的契约、场景和验证
4. 保留完整代码示例（spec的价值在于细节）

## index.md 更新

在现有路由表前增加"任务类型快速路由"表：

```markdown
## 任务类型快速路由

| 我要做什么 | 读这个spec |
|-----------|-----------|
| 新增/修改API接口 | go-zero-conventions + contract-generation |
| 修改数据库操作 | gormx-guidelines |
| 修改定时任务 | crontask-guidelines + trigger-guidelines |
| 修改错误/日志 | error-handling |
| 新增公共包 | common-package-design + coding-standards |
| 修改并发代码 | concurrency-guidelines |
| 跨层/跨服务改动 | guides/index.md → 选择guide |
```

## trellis-before-dev 更新

```markdown
4. **始终读取 core-rules.md**:
   cat .trellis/spec/backend/core-rules.md

5. **按任务类型读取相关spec**:
   - 查 index.md 的"任务类型快速路由"
   - 只读指向的1-2个spec文件
   - 不要读所有spec

6. **跨层改动时才读guides**:
   - 如果改动涉及2个以上目录/进程，读 guides/index.md
   - 否则跳过guides
```
