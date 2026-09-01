# 执行计划

1. 整体重做页面结构和 CSS，建立登录、大厅、会议三种视图。
2. 接入 token 保存、JWT claim 默认身份、统一 API 错误和 toast。
3. 保留并整理 LiveKit 房间、媒体、聊天、Data、RPC、成员和管理逻辑。
4. 增加响应式布局、空状态、按钮忙碌状态和危险操作确认。
5. 运行 inline JS 解析、`go build ./app/livegtw/...`、`git diff --check`，并检查 go:embed 路径。
