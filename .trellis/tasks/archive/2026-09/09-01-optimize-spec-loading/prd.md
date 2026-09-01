# 优化 Spec 分层加载，减少小任务上下文

## 背景

当前 `.trellis/spec/backend/` 下31个spec文件共4574行，加上guides共4751行。即使做一个小任务（如修改一个Logic），AI也会加载大量无关上下文，浪费token并降低响应质量。

## 目标

1. 小任务上下文从~500-1000行降到~150-250行
2. 复杂任务仍能按需加载完整spec
3. 不丢失任何已有规则

## 方案

### 新增 `backend/core-rules.md`（~100行）

从各spec提取**非显而易见的、项目特有的**核心规则，覆盖所有任务都需要知道的内容：

- 命名与边界（来自coding-standards.md）
- Go核心规则（来自coding-standards.md）
- 错误处理核心模式（来自error-handling.md）
- go-zero分层契约（来自go-zero-conventions.md）
- GORM核心规则（来自gormx-guidelines.md，去掉GaussDB场景）
- 并发核心规则（来自concurrency-guidelines.md）
- 安全基线（来自coding-standards.md）

### 精简现有spec

各spec删除已在core-rules.md中的通用规则，保留该领域独有的：
- gormx-guidelines.md：保留GaussDB场景、字段所有权场景、完整验证矩阵
- go-zero-conventions.md：保留网关鉴权模式、ClaimMapping、完整反模式列表
- error-handling.md：保留metadata传播、DJI错误体系、日志边界细节
- 其他spec：去重后保留领域特有内容

### 更新 `backend/index.md`

在路由表前增加"任务类型快速路由"：
- 新增API接口 → go-zero-conventions + contract-generation
- 修改数据库操作 → gormx-guidelines
- 修改定时任务 → crontask-guidelines + trigger-guidelines
- 修改错误处理 → error-handling
- 跨层改动 → guides/index.md

### 更新 `trellis-before-dev` skill

- 步骤4：始终读 `core-rules.md`（~100行）
- 步骤5：根据任务类型，只读index.md指向的1-2个spec
- 步骤6：改为"跨层改动时才读guides/index.md"

## 验收标准

1. `core-rules.md` 不超过120行，覆盖所有任务通用的核心规则
2. 现有spec去重后总行数减少20%以上
3. index.md包含任务类型快速路由
4. trellis-before-dev更新为分层加载策略
5. 所有规则无遗漏（core-rules + 原spec完整覆盖）

## 不做

- 不合并spec文件（保持现有领域边界）
- 不删除任何spec文件
- 不改变spec的适用范围
