# Build domain glossary CONTEXT.md

## Goal

在根目录建立 CONTEXT.md 领域词汇表，覆盖稳定的服务名、协议名和项目特有关键概念，统一 agent 和后续文档的规范术语。

## Background

父任务 `08-18-organize-docs` 的一部分。当前无领域词汇表，各文档术语不一致（如 `Socket.IO`/`socketio`/`SocketIO` 混用）。CONTEXT.md 是让 agent 与读者使用同一语言的基础。

## Requirements

1. 在根目录创建 `CONTEXT.md`
2. 覆盖服务名（ieccaller、iecstash、streamevent、trigger、djicloud、socketgtw、socketpush、ispagent、ispserver、file、gis、podengine、bridge 系列等）
3. 覆盖协议名（IEC 104、ISP、RRULE、Socket.IO、KML/KMZ、DRC 等）
4. 覆盖关键领域概念（ASDU 合并、CronJob 排除语义、三通道分发、DRC 指令飞行等），每项只保留一至两句定义和权威文档链接
5. 遵循 domain-modeling 的 CONTEXT-FORMAT：纯词汇表，不包含实现细节
6. 排除 AI 服务术语（实验性内容）和通用编程概念

## Acceptance Criteria

- [ ] 根目录 CONTEXT.md 已创建
- [ ] 词条来自稳定候选清单，按服务名、协议名、关键概念分组；不以达到任意数量为验收目标
- [ ] 词条定义准确，与现有文档和源码一致
- [ ] 每个词条有规范写法，并列出 `_Avoid_` 别名；现有文档不在本子任务中批量改名
- [ ] CONTEXT.md 是纯词汇表，无实现细节
- [ ] AI 服务术语未收录

## Out of Scope

- 实现细节、配置说明、架构决策（属于 docs/ 和 ADR）
- AI 服务术语
- 修改现有文档术语（如发现不一致，记录到本任务说明中，由后续任务统一修改或标记）

## Technical Notes

- 素材来源：docs/*.md、service-ports.md、.trellis/spec/backend/*.md、common/ 下的包文档
- 格式遵循 `CONTEXT-FORMAT.md`：标题、Language 分组、紧凑定义和 `_Avoid_` 别名
- 术语冲突（如 socketio 大小写）需在词条中明确规范写法

## Key Decisions

- 覆盖粒度：稳定的服务名 + 协议名 + 项目特有关键概念，不使用数量指标
- 位置：根目录 CONTEXT.md
- 排除 AI 术语

## Open Questions

- 无
