# 全量 proto json_name tag — 执行计划

## 前置检查

- [ ] 确认 `task.py current` 指向本任务
- [ ] 确认工作区无未提交的 proto 改动（`git status` 相关文件干净）

## 步骤

### 1. 编写转换脚本（临时目录，不入库）

脚本逻辑（Python）：
1. 遍历 24 个一方 proto 文件（排除 third_party/）
2. 逐行识别字段声明：`^  <type> <name> = <num>`（可含 repeated/optional/map/oneof），排除 enum 值（`^  [A-Z_]+ =`）与 reserved
3. 计算 `json_name = lowerCamelCase(name)`
4. 字段行无 `json_name` 时插入：
   - 行内已有 `[` → 在 `[` 后插入 `json_name = "...", `
   - 无 `[` → 在 `;` 前插入 ` [json_name = "..."]`
   - 多行 option（`=` 行后无闭合）：在闭合 `]` 前插入
5. 输出 dry-run 统计：每文件新增数 / 已有数 / 总字段数

### 2. 执行转换

- 运行脚本（`--apply`），逐文件确认 diff 合理
- 重点人工抽查：
  - app/trigger/trigger.proto（已有 18 处，验证跳过逻辑）
  - app/bridgemodbus/bridgemodbus.proto（54 处，量大）
  - 含多行 validate 规则的文件（如 app/trigger、app/lalproxy）
  - 一个含 oneof / map 的文件

### 3. 校验

- [ ] 脚本重新扫描：0 个字段缺 json_name；所有 json_name == lowerCamelCase(字段名)
- [ ] 选 3 个服务重生成 Go 代码（`./gen.sh` 或对应 goctl 命令）→ `git diff` 为空
- [ ] `git diff --check`
- [ ] 全文无意外改动（`git diff --stat` 仅 24 个 proto）

### 4. 收尾

- 清理临时脚本
- 若有跨语言契约发现，交由 trellis-update-spec 判断是否写入 .trellis/spec
- 向用户汇报统计与验证结果，提交由用户确认（Phase 3.4）

## 回滚点

- 每完成一个文件即 `git diff` 检查；任何破坏可 `git checkout <file>` 回滚
- 重生成 diff 非空 = 算法有误，回退并修复算法
