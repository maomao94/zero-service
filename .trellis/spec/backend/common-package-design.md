# 公共包设计规范

## 适用范围

新增 `common/` 包、扩展公共 API、设计 client option、字节/寄存器转换或通用工作流时读取。

## 进入 `common/` 的条件

- 至少有明确的跨服务复用需求，且能力可以脱离具体业务 proto/model 独立描述。
- 包拥有清晰单一职责、稳定输入输出、错误语义和测试；不能只因代码重复几行就抽象。
- 业务策略留在调用服务，公共包提供机制。例如 `common/mqttx` 管理连接与关联响应，具体 topic/payload 由协议或领域包拥有。
- 优先扩展现有 `common/` 包；`common/tool` 是历史混合工具包，不继续堆放领域能力。

依据：`common/netx`、`common/mqttx`、`common/bytex`、`common/gisx` 及调用点。

## API 与构造

- 必需依赖使用构造参数，行为变体用小接口/函数，非必需配置用 function option。
- option 写入 `XxxOptions` 或私有配置结构，由构造函数校验、规范化并生成运行态对象；option 不直接修改连接、锁、缓存或状态机。
- 零值是否可用必须明确；无法提供安全零值时由构造函数返回 error，而不是延迟到首个调用 panic。
- 公开接口只暴露调用方需要的能力。适配具体框架时在服务或 transport 边界包中完成。
- 长期资源必须暴露幂等 `Close`/`Stop` 或由 context 明确管理，文档说明谁负责调用。

依据：`common/netx/client.go`、`common/djisdk/option.go`、`common/antsx/replypool.go`、`common/wsx/config.go`。

## 二进制与转换

- 多字节编码必须显式指定 byte order、宽度、符号和越界行为；复用 `common/bytex` 已有函数。
- 解码前检查长度，错误输入返回 error；不能依靠 slice 越界 panic 作为校验。
- 整数窄化、浮点位模式、寄存器顺序等有损/协议相关转换必须由名称和测试表达语义。
- 泛型转换只消除同构重复；调用方仍负责确认窄化或符号转换是协议允许的。

依据：`common/bytex/` 及其测试、`common/isp/serializer.go`。

## 工作流封装

- 拦截器顺序属于行为：按声明顺序进入、逆序退出；新增埋点或重试前先确认是否改变 attempt/step 语义。
- context 注入的字段用 typed key 或包内 helper，不能由调用方手工复制内部 key。
- wrapper 要保留原 error、取消和 panic 语义；若做 recover，必须转换成可定位错误而非吞掉。

依据：`common/flowx/`、`common/antsx/invoke.go`。

## 反模式

- 公共包导入具体服务的 `internal/` 或生成 pb。
- 为单个服务的临时流程创建通用框架。
- 暴露内部 map/mutex/连接，使调用方可以绕过不变量。
- 对二进制输入不做长度校验，或在多个包重复 endian 转换。

## 验证

- 公共 API 变更运行目标包及所有直接调用方测试。
- option 覆盖默认值、自定义值、nil/非法输入；转换覆盖边界长度、端序和溢出语义。
- 有共享状态或 goroutine 时增加 `go test -race`。
