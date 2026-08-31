# LiveKit 中文学习与对接指南设计

## 文档边界

`docs/livekit-integration-guide.md` 是面向开发者的学习与实操入口，回答“LiveKit 是什么、先学什么、如何验证、如何调用”。`.trellis/spec/backend/livekit-guidelines.md` 是后续写代码时的工程约束，回答“依赖放在哪里、谁拥有状态、哪些语义不能误判、需要怎样验证”。两者互相链接但不复制大段内容。

## 内容结构

学习指南采用渐进路径：

1. 系统地图与术语，区分媒体平面、信令、业务控制面。
2. 自托管最小环境和生产依赖。
3. 生成 token 并让客户端加入房间的最小闭环。
4. 使用 `server-sdk-go/v2` 调用管理 API。
5. Go 后端作为实时参与者连接房间。
6. Webhook 与业务状态同步。
7. Egress、Ingress、SIP、Agent 等专题能力。
8. 映射到 zero-service 的候选边界，但不提前固定业务 API。
9. 学习检查表、常见误区与源码索引。

## 证据与版本策略

- API 签名优先依据本地 `server-sdk-go` 和 `livekit` 源码及其测试。
- 概念和部署要求用官方文档补充，并记录访问日期或稳定 URL。
- 指南不写脱离依赖图的固定 protocol 版本；Server SDK 自带 protocol 依赖，项目只直接 pin 顶层 SDK。
- 版本敏感字段和 enum 在示例编译检查中验证。

## 示例验证

将核心示例抽成临时 Go module，使用本地 `replace github.com/livekit/server-sdk-go/v2 => server-sdk-go` 编译。示例分为“完整程序”和“上下文片段”；只有完整程序要求直接编译，片段必须列出依赖对象。

## 工程契约

- 管理 API 使用 `lksdk.LiveKitAPI`，除 Webhook/token/proto types 外不直接使用底层 Twirp。
- API secret 只在服务端配置；加入 token 由业务服务签发且最小授权、短有效期。
- LiveKit 是实时媒体事实来源，zero-service 是业务会议、用户授权和审计事实来源。
- Webhook 是异步通知，不替代业务授权和主动查询；消费者按 event ID 幂等并容忍重复、迟到和乱序。
- Egress/Ingress/SIP/Agent 是独立能力和部署项，不因为 Server API 存在就视为可用。

## 风险

- 本地仓库可能位于未来开发分支，API 与公开稳定版不完全一致。文档应明确以项目选定依赖版本为准。
- LiveKit 官方能力迭代快，Cloud 与 OSS 行为可能不同；文档必须在相关章节标注适用范围。
- 大量未编译片段容易产生字段漂移；核心流程通过编译测试降低风险，其余以源码链接和版本注记约束。
