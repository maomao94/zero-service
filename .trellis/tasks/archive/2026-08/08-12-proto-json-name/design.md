# 全量 proto json_name tag — 技术设计

## 目标行为

对每个缺少 `json_name` 的字段，在字段的 option 括号内插入 `json_name = "<lowerCamelCase>"`，与 protoc 默认 json_name 完全一致，因此生成的 Go 代码与 OpenAPI 等下游产物均零变化；价值在于显式契约与跨语言自文档化。

## lowerCamelCase 算法

protoc 的 `ToJsonName` 语义（需严格复刻，避免与默认值不一致导致 codegen diff）：

1. 删除 `_`，其后的字母大写
2. `_` 后跟数字时保留 `_`（如 `data_2nd` → `data_2nd`）
3. 其余字符原样保留，首字母小写（字段名本身以 snake_case 小写开头，天然满足）

已知字段名均为 snake_case 全小写，无需处理首字母大写、连字符等情况。

## 插入规则

- 字段声明形态（单行常见）：
  ```
  string task_code = 3;                          →  string task_code = 3 [json_name = "taskCode"];
  repeated string x = 5 [(validate.rules)...];   →  repeated string x = 5 [json_name = "taskCode", (validate.rules)...];
  ```
- `json_name` 作为 option 列表第一个元素插入；已有 `json_name` 的字段跳过。
- 多行 option（validate.rules 大括号跨行）场景：json_name 插入到 `=` 后第一个 `[` 之后；若字段无 option 括号，则新建 ` [json_name = "..."]`（注意空格与分号位置）。
- oneof 成员与普通字段同规则；map 字段（`map<k,v> name = N`）在 `name` 处插入。
- 处理范围：message 内字段、oneof 内成员；不处理 enum 值、reserved、service/rpc、顶层 option。

## 字段识别（脚本）

逐行扫描每个 proto：

- 用正则识别 `= <数字>` 结尾的字段声明行，过滤 enum 值（值全部大写的 `[A-Z_]+`）与 `reserved` 行
- 多行 option 字段：只处理 `=` 行（该行含字段名），若该行无 `[` 且后续行有 `]`，插入需定位闭合括号
- 注释行（`//`）原样保留，不做任何改动

## 校验方案

1. 脚本 dry-run 输出每文件"新增 N 个 json_name"统计
2. 修改后对含 json_name 的行重新扫描：所有字段行都有 json_name，且值为字段名的 lowerCamelCase
3. 抽查 3 个服务的 `gen.sh` 重生成：`git diff` 为空（zerorpc、app/trigger、facade/streamevent 至少一个含 option 多行场景）
4. `git diff --check` 无空白错误

## 风险与回滚

- 主要风险：插入破坏多行 option 括号配对 → 重生成即报语法错误，易于发现；回滚方式为 `git checkout` 对应 proto
- json_name 值与默认不一致风险 → 用"重生成 diff 为空"作为最终闸门，无漏网
