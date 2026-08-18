# Missing service docs design

每个选定服务使用 `docs/<service>/README.md` 作为专项入口。bridge 是五个稳定桥接服务的分类总览，不被描述为单一运行服务。

文档只总结职责、依赖、配置入口和关键能力；RPC/API 方法、字段和校验链接到 `.proto`、`.api` 或 typed 源码，不复制完整契约。
