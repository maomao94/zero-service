# HTML 测试页 — 执行计划

## 前置

- [ ] livegtw API 契约确定（10 个接口 URL 与字段）
- [ ] 核对 livekit-client v2.22.1 关键 API（RPC 注册位置、聊天发送、屏幕共享）

## 步骤

1. **核对 API**：从 jsDelivr 拉取 livekit-client v2.22.1 的 d.ts 或源码，确认 4 个待核对点（RPC 注册、聊天发送、屏幕共享、publishData 签名）
2. **骨架**：livegtw/static/index.html 单文件：布局 + 连接区 + 视频网格 + 控制区 + 聊天 + Data/RPC + 日志
3. **连接逻辑**：创建/加入（fetch livegtw）→ room.connect → 发布摄像头/麦克风 → 状态管理
4. **事件接线**：ParticipantConnected/Disconnected、TrackSubscribed/Unsubscribed、Disconnected、ConnectionStateChanged、ChatMessage、DataReceived
5. **功能接线**：聊天双向、Data 广播/定向、RPC echo、屏幕共享、管理操作按钮（踢人/静音/结束/列表）
6. **日志与样式**：日志面板 + 布局样式（参考 components-js 风格）
7. **验证**：`python3 -m http.server` 本地打开自测（无 livegtw 时 mock fetch 或直连）；最终 e2e 任务联调

## 验证命令

```bash
# 本地静态预览（API 请求会失败，UI/连接逻辑可看）
python3 -m http.server 8899 --directory livegtw/static
# 真实联调（e2e 任务）
open http://127.0.0.1:11002/test/meeting
```

## 评审门

- [ ] 页面无 console 错误（打开即验证）
- [ ] 六大功能面板齐全
- [ ] 与 livegtw API 字段契约一致
- [ ] 日志面板持续输出

## 回滚点

- 单文件，删 livegtw/static/index.html 即可；不依赖其他文件