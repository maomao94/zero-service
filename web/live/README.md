# Live 会议前端

这是基于 LiveKit `components-js` React 组件体系的独立 Web 工程，提供会议大厅、历史记录、会议房间、实时群聊、邀请票据和主持人管理能力。

## 开发

```bash
npm install
npm run dev
```

Vite 开发服务器默认运行在 `http://localhost:5178`，`/live` 请求代理到本地 `livegtw`（`127.0.0.1:11002`）。生产环境可通过 `VITE_API_ROOT` 指定网关前缀，例如 `https://gateway.example.com/live/v1/meeting`。

## 生产构建

```bash
npm run build
npm run preview
```

房间媒体使用 `@livekit/components-react` 的 `LiveKitRoom`、`useTracks`、`VideoTrack`、`RoomAudioRenderer` 组件；业务接口集中在 `src/lib/api.ts`，对应 `app/livegtw/livegtw.api` 的全部会议接口。
