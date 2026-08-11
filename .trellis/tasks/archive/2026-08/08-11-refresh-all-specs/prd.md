# 全量刷新 Trellis Spec 并补缺 domain spec

## Goal

审视全部现有 `.trellis/spec/` 是否与代码库一致，过时的更新；同时为缺少 domain spec 的 service/common 包新增 spec。

## Scope

### Part A: 刷新现有 Spec（检查是否过时）

| 类别 | Spec 文件 |
|------|----------|
| 基础规范 | directory-structure, coding-standards, quality-guidelines, go-zero-conventions, contract-generation, service-lifecycle, error-handling |
| 公共基础设施 | common-package-design, gormx-guidelines, concurrency-guidelines, messaging-guidelines, crontask-guidelines |
| 领域契约 | trigger-guidelines, isp-guidelines, iec104-guidelines, dji-guidelines, gis-guidelines, realtime-guidelines, ai-guidelines |

### Part B: 新增缺失 Domain Spec

目标 service（共 13 个，部分可合并）：

| Service | 建议合并 |
|---------|---------|
| alarm | alarm-guidelines (含 alarmx) |
| bridgedump/bridgegtw/bridgekafka/bridgemodbus/bridgemqtt | bridge-guidelines (含 mqttx/modbusx) |
| file | file-guidelines (含 filex/ossx) |
| lalhook/lalproxy | lal-guidelines (含 lalx) |
| logdump | logdump-guidelines |
| podengine | podengine-guidelines (含 dockerx/executorx) |

目标 common 包：

| 包 | Spec |
|----|------|
| netx/wsx/socketiox/ssex | networking-guidelines |
| asynqx | 合并到 concurrency-guidelines 或 trigger-guidelines |
| mcpx/einox | 合并到 ai-guidelines |
| mediax | media-guidelines |
| flowx | flow-guidelines |
| nacosx | 合并到 service-lifecycle |

## Child Tasks

| Child | 范围 | 交付物 |
|-------|------|--------|
| refresh-base-specs | Part A 基础规范 + 公共基础设施（~10 文件） | 更新的 spec 文件 |
| refresh-domain-specs | Part A 领域契约（~7 文件） | 更新的 spec 文件 |
| add-bridge-specs | bridge 系列 + alarm + file | 新增 spec 文件 + index 更新 |
| add-remaining-specs | lal, logdump, podengine, networking, media, flow | 新增 spec 文件 + index 更新 |

## Acceptance Criteria

- [ ] 所有现有 spec 文件至少经过一遍审视，引用路径/模式无过时
- [ ] 缺失 domain 的 service 均有对应 spec
- [ ] 无占位符文本
- [ ] index.md 与实际文件一致
- [ ] 每个 spec 至少有具体 source file 引用
