# 刷新 rrulex 与 crontask Trellis 规格

## Goal

基于当前 `common/rrulex`、`common/crontask`、ISP/Trigger 调用方与测试，补齐 RRULE 契约测试并刷新规格。重点验证 DTSTART 相位、INTERVAL 对齐、查询点平移后的 next、官方日历无法表达的业务无效区间，以及无法安全平移时的回退行为；测试暴露的确定性缺陷做最小修复。

## Requirements

- 修改范围限于 `common/rrulex`、必要的 `common/crontask`/ISP 集成测试、`.trellis/spec/` 与本任务工件；不扩散到无关服务。
- 从当前源码与测试提取 `common/rrulex` 的真实契约：完整 Set 解析、配置校验、查询起点平移、批量迭代、谓词过滤和中文描述。
- 将 `common/crontask` 的 Scheduler/Store/lease/执行状态契约与 rrule 工具契约分离，避免同一规范混合两个所有权边界。
- 新增 `.trellis/spec/backend/rrulex-guidelines.md`，并更新 backend index；`crontask-guidelines.md` 只保留调度集成所需的 rrulex 使用约束。
- 记录当前实现中会影响维护和测试设计的边界：DST 日历频率平移、永久规则与永久拒绝谓词、Enable 单次 `Set.After` 的全历史扫描、谓词调用顺序、ISP Enable 与无效区间一致性。
- 补充 `common/rrulex` 差分测试：以未经平移的官方 `rrule.Set` 为参考，覆盖 DTSTART 前/等于/之后查询、`inc` 两种边界、`INTERVAL` 相位、YEARLY/MONTHLY/WEEKLY/DAILY/HOURLY/MINUTELY/SECONDLY、RDATE/EXDATE、COUNT/UNTIL、DST 回拨/前拨和回退规则。
- 补充“开始日期对齐”测试：平移锚点必须位于 DTSTART 的 INTERVAL 相位，不能晚于查询点，且平移前后 next 序列一致。
- 补充“无法安全平移”测试：COUNT、BYWEEKNO、BYYEARDAY、BYEASTER、BYSETPOS、AddDate 钳制或 DST 日历差异必须回退原始 Set，不能猜测日历可达性。
- 区分两类校验：`rrulex.ParseSet`/`Validate` 只校验 Set 结构与语法；永久空日历、任意谓词是否最终接受等一般可达性不能靠无界遍历校验，必须用有界规则/明确业务边界测试或记录为调用约束。
- 补充 ISP 无效区间谓词测试：两端闭区间、窗口缺失/解析失败、窗口后恢复有效；该功能属于 RRULE 日历外的业务校验。
- 新测试若证明当前实现与官方未平移结果不一致，允许对 `rrulex` 做最小正确性修复；不以测试绕过真实缺陷。
- 所有规则必须引用真实源码或测试，不保留模板占位符，不把未验证假设写成已保证行为。

## Acceptance Criteria

- [ ] backend index 包含 `rrulex-guidelines.md`，描述与实际文件集一致。
- [ ] rrulex 规范包含适用范围、公开签名、解析/迭代/错误矩阵、Good/Base/Bad、测试要求和 Wrong/Correct 示例。
- [ ] crontask 规范明确 Scheduler 与 rrulex 的边界，以及首次计算、完成推进、Preview、Enable 的数据流。
- [ ] 当前已知风险被标注为“已知实现限制/待修复”，未被误写成可靠契约。
- [ ] DTSTART/INTERVAL 相位和 next 平移差分测试覆盖全部频率与 DST 日历频率；平移结果不得晚于查询点。
- [ ] `inc=true/false`、count、谓词过滤、RDATE/EXDATE、COUNT/UNTIL 和安全回退均有可观察断言。
- [ ] `ParseSet`/`Validate` 的结构校验与“日历可达性无法通用验证”的边界有测试和规格说明。
- [ ] ISP 无效区间谓词的 start/end 精确边界和窗口外恢复有单测。
- [ ] 新增聚焦测试与现有全仓测试、race、vet 通过；若测试暴露实现缺陷，最小修复通过官方未平移差分验证。
- [ ] 全部源码路径、函数名和验证命令与当前仓库一致。
- [ ] `.trellis/spec/` 无 `TBD`、`TODO: fill`、`placeholder` 等模板占位符。
- [ ] 规格内部不存在旧 API 名称（`InvalidTimeFilter`、`NextRunsFiltered`、`DescribeRRule`、crontask 下的 rrule 工具）。

## Notes

- 不解决任意永久谓词的通用终止判定：该问题一般不可判定；本任务只把它写成调用约束，并用有界规则测试。
- Enable 全历史扫描和 ISP Enable 是否应用无效区间属于调度集成策略，不在本轮 RRULE 算法修复范围；规格记录现状与风险，后续另立实现任务。
- 当前工作树已有本会话的产品代码改动，规格任务不得覆盖或回退它们。
