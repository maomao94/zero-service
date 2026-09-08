# HTTP 网关响应

> 修改 HTTP 网关 Handler 或新增 API 接口时读取。

### 标准响应格式

所有网关接口统一使用 `xhttp.JsonBaseResponseCtx` 返回，响应格式为 go-zero-x 的 `BaseResponse`：

```json
// 成功
{"code": 0, "msg": "ok", "data": {...}}

// 成功（无数据）
{"code": 0, "msg": "ok"}

// 错误
{"code": 5, "msg": "会议不存在"}
```

### Handler 模板

```go
import (
    "net/http"

    "github.com/zeromicro/go-zero/rest/httpx"
    xhttp "github.com/zeromicro/x/http"
)

func XxxHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req types.XxxRequest
        // 解析用 httpx.Parse
        if err := httpx.Parse(r, &req); err != nil {
            xhttp.JsonBaseResponseCtx(r.Context(), w, err)
            return
        }

        l := xxx.NewXxxLogic(r.Context(), svcCtx)
        resp, err := l.Xxx(&req)
        if err != nil {
            xhttp.JsonBaseResponseCtx(r.Context(), w, err)
        } else {
            xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
        }
    }
}
```

### 关键规则

| 规则 | 说明 |
|------|------|
| 解析请求 | 用 `httpx.Parse(r, &req)` |
| 返回成功 | 用 `xhttp.JsonBaseResponseCtx(r.Context(), w, resp)` |
| 返回成功（无数据） | 用 `xhttp.JsonBaseResponseCtx(r.Context(), w, nil)` |
| 返回错误 | 用 `xhttp.JsonBaseResponseCtx(r.Context(), w, err)` |
| 不使用 | `httpx.OkJsonCtx`、`httpx.ErrorCtx`、`httpx.Ok` |

### 为什么不直接用 httpx

- `httpx.OkJsonCtx` 返回原始数据，不包装 `BaseResponse`
- `httpx.ErrorCtx` 依赖 `SetErrorHandlerCtx`，格式不统一
- `xhttp.JsonBaseResponseCtx` 自动处理成功和错误，统一格式

### 错误码来源

`xhttp.JsonBaseResponseCtx` 内部 `wrapBaseResponse` 处理逻辑：

| 输入类型 | Code | Msg |
|---------|------|-----|
| `*errors.CodeMsg` | 自定义 Code | 自定义 Msg |
| `*status.Status` | gRPC 状态码 | gRPC 消息 |
| `error` | -1 | err.Error() |
| 其他 | 0 | "ok" |

依据：`app/livegtw/internal/handler/meeting/`、`common/gtwx/errorhandler.go`。

验证目标 Handler 的请求解析、成功与错误响应，并运行对应网关包测试和 `git diff --check`。
