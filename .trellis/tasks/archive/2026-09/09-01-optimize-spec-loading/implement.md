# 执行计划：Spec 分层加载优化

## 执行顺序

### Step 1: 创建 core-rules.md [高优先级]

从以下spec提取核心规则，合并为~100行的精华文件：

1. 读取 coding-standards.md → 提取命名、Go规则、安全基线
2. 读取 error-handling.md → 提取错误处理核心模式
3. 读取 go-zero-conventions.md → 提取分层契约、依赖方向
4. 读取 gormx-guidelines.md → 提取模型组合、查询写入核心规则
5. 读取 concurrency-guidelines.md → 提取并发核心规则
6. 读取 service-lifecycle.md → 提取资源装配核心规则

输出：`backend/core-rules.md`

### Step 2: 精简现有spec [中优先级]

逐个spec删除已在core-rules.md中的内容：

1. coding-standards.md → 删除Go规则、安全基线、命名（保留在core-rules）
2. error-handling.md → 删除核心错误模式（保留在core-rules）
3. go-zero-conventions.md → 删除分层契约基础（保留在core-rules）
4. gormx-guidelines.md → 删除模型组合基础、查询写入基础（保留在core-rules）
5. concurrency-guidelines.md → 删除基础并发规则（保留在core-rules）
6. 其他spec → 检查是否有与core-rules重复的内容

每个spec修改后验证：保留的内容仍然是该领域独有的、有价值的。

### Step 3: 更新 index.md 路由 [中优先级]

在现有路由表前增加"任务类型快速路由"表。

### Step 4: 更新 trellis-before-dev [低优先级]

更新skill的加载步骤：
- 步骤4：始终读 core-rules.md
- 步骤5：按任务类型只读相关spec
- 步骤6：跨层改动时才读guides

### Step 5: 验证 [必须]

1. 检查core-rules.md不超过120行
2. 检查所有规则无遗漏
3. 检查spec去重后总行数减少
4. 模拟小任务加载：core-rules + 1个spec = ~200行

## 回滚

如果发现问题，恢复修改的文件即可。所有修改都是文本文件，无编译依赖。
