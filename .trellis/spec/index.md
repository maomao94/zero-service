# zero-service 编码规范索引

> 本目录下是项目范围的项目规范和编码指南。每层（Layer）各自独立维护。

## Layer

| Layer | Index | 职责范围 |
|-------|-------|----------|
| **Backend** | [`backend/index.md`](./backend/index.md) | Go 服务端：基础规范、通信协议、领域模块、GORM 模型、并发工具、DJI SDK、前端 UI |

## 使用规则

- 开始编码前，先读当前任务涉及的 Layer 的 index.md，确认需要哪些 spec。
- 跨层任务（.proto + Logic + Model + MQTT）优先读 `guides/cross-layer-thinking-guide.md`。
- 修改字段、枚举、常量、配置、Topic、Method 或错误码前，先搜索所有引用。
