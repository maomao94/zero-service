# 文档配图

本目录存放 `docs/` 与 `deploy/` 文档中引用的架构图、流程图和数据流图。每张图由同名 `.mmd`（Mermaid 源码）导出 PNG，文档正文只引用 PNG，避免依赖阅读器是否支持 Mermaid。

| 图片 | 用于 |
| --- | --- |
| `doc-map.png` | [docs/README.md](../../README.md) 文档地图 |
| `architecture-layers.png` | [docs/architecture.md](../../architecture.md) 系统分层 |
| `flow-iec104.png` | [docs/architecture.md](../../architecture.md)、[docs/iec104/iec104.md](../../iec104/iec104.md) |
| `flow-dji.png` | [docs/architecture.md](../../architecture.md)、[docs/djicloud/djicloud.md](../../djicloud/djicloud.md) |
| `flow-socketio.png` | [docs/architecture.md](../../architecture.md) 实时通信 |
| `flow-livekit.png` | [docs/architecture.md](../../architecture.md) 视频会议 |
| `flow-oryx.png` | [docs/architecture.md](../../architecture.md)、[docs/oryx/README.md](../../oryx/README.md) |
| `flow-openapi.png` | [docs/architecture.md](../../architecture.md)、[docs/open-api/README.md](../../open-api/README.md) |
| `flow-signature-verify.png` | [docs/open-api/signature.md](../../open-api/signature.md) 请求校验时序 |
| `flow-sip-bridge.png` | [docs/live/sip-integration.md](../../live/sip-integration.md) SIP 桥接架构 |
| `socketio-architecture.png` | [docs/socketio/socketio.md](../../socketio/socketio.md) |
| `trigger-plan-model.png` | [docs/trigger/trigger.md](../../trigger/trigger.md) 三级模型 |
| `trigger-plan-sequence.png` | [docs/trigger/trigger.md](../../trigger/trigger.md) 执行时序 |
| `trigger-execitem-state.png` | [docs/trigger/trigger.md](../../trigger/trigger.md) 状态机 |
| `trigger-cronjob.png` | [docs/trigger/trigger.md](../../trigger/trigger.md) 周期执行 |
| `gateway-request-chain.png` | [docs/gateway-standard.md](../../gateway-standard.md) 调用链 |
| `gateway-error-normalize.png` | [docs/gateway-standard.md](../../gateway-standard.md) 错误码归一化 |
| `deploy-topology.png` | [deploy/README.md](../../../deploy/README.md) 依赖拓扑 |

## 重新生成

先确保本机装有 Google Chrome，然后在本目录执行：

```bash
cat > /tmp/puppeteer.json <<'JSON'
{"executablePath":"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome","args":["--no-sandbox"]}
JSON

npx -y @mermaid-js/mermaid-cli -i doc-map.mmd -o doc-map.png \
  -b white -s 2 -p /tmp/puppeteer.json
```

约定：

- 只改 `.mmd` 后重新导出 PNG，不直接编辑图片。
- 新图沿用「语义命名 + 同目录 `.mmd`」的规则，并在上表登记用途。
- 导出参数保持 `-b white`（白底，深浅主题都可读）与 `-s 2`（2 倍图，缩放后不糊）；不要额外放大画布，宽度过大的图在正文里会被缩小到看不清。
