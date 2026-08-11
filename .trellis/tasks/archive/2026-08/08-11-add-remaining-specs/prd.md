# 新增剩余 service/common Spec

参考父任务 [prd.md](../08-11-refresh-all-specs/prd.md)。

## Scope

| 新 Spec 文件 | 涵盖范围 |
|-------------|---------|
| lal-guidelines.md | app/lalhook, lalproxy + common/lalx, mediax |
| podengine-guidelines.md | app/podengine + common/dockerx, executorx |
| networking-guidelines.md | common/netx, wsx, socketiox, ssex |
| logdump-guidelines.md | app/logdump |
| flow-guidelines.md | common/flowx |

合并到已有 spec：
- mcpx/einox → 合并到 ai-guidelines.md
- asynqx → 合并到 concurrency-guidelines.md

## Acceptance Criteria

- [ ] 每个新 spec 有真实 source file 引用
- [ ] 合并项已融入对应 spec
- [ ] backend/index.md 已更新
- [ ] 无占位符
