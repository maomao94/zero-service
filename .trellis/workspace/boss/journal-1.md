# Journal - boss (Part 1)

> AI development session journal
> Started: 2026-04-24

---

## Session: 2026-04-24 — Sprint S6: Dock3 全量 gRPC 接口暴露

### 上下文
- 老板在 `需求输入.md` 中要求开发大疆 Dock3 全部功能接口
- 提出三个关键问题：返回数据最佳实践、need_reply 字段含义、gRPC 请求字段完善

### 执行过程

**Phase 1 - Planning (PM)**:
- 读取需求输入 → 需求分析 → Gap 分析（SDK vs proto 对比）
- 识别出 29 个缺失的 gRPC 接口
- 回答老板问题：CommonRes 够用、need_reply 是 DJI 标准协议字段
- 拆解为 B-009 ~ B-013 五个 Backlog 条目
- 归档 S3 到历史归档，规划 Sprint S6

**Phase 2 - Execute (Backend)**:
- S6-01~05: Proto 定义 29 个 RPC + 16 个消息类型
- S6-06: gen.sh 生成代码骨架
- S6-07~11: Logic 层实现全部 29 个接口
  - 远程调试 15 个（大部分仅需 DeviceSn）
  - 相机/云台 6 个（需 proto → SDK 结构体转换）
  - 直播 3 个（需结构体转换）
  - 航线补充 4 个（含断点续飞复杂参数）
  - 属性设置 1 个（JSON → map 解析）

**Phase 3 - Review (QA)**:
- go build ./... ✅
- go mod tidy ✅
- go vet ./... ✅（djigateway 零警告）
- proto 注释完整性 ✅（249 行注释 / 45 个 RPC）
- 命名规范 ✅（全部 xxxReq）
- 禁止模式 ✅（无 Java 风格、无跳过 gen.sh）

**Phase 4 - Retro (PM)**:
- Backlog 状态更新为已完成
- 任务清单 S6 全部标记 ✅
- 变更记录回填完整
- 需求输入处理记录已追加

### 交付物
- djigateway.proto: 从 14 个 RPC → 43 个 RPC
- 新增 29 个 Logic 文件 + 16 个消息类型
- 4 个文档文件更新

### 问题 & 反思
- **流程遗漏**: 初次执行时跳过了 Trellis /start 上下文加载和 spec 规范注入，被老板指出后补齐
- **改进**: 后续 Sprint 必须严格按 Phase 0 → 1 → 2 → 3 → 4 顺序执行，不可跳步

### 下一步
- 后续可考虑：固件升级接口、媒体文件管理、日志拉取等增强功能

---

## Session: 2026-04-24 — 流程自检机制优化（提示词改进）

### 上下文
- 老板指出 AI 执行 Sprint 时跳步骤（Phase 0 未执行、spec 未注入、quality check 漏掉），需要老板干预才能纠正
- 根因：提示词是"描述性"的而非"命令性"的，缺少门禁和自检机制

### 根因分析（4 个结构性问题）

1. **流程是描述性的，不是命令性的**：用大量篇幅描述流程怎么走，但没有 MUST/NEVER 强约束
2. **Phase 之间缺少门禁**：Phase 0 → 1 → 2 → 3 → 4 之间没有硬性前置条件
3. **缺少自检清单输出**：没有要求每个阶段结束时输出自检结果
4. **workflow.md 太弱**：只是参考手册，没有强制执行力

### 改动内容

**SKILL.md（agile-dev-manager）**：
- 新增「零号法则」（5 条 MUST 规则），置于角色定位之后、流程图之前
- 改造流程图为「含门禁 + 退出清单」版本，每个 Phase 增加：
  - 入口门禁：前一个 Phase 退出清单全部 ✅
  - 退出清单：本 Phase 必须完成的检查项
- 新增「自检输出格式」规范

**workflow.md**：
- 将"编码三段式"升级为"门禁版编码三段式"
- 每个阶段增加具体 bash 命令和 MUST 约束
- 明确"违反此规则等同于违反零号法则"

### 变更文件
- `.trae/skills/agile-dev-manager/SKILL.md`
- `.trellis/spec/workflow.md`

### 预期效果
- AI 在每个 Phase 切换时自动输出退出自检，形成可追溯的执行记录
- 老板不再需要手动纠正流程，AI 自主闭环

