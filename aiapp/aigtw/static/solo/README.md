# AI Solo Web UI

Preact + htm 驱动的零构建 (no-bundler) SPA, 对应 `aisolo` 的 Mode 驱动架构。

## 架构

```
static/solo/
├── index.html              # 入口 + importmap
├── styles/solo.css         # 单一样式文件
├── vendor/                 # marked / highlight.js 离线本地包
└── js/
    ├── main.js             # render(<App/>)
    ├── lib/
    │   ├── deps.js         # Preact + htm 统一再导出, html = htm.bind(h)
    │   └── markdown.js     # marked+hljs 渲染
    ├── api/
    │   ├── client.js       # REST /solo/v1 轻封装 + JWT
    │   └── sse.js          # 基于 fetch+ReadableStream 的 SSE 客户端 (支持自定义 header)
    ├── hooks/
    │   ├── useSSE.js       # 封装 start/stop, 每次 start 关闭上一次
    │   └── useToast.js
    └── components/
        ├── App.js          # 顶层状态机: sessions / messages / interrupt / mode
        ├── SessionList.js
        ├── ModePicker.js
        ├── ChatView.js
        └── interrupt/
            ├── Approval.js       # approval
            ├── SingleSelect.js   # single_select
            ├── MultiSelect.js    # multi_select
            ├── FreeText.js       # free_text
            ├── FormInput.js      # form_input
            └── InfoAck.js        # info_ack
```

## 协议

所有对外接口都是 SSE + JSON NDJSON 流: 每一帧 `data:` 后面是一个完整 JSON 对象
(定义见 `common/einox/protocol/event.go`)。前端 `api/sse.js` 按 `\n\n` 拆帧,
单帧 JSON.parse, 派发给 App 的事件状态机 `applyEvent`。

事件类型:

| type            | 说明                              |
|-----------------|-----------------------------------|
| `turn.start`    | 一轮开始                          |
| `message.start` | 新 assistant 消息开始             |
| `message.delta` | 文本增量                          |
| `message.end`   | 消息结束, 携带完整文本            |
| `tool.call.start` | 工具调用开始                    |
| `tool.call.end`   | 工具调用结束                    |
| `interrupt`     | 中断, 渲染对应的 InterruptPanel   |
| `turn.end`      | 一轮结束                          |
| `error`         | 错误                              |

## 6 种中断 (InterruptPanel)

前端按 `interrupt.kind` 分发到对应子面板, 各自把用户响应打包为
`POST /solo/v1/interrupt/:id/resume` 的 body:

| kind            | 子面板            | Action      | 额外字段                |
|-----------------|-------------------|-------------|-------------------------|
| `approval`      | Approval.js       | approve/deny/cancel | reason          |
| `single_select` | SingleSelect.js   | select/cancel       | selectedIds[1]  |
| `multi_select`  | MultiSelect.js    | select/cancel       | selectedIds[N]  |
| `free_text`     | FreeText.js       | text/cancel         | text            |
| `form_input`    | FormInput.js      | form/cancel         | formValues map  |
| `info_ack`      | InfoAck.js        | ack/cancel          | 无              |

## 运行

确保后端 `aigtw` 已启动且 `static/solo` 目录被托管在 `/solo/`。
浏览器打开 `http://<host>/solo/`, 在右上角粘贴 JWT access token → 保存,
即可在左栏新建会话并开始对话。
