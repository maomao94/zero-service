# 研究: SRS HTTP API (v6) — /api/v1/ 全量接口、字段、分页、鉴权与 Oryx 代理行为

- **Query**: SRS HTTP API v6（被 Oryx 代理为 /api/v1/ 前缀）的全量接口、字段、分页语义，供 oryxserver 补齐代理
- **Scope**: external（SRS 源码 + 官方文档 + Oryx 源码）
- **Date**: 2026-08-25
- **核查源码版本**: SRS `6.0release`(=v6.0.184 发布线)、SRS `5.0release`、SRS `develop`(v7/v8)、Oryx `ossrs/oryx@main`

## 0. 关键结论（先看这里）

1. **任务描述的「6 个接口 versions/requests/streams/clients/hosts/summary」与 SRS v5/v6 实际不符**：v6 不存在 `/api/v1/hosts`、`/api/v1/summary`；实际是 `/api/v1/summaries`（复数）；`requests` 调试接口实际注册在 **`/api/v1/tests/requests`**，而 `/api/v1/requests` 会命中 `/api/v1/` 前缀导航（返回 urls 列表）。v6 共有 15+ 端点（见 §2）。
2. **分页只有 streams/clients 支持 start/count**；v5/v6.0 的 count 有硬下限 10（`count = srs_max(10, atoi(count))`）：count=0→10、count=1→10、count=-1→10。**没有 count=0/全部、count=-1/全量 语义**。v7.0.103+ 才改为最小 1（issue #4358），并给 streams 增加 `total`。
3. **列表接口不返回 total**（v6.0 streams/clients 均无 total；v7+ 的 streams 才有）。客户端翻页需自累加 start。
4. **全部接口只读（GET）**，除两处写操作：`DELETE /api/v1/clients/{id}`（踢客户端）、`/api/v1/raw?rpc=reload|reload-fetch`（需开启 raw_api，默认 off，未开启返回 code=1061）。
5. **Oryx 代理 /api/v1/ 时只有 /api/v1/versions 免鉴权**，其余 `/api/*` 需要 `Authorization: Bearer {SRS_PLATFORM_SECRET}`（或 `?token=<JWT>`，同一 secret 签名）；Oryx 内部纯透传（path/query 不改写）；SRS 自身 http_api.auth 默认关闭（Oryx 不设 SRS_HTTP_API_AUTH_*），**Bearer 校验发生在 Oryx 层**。
6. 任务提到的 `docker-srs-platform/platform/srs-api.go（proxy_hl/action）` **已不存在**：ossrs/docker-srs-platform 仓库 404（已被 ossrs/oryx 取代），当前代理实现在 `ossrs/oryx/platform/service.go` + `platform/utils.go`（见 §5）。

## 1. 术语修正对照（任务 vs 实际）

| 任务/PRD 说 | 实际 SRS v6 | 说明 |
|---|---|---|
| `/api/v1/hosts` | 不存在 | 机器信息在 `/api/v1/system_proc_stats`、`/api/v1/self_proc_stats`、`/api/v1/summaries`；Oryx 平台侧 `/terraform/v1/host/versions`（非 SRS） |
| `/api/v1/summary` | `/api/v1/summaries` | v5/v6 均复数 |
| `/api/v1/requests` | 注册在 `/api/v1/tests/requests` | `/api/v1/requests` 命中 `/api/v1/` 前缀导航（urls），非请求 dump |
| 分页 count=0 语义 | 无 | count 下限 10（v5/v6.0）；v7.0.103 后下限 1 |
| count=-1 全量 | 无 | `max(10, atoi(-1)) = 10` |

## 2. /api/v1/ 全量端点表（SRS 6.0release）

注册代码：`trunk/src/app/srs_app_server.cpp` `SrsServer::http_handle()` L699-779（v5 同文件 L701-790 基本一致）；mux 路由语义 `trunk/src/protocol/srs_protocol_http_stack.cpp` L832-890（最长前缀匹配；pattern 以 `/` 结尾=子路径匹配，否则精确匹配；`/api/v1/streams` 无尾斜杠 302 跳到 `/api/v1/streams/`，L727-748）。

