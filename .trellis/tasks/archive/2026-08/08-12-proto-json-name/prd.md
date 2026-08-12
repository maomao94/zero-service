# 全量 proto 补齐 json_name 驼峰 tag

## Goal

为仓库内全部一方 proto 文件的每个 message / oneof 字段显式声明 `json_name` tag，值为该字段名的 lowerCamelCase（与 protoc 默认一致，与 trigger.proto 现有 18 处写法统一），使跨语言序列化的 JSON 字段名在 proto 源码中显式、自文档化，避免依赖各语言 codegen 对默认值的推导差异。

## Requirements

- 范围：全部 24 个一方 proto 文件（`third_party/` 下依赖不修改）：
  aiapp/aichat、aiapp/aisolo、app/alarm、app/bridgedump、app/bridgekafka、app/bridgemodbus、app/bridgemqtt、app/djicloud、app/file、app/gis、app/iecagent、app/ieccaller、app/iecstash、app/ispagent、app/ispserver、app/lalproxy、app/logdump、app/podengine、app/trigger、app/xfusionmock、facade/streamevent、socketapp/socketgtw、socketapp/socketpush、zerorpc/zerorpc
- 字段命名规范保持不变：proto 字段名维持 snake_case（当前全部文件已是，0 处 camelCase 字段名）
- json_name 统一为 lowerCamelCase（`task_code` → `task_code` 的字段名保持，json_name 写 `taskCode`）
- 已有 `json_name` 的字段不动（当前全部为 camelCase，与约定一致，无 snake_case 多词 json_name 存在）
- 覆盖所有字段形态：普通字段、repeated、map、optional、oneof 成员、嵌套 message
- 不改变 proto 语义与既有行为；显式 json_name 与 protoc 默认值完全一致

## Acceptance Criteria

- [ ] 24 个一方 proto 中，每个字段行均带 `[json_name = "..."]`（或与其它 option 共存于同一 `[...]` 中）
- [ ] 每个 json_name 值等于该字段名的 lowerCamelCase（已有 tag 保持原值但需与约定一致）
- [ ] 各服务 `gen.sh` 重新生成 Go 代码后 `git diff` 无任何变化（证明显式 tag == 默认 json_name，行为零变化）
- [ ] `protoc`/`buf` 编译校验通过（若有 CI 校验方式），`git diff --check` 无空白错误
- [ ] 源码注释、枚举、service 定义、option 块（如 validate.rules）不被破坏

## Notes

- 背景：protoc 对 snake_case 字段默认生成 lowerCamelCase json_name（如 `max_retry` → `maxRetry`），Go 侧 protojson 序列化输出即此名；显式声明用于跨语言（OpenAPI、JS/TS、Python 等）读取 proto 时得到确定且一致的 JSON 字段名。
- 任务方向已与用户确认：字段用下划线、tag 用驼峰，保持统一规范。
