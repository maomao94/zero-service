# Relay UID 与 Redis Key 设计

## Architecture

`UID` 是 relay 的唯一身份，`Target` 不再承担身份职责。所有生命周期组件只接收 UID 作为 process/map/Redis/queue identity；`RelayState` 另外保存 Host、Port、App、Stream 和 RelayURL，供启动和运维查看。

## Canonical UID

```go
func CanonicalUID(target string) (string, error)
```

输入可以是完整 URL（`rtmp://host:1935/app/stream?query`）、normalized endpoint（`host:1935/app/stream`）或 UID（`host_1935/app/stream`）。输出固定为 `host_port/app/stream`，并验证 app/stream 恰好两段。

解析规则：

- URL 先取 `u.Host` 和 `u.Path`，丢弃 scheme/query。
- normalized endpoint 取第一个 `/` 前的 host，`:` 替换成 `_`。
- UID 形式不再重复替换或追加 host。
- IPv6 等包含多个 `:` 的 host 必须明确处理，不能用简单字符串切分造成歧义；若当前 relay endpoint 不支持该格式则返回 validation error。

## Redis Contracts

```text
state:   oryx:relay:state:{uid}
lease:   oryx:relay:lease:{uid}
lock:    oryx:relay:lock:{uid}
registry: oryx:relay:registry   // ZSET member = uid, score = heartbeat Unix seconds
```

`SaveState` 通过 `stateKey(st.UID)` 写入 JSON；`DeleteState`, `HasLease`, `TryClaim`, `Renew`, `Release`, `Lock`, `AddToRegistry`, `RemoveFromRegistry`, `RenewRegistry` 全部只使用 UID。

## State and Queue Contracts

```go
    UID            string `json:"uid"`
    Host           string `json:"host"`
    Port           int    `json:"port"`
    App            string `json:"app"`
    Stream         string `json:"stream"`
    Source         string `json:"source"`
    RelayURL       string `json:"relay_url"`
    DeadlineAtUnix int64  `json:"deadline_at_unix,omitempty"`
    DeadlineAtStr  string `json:"deadline_at_str,omitempty"`
    CreatedAtStr   string `json:"created_at_str,omitempty"`
    RetryCount     int    `json:"retry_count,omitempty"`
    PendingReconcile bool `json:"pending_reconcile,omitempty"`
}

type ReconcilePayload struct {
    UID        string `json:"uid"`
    Source     string `json:"source,omitempty"`
    RelayURL   string `json:"relay_url,omitempty"`
    RetryCount int    `json:"retry_count,omitempty"`
}

type StopPayload struct {
    UID    string `json:"uid"`
    App    string `json:"app"`
    Stream string `json:"stream"`
}
```

The exact field types must follow the existing config endpoint representation, but no payload may use a second target-derived identity.

## Lifecycle Ordering

```text
Start: parse target -> UID -> lock(UID) -> check lease -> write state(UID) -> claim lease(UID)
       -> start process ID=UID -> ZADD registry member=UID

Progress: Renew lease(UID) -> ZADD registry member=UID

Stop: lock(UID) -> delete registry member/state/lease -> stop process ID=UID

Scanner: stale UID -> lock(UID) -> reread state -> check lease/pending
          -> enqueue payload UID first -> save pending=true
```

Stop cleanup must be attempted independently of whether the local process exists. A lock miss is not equivalent to successful cleanup and must remain observable/retriable.

## Compatibility and Migration

This is a deliberate key format change. Existing keys using MD5 or `rtmp_//` are not read by the new code. Operators must remove those keys from the relay Redis database before validation, or wait for their configured TTL. No dual-read path is planned.