| 端点 | 方法 | Query 参数 | 返回 data | 备注 |
|---|---|---|---|---|
| `/api/v1/` | GET | — | `urls` 导航 | server.cpp L716-717 |
| `/api/v1/versions` | GET | — | `{major,minor,revision,version}` | L708-710；**Oryx 免鉴权**；HTTP server 层另注册一份（http_conn.cpp L495） |
| `/api/v1/summaries` | GET | — | `{ok,now_ms,self{...},system{...}}` | L719-720；实现 srs_app_utility.cpp L1271-1380 |
| `/api/v1/rusages` | GET | — | `{ok,sample_time,ru_utime,ru_stime,ru_maxrss,...}` | L722-723 |
| `/api/v1/self_proc_stats` | GET | — | 进程自身 proc stat | L725-726 |
| `/api/v1/system_proc_stats` | GET | — | 系统全部进程 | L728-729 |
| `/api/v1/meminfos` | GET | — | 系统内存 | L731-732 |
| `/api/v1/authors` | GET | — | license/copyright/authors | L734-735 |
| `/api/v1/features` | GET | — | 功能特性列表 | L737-738 |
| `/api/v1/vhosts/` | GET | **无分页** | `vhosts` 数组（全量） | L740-741 |
| `/api/v1/vhosts/{id}` | GET | — | `vhost` 对象 | 找不到 200+code `ERROR_RTMP_VHOST_NOT_FOUND` |
| `/api/v1/streams/` | GET | `start`,`count` | `streams` 数组 | L743-744；dump 实现 statistic.cpp L608-629 |
| `/api/v1/streams/{id}` | GET | — | `stream` 对象 | 找不到 200+code `ERROR_RTMP_STREAM_NOT_FOUND` |
| `/api/v1/clients/` | GET | `start`,`count` | `clients` 数组 | L746-747；dump 实现 statistic.cpp L631-652 |
| `/api/v1/clients/{id}` | GET / **DELETE** | — | `client` 对象 / `{code:0,...}` | DELETE=踢下线 `client->conn->expire()`，L866-877 |
| `/api/v1/raw` | GET/POST | `rpc=raw\|reload\|reload-fetch` | 配置 JSON / `{}` | L749-750；raw_api.enabled 默认 off → code 1061 |
| `/api/v1/clusters` | GET | — | origin cluster API | L752-753 |
| `/api/v1/tests/requests` | GET | — | 请求 dump（uri/path/METHOD） | L757-759 |
| `/api/v1/tests/errors` | GET | — | `{"code":100}` | L761-762 |
| `/api/v1/tests/redirects` | GET | — | 301 → tests/errors | L765-766 |
| `/api/v1/tcmalloc` | GET | `page=summary\|api` | 仅 gperf 构建 | L776-778 |
| `/metrics`(v5) / `/console/`(v5) | GET | — | v5 注册在 api mux；v6 移到 http server/static | v5/v6 行为差异点 |

**响应包裹（所有端点一致）**：`{"code":0,"server":"<server_id>","service":"<service_id>","pid":"<pid>", ...}`。`server` 标识 SRS 实例（变化=重启，客户端应作废缓存）。列表端点字段名为 `streams`/`clients`/`vhosts`，单对象端点 `stream`/`client`/`vhost`，其余 `data`。

**错误响应**：handler 内错误（资源不存在、raw 未启用等）返回 **HTTP 200 + `{"code":<非0>}`**（`srs_api_response_json_code`，http_api.cpp L108-124，未设置 HTTP 状态码）；只有 mux 匹配不到才是真 HTTP 404（http_stack.cpp L824）。错误码：stream 未找到/ client 未找到 / vhost 未找到 / raw 禁用(1061)。

**JSONP**：`?callback=JSON_CALLBACK` 返回 `JSON_CALLBACK({...})`；DELETE 用 `?method=DELETE`（文档声明）。

## 3. 字段结构详解（v6.0release）

### 3.1 GET /api/v1/versions（http_api.cpp L301-321）
`{"code":0,"server":"...","service":"...","pid":"...","data":{"major":6,"minor":0,"revision":184,"version":"6.0.184"}}`

### 3.2 GET /api/v1/summaries（utility.cpp L1271-1380）
```
data: {
  ok, now_ms,
  self: { version, pid, ppid, argv, cwd, mem_kbyte, mem_percent, cpu_percent, srs_uptime },
  system: { cpu_percent, disk_read_KBps, disk_write_KBps, disk_busy_percent,
            mem_ram_kbyte, mem_ram_percent, mem_swap_kbyte, mem_swap_percent,
            cpus, cpus_online, uptime, ilde_time,        # ilde_time 为源码拼写，原样透传
            load_1m, load_5m, load_15m,
            net_sample_time, net_recv_bytes, net_send_bytes, net_recvi_bytes, net_sendi_bytes,
            srs_sample_time, srs_recv_bytes, srs_send_bytes,
            conn_sys, conn_sys_et, conn_sys_tw, conn_sys_udp, conn_srs }
}
```

