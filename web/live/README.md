# Live 会议前端

这是基于 LiveKit `components-js` React 组件体系的独立 Web 工程，提供会议大厅、历史记录、会议房间、实时群聊、邀请票据和主持人管理能力。

## 开发

```bash
npm install
npm run dev
```

Vite 开发服务器默认运行在 `http://localhost:5178`，`/live` 请求代理到本地 `livegtw`（`127.0.0.1:11002`），`/socket.io` 代理到本地 `socketgtw`（`127.0.0.1:11003`），`/livekit` 代理到本地 LiveKit 容器映射端口（`127.0.0.1:7880`）。生产和测试环境应通过 `VITE_API_ROOT`、`VITE_SOCKET_URL`、`VITE_LIVEKIT_URL` 分别配置 HTTP、Socket.IO 和 LiveKit 的 Nginx 对外地址；未指定时使用当前站点的同源反向代理。

使用同源 `/livekit` 时，Nginx 必须把此前缀去掉后转发到 `http://livekit-server:7880`，并透传 WebSocket `Upgrade`/`Connection` 请求头。例如浏览器请求 `/livekit/rtc/v1` 时，LiveKit 应收到 `/rtc/v1`。

## 生产构建

```bash
npm run build
npm run preview
```

房间媒体使用 `@livekit/components-react` 的 `LiveKitRoom`、`useTracks`、`VideoTrack`、`RoomAudioRenderer` 组件；业务接口集中在 `src/lib/api.ts`，对应 `app/livegtw/livegtw.api` 的全部会议接口。
