# 技术设计

页面仍是 `index.html` 单文件，由现有 `go:embed` 提供。客户端状态分为 `loggedOut`、`lobby`、`connecting`、`connected`，视图只按状态切换，不让未连接用户接触会议操作。

登录只保存 Bearer token 并解析 JWT payload 得到默认身份；由于后端没有无参 token 校验接口，首次受保护业务请求承担服务端校验，失败时回到登录页并显示 401 原因。API 封装统一处理 HTTP、响应 code 和网络异常。

会议区采用视频舞台、底部媒体工具栏和侧栏 tab。LiveKit 事件只更新视频 tile、成员和状态；聊天使用已验证的 `publishData` topic，管理操作复用既有 HTTP API。危险操作使用确认框，所有异步按钮采用忙碌状态并在 finally 恢复。
