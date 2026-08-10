# 实施计划

## 子任务执行顺序

1. 并行执行并分别检查 `08-10-hms-template-resolution`、`08-10-hms-device-placement`。
2. 两项通过后执行并检查 `08-10-hms-event-persistence`。
3. 持久化字段稳定后执行并检查 `08-10-hms-proto-query`。
4. 父任务执行全量 HMS 数据流、生成 diff 和目标服务集成验证。

以下清单是父任务集成验收视图，具体改动由对应子任务负责。

## 1. 协议与 HMS 解析

- [ ] 为 HMS 回调补充窄事件元数据，传递外层 `tid`、`bid`，保持其他事件 handler API 不变。
- [ ] 修正 `HmsItem.Imminent` 注释，明确即时性与 `level` 严重程度的区别。
- [ ] 校正文案 Key 测试，覆盖机场、飞机地面、飞机空中专用和空中回退，并断言最终 `HmsResolveResult.Key`。
- [ ] 保持 `%component_index` 的通用加一规则，不增加产品数量限制；补充大于当前产品云台数量、缺参和非法类型测试。
- [ ] 保持其他官方占位符的既有规则并补足左右、方向和 alarmid 测试。
- [ ] 校对设备三元组注册表和产品名称测试，保持负载产品名与宿主/挂载位置分离。
- [ ] 将负载位置转换改为显式接收宿主飞机型号，覆盖 M300 RTK 0/1/2、其他机型 0、通用 7 和预留枚举。

## 2. 告警持久化链路

- [ ] 在 `DjiHmsAlert` 新增 `message_key`、`tid`、`bid`、`trace_id`，修正 `imminent` 列注释。
- [ ] 新增 `gimbal_position`；负载 HMS 告警通过当前网关拓扑唯一宿主飞机和 args `gimbal_index` 派生位置。
- [ ] 移除五个 args 索引平铺字段，继续通过 `item_json` 保存完整 args。
- [ ] HMS hook 保存 resolver 实际 Key、事件 `tid`/`bid` 和 context trace ID；日志沿用同一 context 关联字段。
- [ ] 扩展 hook 测试：一个事件含多个 item 时逐条断言关联 ID、message key、message 和 item JSON。

## 3. RPC 契约与查询

- [ ] 按设计重排 `HmsAlertInfo` 声明，删除 reserved 并按语义顺序连续重编号。
- [ ] 新增 `message_key`、`tid`、`bid`、`trace_id` 及准确注释，不新增 `gimbal_position`。
- [ ] 在 `app/djicloud` 目录执行 `./gen.sh`，审查生成 diff，不手改 `*.pb.go`。
- [ ] 更新 `ListHmsAlerts` 映射和测试，返回新增字段并停止返回已删除字段。

## 4. 验证与审查

- [ ] 运行 `gofmt` 格式化修改的手写 Go 文件。
- [ ] 运行 `go test ./common/djisdk`。
- [ ] 运行 `go test ./app/djicloud/internal/hooks`。
- [ ] 运行 `go test ./app/djicloud/...`。
- [ ] 搜索被删除字段的生产代码引用，确认仅允许出现在 Proto reserved、历史任务或原始 `hms.json` args 中。
- [ ] 运行 `git diff --check`，检查生成文件、schema 变化及任务范围。

## 风险与回滚点

- Proto 字段删除是对读取这些字段的调用方的兼容性收缩；实施前以全仓搜索和目标服务编译确认仓库内无其他消费者。
- GORM AutoMigrate 通常不会删除旧列；生产删列必须与应用发布解耦，先发布不再读写旧列的代码，再由部署迁移清理。
- 若事件 handler 签名调整扩散到非 HMS handler，停止并收窄为 HMS 专用事件元数据，不做全 SDK 回调重构。
- HMS item 没有负载 SN；宿主飞机无法从当前网关拓扑唯一确定时禁止选择任一候选飞机。