### 3.3 GET /api/v1/streams/（列表）与 /api/v1/streams/{id}（单条）
handler http_api.cpp L759-810；字段 dump statistic.cpp SrsStatisticStream::dumps L110-178：
```
streams: [ {
  id, name, vhost, app, tcUrl, url,
  live_ms, clients, frames, send_bytes, recv_bytes,
  kbps: { recv_30s, send_30s },
  publish: { active, cid },          # 未推流: active=false 且无 cid
  video:   { codec, profile, level, width, height } | null,   # HEVC 时 profile/level 对应 hevc
  audio:   { codec, sample_rate, channel, profile } | null
} ]
```
- 单条同字段放 `stream` 对象。
- **无 total**（v6.0）。迭代序=内部 std::map（按 id），非时间序。

### 3.4 GET /api/v1/clients/ 与 /clients/{id}（+DELETE）
handler http_api.cpp L820-883；dump statistic.cpp SrsStatisticClient::dumps L224-250：
```
clients: [ {
  id, vhost, stream, ip, pageUrl, swfUrl, tcUrl, url, name,
  type,          # publish/play/... srs_client_type_string
  publish,       # bool
  alive,         # 秒（float）
  send_bytes, recv_bytes, kbps: { recv_30s, send_30s }
} ]
```
- DELETE 成功：200 + `{"code":0,"server","service","pid"}`（无 data）；client 不存在 → 200+code。
- Oryx 内部即用它踢流（oryx_service.go L1693 `http://127.0.0.1:1985/api/v1/clients/%v`）。

### 3.5 GET /api/v1/vhosts/（列表）/ {id}
dump statistic.cpp L46-77：`{ id, name, enabled, clients, streams, send_bytes, recv_bytes, kbps:{recv_30s,send_30s}, hls:{enabled, fragment} }` — **不支持 start/count，全量返回**。

### 3.6 GET /api/v1/tests/requests（唯一“requests”实现，http_api.cpp L657-692）
```
data: { uri, path, METHOD, headers: { sigature, version, link, time } }
```
⚠️ 源码 bug 标记：L678-684 先 `data->set("headers", 请求头)`，随后 L684 又用 `data->set("headers", server)` 覆盖（本意应为 `data->set("server", server)`），实际 JSON 中 headers=sigature/version/link/time，请求头丢弃。

## 4. 分页语义（重点）

### 4.1 v5 / v6.0（当前 Oryx 背后是 SRS 6.0）
http_api.cpp L786-789（streams）/ L847-850（clients）；v5 对应 L793-798 / L855-860，一致：
```cpp
int start = srs_max(0, atoi(rstart.c_str()));   // 默认&下限 0
int count = srs_max(10, atoi(rcount.c_str()));  // 默认 10，下限 10
```
dumps 遍历（statistic.cpp L608-629）：`for (i=0; i < start+count && it!=end; it++,i++){ if(i<start) continue; ... }`
→ 语义：
- `?start=N&count=M` → 返回第 N..N+M-1 项（0 基）
- 缺省：`start=0, count=10`
- **count=0 / count=1 / count=-1 / 任何 <10 的值 → 一律 10**（无“全量”语义）
- 无法一次拿全量（循环累加 start 多次请求）
- 无 total 字段

### 4.2 v7+（develop / 7.0.103+）
develop 版 http_api.cpp L150-160：
```cpp
start = srs_max(0, atoi(rstart.c_str()));
count = rcount.empty() ? 10 : srs_max(1, atoi(rcount.c_str()));  // 默认10，最小1
```
且 `/api/v1/streams` 列表增加 `total`（总流数，L797-804）。参考 issue#4358（2025-05），“Fixed in v7.0.103”。

### 4.3 hosts/summary/vhosts 分页
vhosts 无分页全量；summaries/rusages/stats/versions/authors/features 单对象无分页。

## 5. Oryx 代理行为（权威：ossrs/oryx@main，platform/service.go + utils.go）

