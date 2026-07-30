# 跨层思考指南

## 适用范围

字段、状态、错误、协议、配置、消息或持久化变更跨越两个以上目录/进程时读取。

## 数据流检查

按实际链路从源到终点列出：

1. 契约源：`.proto`、`.api`、协议 typed struct、数据库 schema/model 或外部规范。
2. 入口与校验：Handler/Server、MQTT/Socket/IEC hook、decoder。
3. 业务编排：Logic、scheduler、workflow、AI runner。
4. 边界依赖：Store、SDK、client、producer、文件/OSS。
5. 异步消费者与回执：worker、ReplyPool、callback、状态聚合。
6. 对外呈现：网关 body、事件 payload、日志/trace、`docs/`。

每一跳记录输入/输出类型、身份字段、状态所有者、失败返回和是否跨 goroutine/进程。

依据：`app/trigger`、`app/ispagent`、`app/djicloud`、`socketapp` 和 `facade/streamevent` 的服务链路。

## 重点问题

- 谁创建该字段，谁允许更新，谁只读？完成路径是否会整行覆盖配置路径？
- SessionID、ClientID、TID、task code、patrol ID、exec ID 是否在跨层中保持原义？
- 空值是未知、未发生、终止还是缺省？Go 零值、SQL NULL 和 Proto 缺省值如何转换？
- 返回成功表示本地接受、入队、broker 确认、远端收到、业务完成还是持久化完成？
- 重试、重复消息、迟到回调、旧 lease/session 如何被识别并阻止覆盖新状态？
- 并发保护承诺的是状态不变、通知不发，还是仅阻止外部副作用？补查发生在 CAS 前还是 CAS 后，是否可能留下已变更但未执行的中间状态？
- ORM mixin/plugin 是否已经自动注入租户、软删除或版本条件？先确认普通 model query、`Table(...)`、原生 SQL、`Unscoped()` 的 scope 差异，再决定是否手写过滤条件。
- trace 与业务 correlation ID 是否分别贯穿所有异步分支？
- 同名概念是否其实属于不同状态机？例如 Plan 预展开执行项与 CronJob lease 调度只能共享纯规则编译，不能共享运行状态和 Store 契约。
- 一个时间字段是否在生命周期不同阶段偷偷改变语义？分别列出“未来计划点、在途原计划点、lease 截止点、实际完成点”的字段和写入者。
- 序列化规则是否有多个合法形态？若未上线，优先规定单一 canonical 写入格式，避免生成层、存储层、执行层和展示层各自 fallback。

## 证据等级

- 代码可证：类型、调用、SQL 条件、锁、生成脚本、测试断言。
- 架构推论：从多个路径推导的所有权或意图，写明“当前结构表明”，不要伪装正式 SLA。
- 需环境验证：集群路由、broker 重投、设备行为、故障转移、顺序、吞吐和 Exactly Once。必须通过配置、服务端实现或故障测试确认。

## 反模式

- 只改 producer，不搜索 consumer 和回执。
- 只改 Proto 字段，不执行生成或检查网关/外部客户端。
- 看到唯一 ID 就假设它在所有层含义相同。
- 用单元测试推断跨进程交付保证。
- 用“下游未调用”推断“数据库状态未变化”，忽略 claim/CAS 已提交但补查拦截下游的窗口。
- 因为两个模块都使用 RRULE，就推断它们属于同一调度体系并合并状态机。
- 在一层修改公共接口后，只验证该包，不搜索所有 Store、转换、模型、proto 和 handler 消费字段。

## 验证

- 用 `rg` 搜索字段、方法、事件、topic、错误码和表字段的所有引用。
- 画出或写出至少一条成功、一条业务失败、一条超时/重复数据流。
- 对时间状态机写出 claim 前、claim 后、重试、完成、人工执行五个快照，逐项检查字段所有权。
- 对规则字符串检查 canonical 写入格式、解析入口、数据库列长度、详情原文和描述是否使用同一真值。
- 按各层索引运行目标测试，并检查生成 diff、store 条件和消费者断言。
