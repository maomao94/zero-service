# Research: SRS/Oryx 中继拉流与转码能力调研（结论：不满足按需多路转推/转码）

- **Date**: 2026-08-25
- **背景**: 无人机巡检场景：业务侧动态下发「拉取外部流（RTSP/RTMP）转推到 SRS」任务，多路并发、按需启停、后续要按需转码。
- **结论一句**: SRS/Oryx 全家族**无动态中继拉流接口**、**无按流转码接口**；满足需求只能自管 FFmpeg 或引 LAL/ZLM。本任务选定**自管 FFmpeg（ffmpeg-go）**。

## 1. Oryx transcode API（/terraform/v1/ffmpeg/transcode/*）— 全局单任务

源码：`ossrs/oryx/platform/trancode.go`（注意拼写无 s；main = v6 线，release/5.14 = v5 线，实现一致）

**三个接口**：

| 接口 | 作用 |
|---|---|
| `POST /terraform/v1/ffmpeg/transcode/query` | 查全局转码配置（Redis `SRS_TRANSCODE_CONFIG/global`） |
| `POST /terraform/v1/ffmpeg/transcode/apply` | 提交 TranscodeConfig → 写 Redis → 重启任务 |
| `POST /terraform/v1/ffmpeg/transcode/task` | 查任务状态（uuid/pid/input/output/frame log） |

**TranscodeConfig 字段**：`all(v-bool) / vcodec / acodec / vbitrate / abitrate / vprofile / vpreset / achannels / server / secret`

**核心机制**：

```go
// The global transcode task, only support one transcode task.
task *TranscodeTask   // ← 全局唯一

// Run 循环每 300ms：
//  1) !config.All → 跳过
//  2) selectActiveStream()：从 Redis SRS_STREAM_ACTIVE 挑「更新最新」的一条活跃流
//  3) doTranscode()：阻塞至该路结束（流断开 / cancel）→ 再选当前最新
```

**FFmpeg 参数**（固定，为 WebRTC 低延迟定制）：`-re -i rtmp://localhost/<app>/<stream> -vcodec ... -profile:v ... -preset:v ... -tune zerolatency -b:v <k>k -r 25 -g 50 -bf 0 -acodec ... -b:a <k>k [-ac N] [-f flv] rtmp://<server>/<secret>`

**关键限制**：
- ❌ 输入必须是 SRS 本地已发布流（`rtmp://localhost/...`），不能拉外部 RTSP/RTMP 源
- ❌ 只能全局单任务（同一时刻 1 个 FFmpeg），多路不并发
- ❌ 不能按流指定（谁最新转谁），无 queue
- ✅ 输出 `server+secret` 可指向任意 RTMP 服务器（转码后分发）
- 用途定位：单路重点直播（WebRTC 低延迟分发），非多路按需

## 2. SRS Ingest（relay pull）— 配置驱动，无 API

- SRS v4/v5/v6/v8 全部：`vhost { ingest <id> { input{...}; engine{...} } }`，conf 文件 + reload
- v8 文档原文：`ingest id is used in reload or http api management`（“http api management”= reload，非动态 API）
- GitHub issue #2272 官方确认：无动态修改 ingest 接口
- **无 start_ingest / addStreamProxy 类接口**

## 3. SRS Dynamic Forward（relay push）— 方向反向，非转码

- v6 stable / v8 unstable 均有：`forward { backend http://... }`，推流事件（on_forward）时回调 backend，返回 `{code:0,data:{urls:[...]}}`，SRS 把该流**转发出去**
- 是「中继**推流**」（relay push），不是「中继**拉流**」（relay pull）
- 原样转发（无转码）；要转码需另配 conf transcode engine（静态）
- 结论：方向与需求相反（流已在 SRS → 推出去），不能满足

## 4. SRS transcode engine（conf 静态）— 不支持 API 按流启停

- `vhost { transcode { engine 720p {...} } }`，SRS fork ffmpeg 转码后推回自身
- 静态配置 + reload，不能「这路转、那路不转」按任务下发

## 5. 第三方组件（动态 API，可行方向）

| 组件 | 动态拉流 API | 转码 | 备注 |
|---|---|---|---|
| LAL | `POST /api/ctrl/start_relay_pull`（JSON Body）、`GET /api/ctrl/stop_relay_pull?stream_name=` | ❌ 不支持 | 轻量纯中转；项目已有 `app/lalproxy`/`app/lalhook` |
| ZLM (ZLMediaKit) | `POST /index/api/addStreamProxy?vhost=&app=&stream=&url=`、`GET /index/api/delStreamProxy?key=` | ✅ `enable_hls/rtmp/fmp4`、编码参数 | 支持重试 retry_count、auto_close、超时；还有 addStreamPusherProxy 可转推 |

## 6. 项目现状（zero-service）

- SRS Oryx 版本：用户部署 `/api/v1/versions` = **5.0.213**（SRS 5.0 + Oryx v5 线 `release/5.14`）；GitHub srs main = 6.0 stable / 8.0 unstable
- v5 线 transcode 实现与 v6 一致（已逐行核对），调研结论对部署环境同样成立
- 录制合并：SRS 负责 TS 切片（HLS）；**Oryx 负责合并 MP4**（`dvr-local-disk.go:1044-1045`：`ffmpeg -i index.m3u8 -c copy -y index.mp4`，及动态重建 VOD m3u8 `buildVodM3u8ForLocal`）
- 项目已有 `common/mediax`（ffmpeg-go 截图封装）、`common/mqttx`+`common/antsx`（ieccaller 广播/请求应答）、`app/ieccaller` 完整范例
