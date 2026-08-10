# Implementation Plan: Persist Topology Device Name and Type

## Steps

1. 在 `DjiDevice`、`DjiDeviceTopo` 的 `GatewaySn` 后增加 `DeviceType`、`DeviceName` 及明确的 GORM 列类型、默认值和注释。
2. 在 `update_topo` 循环中生成规范三元组字符串，并复用 `LookupDeviceTypeName` 完成严格解析和产品名称查询。
3. 将两个字段加入拓扑和主设备的 `FirstOrCreate + Assign` 更新字段；主设备按 `device_sn = sub.SN` 唯一且始终刷新型号和名称，`GatewaySn` 保持 domain 0/1 不覆盖的原逻辑，拓扑仍按 pair 唯一。
4. 在 `DeviceTopoInfo` 追加字段号 8、9；在未发布的 `DeviceInfo` 中插入字段号 4、5 并连续顺延后续字段，执行 `app/djicloud/gen.sh`。
5. 更新 `toDeviceInfo`、`toTopoInfoList` 和 `appendTopoInfo`，让列表与详情返回持久化值。
6. 扩充 hook 测试，覆盖已知产品、未知产品、重复上报更新和软删除恢复。
7. 补转换层测试，断言列表与详情共用的 RPC 字段语义一致；如最小重构可消除重复映射，则仅复用现有 `toTopoInfoList`，不引入新抽象层。

## Validation

- `gofmt -w` 仅格式化本任务修改的 Go 文件。
- `go test ./common/djisdk`
- `go test ./app/djicloud/internal/hooks`
- `go test ./app/djicloud/internal/logic`
- `go test ./app/djicloud/model/gormmodel`
- `go test ./app/djicloud/...`
- `git diff --check`
- `git status --short` 并确认未包含或覆盖既有 `common/djisdk/hms.go` 修改。

## Review Gates

- 生成文件必须来自 `app/djicloud/gen.sh`，且无无关重排。
- `device_type` 与三个原始字段在同一次 Assign 中更新，避免快照不一致。
- 未命中产品名称不能导致事务失败。
- 最终 diff 不修改产品注册表，不增加历史回填，不改变 `DjiDevice` 在线/网关语义。

## Rollback Points

- Proto 生成出现环境噪声时，在继续 Logic 修改前停止并检查生成链路。
- 测试暴露历史行必须立即可查询名称的产品要求时，回到规划阶段评估独立回填方案，不在本任务中临时扫描全表。
