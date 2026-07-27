# Thinking Guides

Guides 用于在动手前补齐分析路径，不直接定义实现规则。得出结论后，必须回到 [Backend Coding Specs](../backend/index.md) 选择对应 Code-Spec。

| Guide | 何时读取 | 需要产出 | 随后读取 |
| --- | --- | --- | --- |
| [跨层思考指南](./cross-layer-thinking-guide.md) | 字段、状态、错误、协议或消息跨两个以上目录/进程 | 契约源、写入者、读取者、消费者、失败与验证链路 | 契约/生成、数据访问、消息及领域规范 |
| [代码复用思考指南](./code-reuse-thinking-guide.md) | 新增工具、client、SDK、常量、转换或公共包 | 复用、扩展、服务私有、窄接口或新公共包的决定 | 目录边界与公共包设计规范 |
| [文档思考指南](./documentation-guide.md) | 修改 Spec、README、`docs/`、协议注释或包文档 | 权威来源、归档位置、去重与同步范围 | 质量规范及受影响主题规范 |

## 使用原则

- Guide 中的问题只用于发现风险，不能替代源码和测试证据。
- 对外可靠性、顺序、重试、故障转移和 Exactly Once 必须由服务端实现、配置或故障测试证明。
- 实验代码没有独立 Guide 或 Spec；只有转为稳定项目能力后才纳入。