### 5.1 路由与鉴权（service.go L374-394）
```go
// Use versions API as health check API, no auth.
if r.URL.Path == "/api/v1/versions" { proxy1985.ServeHTTP(w, r); return }
// Proxy to SRS HTTP API, for console, by /api/ prefix.
if strings.HasPrefix(r.URL.Path, "/api/") {
    token := r.URL.Query().Get("token")
    apiSecret := envApiSecret()          // = os.Getenv("SRS_PLATFORM_SECRET")，utils.go L366-369
    if err := Authenticate(ctx, apiSecret, token, r.Header); err != nil {
        w.WriteHeader(http.StatusUnauthorized); ohttp.WriteError(...); return
    }
    proxy1985.ServeHTTP(w, r); return
}
```
- **路径改写**：无。`httpCreateProxy` = `httputil.NewSingleHostReverseProxy("http://127.0.0.1:1985")`（utils.go L1459-1485）——path/query 原样转发（`?start=0&count=20` 透传；Authorization 头也透传）。
- **Bearer 要求**：除 `/api/v1/versions` 外，`/api/*` 必须 `Authorization: Bearer <SRS_PLATFORM_SECRET>` 或 `?token=<JWT>`（Authenticate 定义 utils.go L1334-1380：Bearer 匹配 apiSecret；JWT 用 apiSecret 验签）。401 由 Oryx 返回（非 SRS）。
- 响应处理：删除上游 `Server` 与全部 CORS 头（utils.go L1467-1476）。
- **限制**：仅鉴权，无速率限制、无 method 限制（GET/POST/DELETE 均透传）。
- 相关：`/rtc/`(WHIP/WHEP) L342-372、`/terraform/v1/*`(mgmt) 独立处理；路径上 `/api/v1/` 与 Oryx 回调 `http_hooks`（进 SRS 8000/2022 端口路径）不冲突。

### 5.2 SRS 侧鉴权状态
- Oryx 不设置 SRS_HTTP_API_AUTH_* 环境变量；SRS http_api.auth 默认 off（文档：5.0.152+/6.0.40+ 支持 Basic auth）。
- 因此：**经 Oryx 代理 → 鉴权入口只有 Oryx Bearer(apiSecret)，SRS 自身无密码**；直连 SRS:1985 默认无鉴权。
- 对 oryxserver：请求统一带 `Authorization: Bearer {SRS_PLATFORM_SECRET}`（design.md 的 `OryxConfig.Secret`），与现有 /terraform/v1/* 模式一致（common/oryxx 统一注入）。仅 /api/v1/versions 可不带，带上无害。

## 6. v5 (5.0release) vs v6 (6.0release) 差异

注册与核心代码几乎完全一致（server.cpp 行号差 3-9；versions/summaries/streams/clients 字段与分页逻辑完全相同；statistic dumps 字段一致）。差异点：
1. `/metrics`、`/console/`：v5 注册在 api mux（server.cpp L777/L784 附近）；v6 由 http static/server 处理。
2. v5 stream video 无 HEVC profile/level 分支（无 SRS_H265 判断），v6 有（statistic_v6.cpp L151-155）。字段名相同。
3. raw api 两版本相同（rpc=raw|reload|reload-fetch）。
4. auth 行为相同（默认 off）。
5. v7+ 变化（未来 Oryx 换 SRS 7/8 才相关）：count 下限 10→1、streams 加 total、导航项 clusters/perf/tcmalloc（v6 源码 L259-280 导航已含，perf/tcmalloc 为条件编译）。

## 7. 参考链接
- SRS HTTP API v6 文档：https://ossrs.net/lts/zh-cn/docs/v6/doc/http-api
- SRS HTTP API v5 文档：https://ossrs.net/lts/en-us/docs/v5/doc/http-api
- SRS 源码（6.0release）：https://github.com/ossrs/srs/blob/6.0release/trunk/src/app/srs_app_http_api.cpp 、srs_app_server.cpp(http_handle) 、srs_app_statistic.cpp 、srs_app_utility.cpp(srs_api_dump_summaries) 、srs_protocol_http_stack.cpp(mux) 、srs_app_http_conn.cpp(server 层 versions)
- SRS issue #4358（count 最小值）：https://github.com/ossrs/srs/issues/4358
- Oryx 源码：https://github.com/ossrs/oryx （platform/service.go、platform/utils.go）
- Oryx 代理说明（官方文档）：https://ossrs.net/lts/en-us/docs/v7/doc/getting-started-oryx （“Oryx also proxy the SRS HTTP API, which prefix with /api/v1/”）
- Oryx 与 SRS 6.0：`ossrs/oryx:5` 内含 SRS 6.0（docs v6 getting-started-oryx）
