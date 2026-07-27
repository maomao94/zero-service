# 代码复用思考指南

## 适用范围

准备新增工具、client、SDK、option、常量、协议转换、store helper 或发现相似代码时读取。

## 搜索顺序

1. 搜索同名/同语义标识和调用：`rg -n '<term>' app aiapp socketapp gtw facade common model`。
2. 检查 `common/` 已有包、目标服务私有 helper/store/SDK 和 `go.mod` 依赖。
3. 比较的不只是函数签名，还包括错误、context、超时、资源关闭、协议所有权和测试。
4. 决定扩展现有能力、保留服务私有、提取窄接口，或最后才创建新公共包。

仓库中的正例包括：HTTP 使用 `common/netx`，MQTT 关联响应使用 `common/mqttx`/`common/antsx`，数据库共性使用 `common/gormx`，空间计算机制在 `common/gisx`，具体围栏存储留在 `app/gis`。

依据：上述公共包、`app/gis/model/fencestore.go` 及其直接调用点。

## 决策标准

- 复用现有：语义相同，现有错误/生命周期满足需求，通过 option 或小扩展即可支持。
- 服务私有：只服务一个领域、绑定具体 pb/model/topic，或复用场景尚未证明。
- 窄接口/回调：只需要公共组件的一个动作，避免注入大型对象和反向依赖。
- 新公共包：至少两个真实调用场景，API 可传输中立描述，所有者、关闭、错误和测试明确。

复制代码有时比错误抽象更安全；但协议常量、身份转换、加解码和高风险并发逻辑不得多处复制。

## 反模式

- 按名字相似复用，忽略完成语义或数据所有权不同。
- 在 `common/tool` 继续加入带领域依赖的 helper。
- 为单个调用点设计复杂 generic/builder/plugin 框架。
- 引入新第三方库前不检查现有依赖与封装。
- 让 option 直接改运行态对象以省一个配置 struct。

## 验证

- 列出新/扩展 API 的所有真实调用方和不适用场景。
- 运行公共包与直接调用方测试，检查依赖方向没有服务 `internal/` 反向导入。
- 公共 client/并发能力测试默认值、取消、错误、关闭和 race；协议转换测试兼容输入输出。
