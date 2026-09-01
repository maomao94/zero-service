# HTML 测试页 — 技术设计

## 技术栈

- `livekit-client@2.22.1` UMD：`<script src="https://cdn.jsdelivr.net/npm/livekit-client/dist/livekit-client.umd.min.js"></script>`，全局 `LivekitClient`
- 单文件 index.html：内嵌 `<style>` + `<script>`，无构建
- 文件位置：`livegtw/static/index.html`（livegtw 任务 embed 此文件）

## 页面布局（参考 components-js PreJoin/VideoConference）

```
┌────────────────────────────────────────────────────┐
│ Header: 连接配置（URL / 会议号 / 身份 / 名称）+ 创建/加入/离开 │
├──────────────────────────┬─────────────────────────┤
│ 视频区（本地+远端网格）     │ 控制区: 摄像头/麦克风/共享开关  │
│  participants 列表         │ 管理: 结束/踢人/静音       │
├──────────────────────────┼─────────────────────────┤
│ 聊天区（两路径收发）        │ Data: 广播/定向 + RPC echo │
├──────────────────────────┴─────────────────────────┤
│ 事件日志面板（滚动）                                   │
└────────────────────────────────────────────────────┘
```

## 关键实现点（livekit-client v2 API）

```js
const { Room, RoomEvent, Track, createLocalAudioTrack, createLocalVideoTrack } = LivekitClient;

// 拿 token（走 livegtw）
const r = await fetch('/live/v1/meeting/join', {method:'POST', headers:{'Content-Type':'application/json'},
  body: JSON.stringify({meetingNo, identity, name})});
const {token, wsUrl} = (await r.json()).data;

// 连接
const room = new Room({autoSubscribe: true});
await room.connect(wsUrl, token);

// 发布音视频
await room.localParticipant.setCameraEnabled(true);   // 或 publishTrack(await createLocalVideoTrack())
await room.localParticipant.setMicrophoneEnabled(true);

// 事件（v2 枚举）
room.on(RoomEvent.ParticipantConnected, p => {});
room.on(RoomEvent.ParticipantDisconnected, p => {});
room.on(RoomEvent.TrackSubscribed, (track, pub, p) => { const el = track.attach(); grid.appendChild(el); });
room.on(RoomEvent.TrackUnsubscribed, (track) => track.detach());
room.on(RoomEvent.Disconnected, () => {});
room.on(RoomEvent.ConnectionStateChanged, s => {});

// 聊天：原生 ChatMessage 事件
room.on(RoomEvent.ChatMessage, (msg, participant) => {});  // msg: {id, timestamp, message}
// 自定义 topic：DataReceived（topic 参数）
room.on(RoomEvent.DataReceived, (payload, participant, kind, topic) => {
  if (topic === 'chat') { /* UserData 聊天 */ }
  if (topic === 'data') { /* 广播/定向 Data */ }
});

// 发送
await room.localParticipant.publishData(new TextEncoder().encode(text), {reliable:true, topic:'chat'}); // 定向加 destinationIdentities
// 原生聊天：room.localParticipant.publishData(encode(text), {reliable:true, topic:'lk.chat'})？——v2 无 ChatMessage 发送 API？
//   → 用 publishDataPacket? JS v2 无此 API；原生聊天接收走 ChatMessage 事件，发送用 publishData + topic 'lk.chat'？需实现时核对

// RPC
await room.localParticipant.registerRpcMethod('echo', async (data) => data.payload);  // 或 room.registerRpcMethod（文档两种都有，实现时确认）
const resp = await room.localParticipant.performRpc({destinationIdentity, method:'echo', payload:'ping'});

// 屏幕共享
await room.localParticipant.setScreenShareEnabled(true);  // v2: createScreenShareTrack 或 setScreenShareEnabled

// 静音自己
await room.localParticipant.setMicrophoneEnabled(false);  // 本地静音
```

**实现时需核对的 API 点**：
1. `registerRpcMethod` 挂 room 还是 localParticipant（文档 v2 两处都有；源码 main 显示在 LocalParticipant）
2. JS v2 聊天发送 API（ChatMessage 接收事件确认存在；发送可能无专门 API，用 publishData topic）
3. `setScreenShareEnabled` 存在性（v2 中可能改为 `createScreenShareTrack` + publishTrack）
4. `setCameraEnabled/setMicrophoneEnabled` 为 v1/v2 通用

## 管理操作调用（livegtw API）

```js
POST /live/v1/meeting/create   {title, creatorIdentity}
POST /live/v1/meeting/join     {meetingNo, identity, name}
POST /live/v1/meeting/end      {meetingNo}
POST /live/v1/meeting/kick     {meetingNo, identity}
POST /live/v1/meeting/mute     {meetingNo, identity, muted, kind}
POST /live/v1/meeting/sendData {meetingNo, topic, payload(b64), destinations[]}
GET  /live/v1/meeting/participants?meetingNo=
```

## 事件日志

统一 `log(type, msg)` 函数：DOM 追加 `[HH:mm:ss] type: msg`，自动滚动；连接/断开/轨道/聊天/Data/RPC 全部打点。

## 双标签测试指引（页面顶部说明文案）

1. 标签 A：创建会议 → 复制会议号
2. 标签 B：填同一会议号 + 不同身份 → 加入
3. 互见画面后：聊天/Data/RPC/共享/管理操作逐项验证