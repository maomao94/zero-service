# 实施计划

1. 保持调度解析边界
   - `parseRRuleSet` 保持 `rrule.StrToRRuleSet` 加 DTSTART/RRULE 必需检查的简单包装。
   - 不向 `ValidateRRule`、`NextAfter` 增加 Set 白名单、TZID 或 ROption 自定义校验。
   - `DescribeRRule` 直接使用包内解析结果，不增加描述专用 Set 形状检查。

2. 修正文案
   - 消除 YEARLY 作用域重复措辞。
   - 参考 rrule.js ToText 的星期集合分类，归并“工作日”和“一周七天”等常见 calendar 场景。
   - 保持中文语法直接生成，不引入英文翻译层。
   - 保持现有频率、相位、BYSETPOS、COUNT/UNTIL、时区和日期列表输出稳定。

3. 固化生成端东八区契约
   - Trigger `CompileSchedule` 测试断言 DTSTART/EXDATE 使用 `Asia/Shanghai`。
   - ISP fixed/cycle/interval 生成测试断言 DTSTART 使用 `Asia/Shanghai`。
   - 仅在测试暴露生成缺陷时最小修复生产生成代码。

4. 补全场景矩阵和差分测试
   - 每个 Frequency 和 ROption 字段至少一个正确描述或安全拒绝断言。
   - 对频率相位、YEARLY 作用域和星期集合归并添加精确回归及 occurrence 差分用例。
   - 保留 DST、RDATE/EXDATE、COUNT/UNTIL 和归一化配置测试。

5. 验证与审查
   - `gofmt -w` 改动 Go 文件。
   - `go test ./common/crontask`
   - `go test ./app/trigger/internal/cronjob`
   - `go test ./app/ispagent/internal/crontask`
   - `go test -race ./common/crontask`
   - `go vet ./common/crontask ./app/trigger/internal/cronjob ./app/ispagent/internal/crontask`
   - `git diff --check`
   - 独立 `trellis-check` 审核场景覆盖和兼容性。
