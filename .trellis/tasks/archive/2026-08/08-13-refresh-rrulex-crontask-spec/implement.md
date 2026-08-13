# 实施计划：刷新 rrulex 与 crontask 规格

1. 阅读 rrulex/crontask 源码、测试和 ISP/Trigger 调用方，整理可证明契约与已知限制。
2. 扩充 `common/rrulex/rrulex_test.go`：DTSTART 前/等/后、inc、INTERVAL、所有频率、RDATE/EXDATE、COUNT/UNTIL、安全回退和 DST 日历频率官方差分。
3. 扩充 ISP `task_rule_test.go`：无效窗口 start/end 闭区间、窗口前后、缺失/解析失败。
4. 运行新增测试；只对测试证明的 rrulex 正确性缺陷做最小修复（重点是 WEEKLY/DAILY DST 平移锚点不得越过查询点）。
5. 新建 `backend/rrulex-guidelines.md`，覆盖公开签名、查询语义、平移、描述、错误矩阵、已知限制和测试要求。
6. 精简并刷新 `backend/crontask-guidelines.md`，保留调度职责，引用 rrulex 规范；记录 Enable 与谓词终止边界为后续事项。
7. 更新 `backend/index.md` 的公共基础设施导航。
8. 检查占位符、旧 API、路径/符号引用和 index 一致性。
9. 验证：`go test -count=1 ./common/rrulex ./common/crontask ./app/ispagent/internal/crontask ./app/trigger/internal/cronjob`；`go test -race -count=1 ./common/rrulex ./common/crontask ./app/ispagent/internal/crontask`；`go vet`；`go test ./...`；`git diff --check`。

## 风险与回滚点

- 先提交测试形状，再修改算法；DST 差分仍失败则回退优化、使用原始 Set，不能放宽断言。
- 不改 Enable 数据流或新增通用谓词扫描上限。
- 不修改 rrule-go 依赖版本。
