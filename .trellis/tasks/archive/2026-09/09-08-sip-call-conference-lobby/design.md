# SIP 拨号会议大厅设计

## Boundary

`DialPad` 负责提交 SIP 外呼和消费 `DialSipReply`；根组件的 `enterMeeting` 负责根据会议号获取 LiveKit token 并切换到会议页面。后端 `DialSip` 契约保持不变。

- 独立外呼：请求不带 `meetingNo`，服务创建 S 前缀会议；前端使用响应中的 `meeting.meetingNo` 进入该会议。
- 会议内外呼：请求带当前 `meetingNo`，外呼继续加入当前房间，前端不切换会议。
- 电话控件：大厅只保留紧凑入口，展开后使用短横向表单；移动端自动换行，不产生横向滚动。

## State Flow

```text
idle -> dialing -> dial submitted -> enter returned meeting (independent only)
                         \-> remain in current meeting (meetingNo supplied)
```

拨号提交成功不代表电话已接通；UI 文案应表达“已发起/等待电话加入”，不伪造接通状态。

## Compatibility

- 不修改 `DialSipReq/Res` 或后端 SIP 逻辑。
- 保留供应商加载、拨号错误、Call ID 展示和已有会议拨号入口。
- 不引入通知中心、Socket.IO 或 LiveKit 房间权限改造。
