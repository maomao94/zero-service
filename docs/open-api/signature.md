# 开放平台凭据与请求签名

本文面向需要理解/对接开放平台的开发同学，讲清楚三件事：

1. `appId`、`secret`、`signKey` 分别是什么，为什么要用不同的密码学处理；
2. `app/opengtw/internal/security/secret.go` 里每一段加解密代码在做什么；
3. 开启 `signRequired` 后，HTTP 请求如何签名、服务端如何验签。

相关代码：

| 文件 | 职责 |
| --- | --- |
| `app/opengtw/internal/security/secret.go` | 凭据生成、secret 摘要、signKey 加解密 |
| `app/opengtw/internal/security/signature.go` | 签名数据结构、canonical string、HMAC 计算与校验 |
| `app/opengtw/internal/middleware/openauthmiddleware.go` | HTTP 层：读 body、时间窗、nonce 防重放、调用 security |
| `app/opengtw/internal/logic/app/*.go` | 创建/重置/启停应用，落库与凭据缓存刷新 |

---

## 1. 全貌

开放平台有两种凭据，用途完全不同：

```text
Basic 鉴权（每个请求）
  Authorization: Basic base64(appId:secret)
  └─ 证明“你是谁”

请求签名（可选，signRequired=true）
  X-Sign = v1:HMAC-SHA256(canonicalString, signKey)
  canonicalString = v1 + appId + path + SHA256(body)
  timestamp/nonce 放在 body 里，由 SHA256(body) 覆盖
  └─ 证明“这个请求确实由你发出，且内容没被改过、没有被重放”
```

创建应用时的数据流：

```text
CreateApp
  ├─ appId   = "app_" + UUID v4 hex           → open_app.app_id
  ├─ secret  = "sec_" + 16 字节随机 hex
  │     └─ sha256(secret)                     → open_app_secret.secret_hash
  └─ signKey = "sk_" + 32 字节随机 hex         （仅 signRequired=true）
        └─ EncryptSignKey(主密钥, signKey)     → open_app_secret.sign_key_cipher
```

调用时的数据流：

```text
第三方 ──HTTP──▶ opengtw
                  ├─ Basic 鉴权：sha256(secret) 与缓存里的摘要比对
                  ├─ 若 signRequired：
                  │    ├─ 从 header 读取 X-Sign（v1:<hex>）
                  │    ├─ 读取原始 body，计算 SHA-256 得到 bodyHash
                  │    ├─ 从 body 读取 timestamp/nonce 并校验（±5 分钟）
                  │    ├─ 解密 sign_key_cipher 得到 signKey
                  │    ├─ 重算 HMAC 并与请求 sign 比对
                  │    └─ Redis SETNX 占用 nonce（防重放）
                  └─ 解析 body，转发 app/openapi ──gRPC──▶ podengine / bridgemodbus
```

---

## 2. 标识与密钥格式

| 名称 | 生成方式 | 示例 | 长度 | 服务端存储 |
| --- | --- | --- | --- | --- |
| `appId` | `app_` + UUID v4 的 16 字节 hex | `app_0123456789abcdef0123456789abcdef` | 36 | 明文（主键） |
| `secret` | `sec_` + 16 字节随机 hex | `sec_...` | 36 | **只存 sha256 摘要** |
| `signKey` | `sk_` + 32 字节随机 hex | `sk_...` | 68 | **加密后存密文** |
| 主密钥 | 配置文件 `SignKeyEncryptionKey` | 至少 32 字节 | ≥32 | 不落库，只在配置/环境变量中 |

### 前缀需要带吗？

**需要，原样使用，不要截断或补全。**

- `appId` 在 Basic 用户名、请求体 `appId`、canonical string 中都带 `app_`；
- `secret` 作为 Basic 密码时带 `sec_`，做 sha256 时也带前缀；
- `signKey` 作为 HMAC 的 key 时带 `sk_`，前缀本身就是密钥材料的一部分。

服务端没有 `TrimPrefix` 之类的逻辑，前缀只用于人工识别和日志排查。少一位、多一位都会导致鉴权或签名失败。

---

## 3. `secret.go` 逐段讲解

### 3.1 随机数：为什么必须是 `crypto/rand`

```go
func prefixedHex(prefix string, size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf), nil
}
```

- `crypto/rand` 读取操作系统熵池，结果不可预测；
- `math/rand` 是可预测的伪随机数，绝不能用来生成密钥；
- 生成后转成十六进制，便于放进 URL、日志和数据库。

### 3.2 `appId`

```go
id, _ := uuid.NewRandom()
return appIDPrefix + hex.EncodeToString(id[:]), nil
```

UUID v4 有 122 位随机数（6 位是版本/变体位），转成 32 位 hex 后加 `app_`，共 36 字符。碰撞概率见第 7 节。

### 3.3 `secret`：只存摘要，不存明文

```go
func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
```

客户端每次请求带明文 `secret`，服务端用同样的算法算摘要，与数据库里的 `secret_hash` 比较。

- **为什么不用加密？** 服务端不需要还原 secret，只需要判断“是否一致”。单向哈希更安全，数据库泄露也拿不到明文。
- **为什么不用 bcrypt/argon2？** 那些慢哈希是给“人类口令”抗暴力破解用的；这里的 secret 是 128 位随机串，本身没有弱口令空间，SHA-256 足够。
- **为什么不加盐？** 随机 secret 已经全局唯一；加盐需要额外存储盐，且不改变抗预映像性。

比较时用常数时间：

```go
subtle.ConstantTimeCompare(expected, actual[:])
```

普通 `==` 会在第一个不同字节提前返回，攻击者可能通过响应时间逐字节猜测。常数时间比较始终走完全部字节。

### 3.4 `signKey`：必须可逆，所以要加密

验签时服务端必须拿到 `signKey` 明文，才能重新计算：

```text
HMAC-SHA256(canonicalString, signKey)
```

所以不能像 secret 一样只存哈希。数据库里保存的是密文 `sign_key_cipher`，读取时用配置中的主密钥解密。

### 3.5 主密钥派生：一把主密钥，两把子密钥

```go
encryptionKey = HMAC-SHA256(message="opengtw/sign-key/encryption/v1",     key=masterKey)
macKey        = HMAC-SHA256(message="opengtw/sign-key/authentication/v1", key=masterKey)
```

- 输出是十六进制，再 `hex.DecodeString` 得到 32 字节，正好是 AES-256 和 HMAC-SHA256 的推荐长度；
- 两个 `purpose` 字符串不同，叫 **domain separation**，保证加密密钥和 MAC 密钥互不相关；
- 这是一种简化 KDF（类似 HKDF 的单次 Expand），避免“同一把 key 既加密又做 MAC”带来的组合风险。

> 注意 HMAC 的参数顺序：`cryptor.HmacSha256(str, key)`，即 `str` 是消息，`key` 是密钥。

### 3.6 加密：AES-256-CBC + PKCS#7 + 随机 IV

```go
ciphertext := cryptor.AesCbcEncrypt([]byte(plaintext), encryptionKey)
```

lancet 的 `AesCbcEncrypt` 内部做了三件事：

1. **PKCS#7 填充**：把明文补到 16 字节整数倍；
2. **随机 IV**：每次加密生成 16 字节随机初始向量；
3. **返回 `IV || 密文`**：IV 不需要保密，但要随密文一起保存。

随机 IV 的意义：同样的 `signKey` 每次加密结果都不同，避免攻击者通过密文相等推断“两个应用用了同一个 key”。

### 3.7 防篡改：Encrypt-then-MAC

```go
encoded := base64.RawStdEncoding.EncodeToString(ciphertext)
mac := cryptor.HmacSha256("v1."+encoded, string(macKey))
return "v1." + encoded + "." + mac, nil
```

最终存储格式：

```text
v1.<base64(IV || 密文)>.<hmac-sha256 hex>
 │        │                  │
 │        │                  └─ 对 "v1.<base64>" 的 HMAC，防篡改
 │        └─ RawStdEncoding，无 '=' 填充
 └─ 格式版本，未来换算法时可识别旧数据
```

为什么先 MAC 再解密（Encrypt-then-MAC）：

- 解密前先验证完整性，攻击者无法构造畸形密文去触发 CBC 填充错误；
- 可以避免 padding oracle 一类攻击；
- MAC 覆盖了版本号和密文，版本号也不能被随意改。

### 3.8 解密：先验 MAC，再解密

```go
parts := strings.Split(ciphertext, ".")
if len(parts) != 3 || parts[0] != "v1" { ... }
expectedMAC := cryptor.HmacSha256(parts[0]+"."+parts[1], string(macKey))
if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expectedMAC)) != 1 { ... }
```

顺序不能反。MAC 通过后才做 base64 解码和长度检查：

```go
len(ciphertextBytes) < 2*16 || len(ciphertextBytes)%16 != 0
```

至少要有 1 个 IV 块 + 1 个密文块，且总长度是 16 的整数倍。最后调用 `decryptCBC`，并用 `recover` 把第三方库可能抛出的 panic 转成普通 error，避免脏数据把进程打挂。

### 3.9 为什么是 CBC + HMAC，而不是 GCM

当前实现是成熟的“CBC + Encrypt-then-MAC”组合，正确性依赖两点：

- 每次都使用随机 IV；
- 严格先验 MAC 再解密。

AES-GCM 是 AEAD，接口更简单（自带完整性），未来如果需要可以平滑升级：把存储格式改成 `v2.<base64(nonce||密文||tag)>`，保留 `v1` 解密逻辑兼容旧数据即可。

### 3.10 运维注意事项

- 主密钥长度必须 ≥32 字节；修改主密钥会导致旧 `sign_key_cipher` 无法解密，签名应用会返回 `130105`；
- 主密钥不要写进代码仓库，使用配置/环境变量注入；
- 日志里不要打印 `secret`、`signKey`、主密钥；
- `sign_key_cipher` 是密文，但仍应按敏感字段保护数据库访问权限。

---

## 4. 请求签名协议

### 4.1 请求头与 body

签名头只有一个 `X-Sign`，格式 `v1:<hex>`；`timestamp`、`nonce` 放在 body 里：

```http
POST /v1/gateway
Authorization: Basic base64(appId:secret)
X-Sign: v1:a21688f6f7bd082ead7fa5b1a34eb48899210c7702d808d6413e3197cbea13ba

{
  "method": "pod.list",
  "data": {"limit":100,"offset":0},
  "tid": "req-001",
  "timestamp": 1761120000000,
  "nonce": "nonce-001"
}
```

| 位置 | 字段 | 必填 | 说明 |
| --- | --- | --- | --- |
| header | `X-Sign` | 签名时必填 | `v1:<hex>`；省略 `v1:` 时按 v1 处理，不识别版本返回 `130105` |
| body | `method` | 是 | 开放 method，如 `pod.list` |
| body | `data` | 否 | 该 method 的 protojson，保持 JSON 对象；缺省视为无参数 |
| body | `tid` | 否 | 链路 ID，参与 bodyHash，空串也参与 |
| body | `timestamp` | 签名时必填 | Unix 毫秒时间戳，允许与服务端相差 ±5 分钟 |
| body | `nonce` | 签名时必填 | 1–64 位字母、数字、点、下划线或短横线 |

`data` 保持 JSON 对象，不需要改成字符串；签名覆盖的是整个 body 的原始字节。

### 4.2 canonical string

固定 4 行，用换行分隔：

```text
v1
{appId}
{path}
{bodyHash}
```

说明：

- `appId` 使用 **Basic 鉴权用户名**（服务端可信值）；
- `path` 是请求实际路径，当前固定 `/v1/gateway`；如果以后挂在其他前缀下，签名实现要一起改；
- `bodyHash = SHA256(原始 body 字节)` 的小写十六进制；
- `sign = HMAC-SHA256(canonical, signKey)`，小写十六进制。

body 里的 `method`、`data`、`tid`、`timestamp`、`nonce` 都由 `bodyHash` 覆盖，不再单独进入 canonical；以后 body 新增字段也会自动被签名。

### 4.3 为什么签整个 body

签 `data` 的原始 JSON 字节时，客户端必须从 body 里精确截取 `data` 片段：值两侧空白、pretty-print、嵌套字段顺序都会影响结果，多语言下很容易和服务端算出的 hash 不一致。

改成签整个 body 后：

- 客户端把业务对象（含 timestamp/nonce）序列化一次，得到 `byte[]`；
- 对这段字节算 `bodyHash`，再原样发送这段字节；
- 服务端读取原始 body 算 `bodyHash`，再从中取出 timestamp/nonce 做时间窗和防重放校验。

没有“截取嵌套片段”这一步，也就没有格式歧义。

### 4.4 完整示例

假设：

```text
appId     = app_0123456789abcdef0123456789abcdef
signKey   = sk_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
timestamp = 1761120000000
nonce     = nonce-001
path      = /v1/gateway
body      = {"method":"pod.list","data":{"limit":100,"offset":0},"tid":"req-001","timestamp":1761120000000,"nonce":"nonce-001"}
```

`bodyHash`：

```text
ba0ebcbe2c20ea9befd138f072721e89b2592800080f04e3f04efd52839f2a7d
```

canonical string：

```text
v1
app_0123456789abcdef0123456789abcdef
/v1/gateway
ba0ebcbe2c20ea9befd138f072721e89b2592800080f04e3f04efd52839f2a7d
```

最终签名：

```text
a21688f6f7bd082ead7fa5b1a34eb48899210c7702d808d6413e3197cbea13ba
```

请求：

```http
POST /v1/gateway HTTP/1.1
Authorization: Basic ...
X-Sign: v1:a21688f6f7bd082ead7fa5b1a34eb48899210c7702d808d6413e3197cbea13ba
Content-Type: application/json

{"method":"pod.list","data":{"limit":100,"offset":0},"tid":"req-001","timestamp":1761120000000,"nonce":"nonce-001"}
```

> 示例中的 `signKey` 只用于文档演示，不要在生产使用固定 key；`timestamp` 必须取当前时间，`nonce` 每次请求都要重新生成。

### 4.5 服务端校验顺序

`app/opengtw/internal/middleware/openauthmiddleware.go`：

1. Basic 鉴权得到 `appId`；
2. 检查签名依赖：`SignKeyEncryptionKey` 与 Redis 必须已配置；
3. 解析 `X-Sign`（`v1:<hex>`），不识别版本则拒绝；
4. 读取原始 body，并立刻用 `io.NopCloser(bytes.NewReader(body))` 还原，后续 handler 还要解析；
5. 从 body 解析 `timestamp`、`nonce`，计算 `bodyHash`，组装 `security.SignatureInput`；
6. `Validate`：必填项、nonce 字符集、时间窗 ±5 分钟；
7. 解密当前应用所有有效 `signKey`，逐个重算 HMAC，任一匹配即通过；
8. 用 `SETNX` 占用 `openapi:nonce:{appId}:{nonce}`，TTL 10 分钟（2 倍时间窗）；
9. 全部通过后把 `appId` 写入 context，交给 GatewayHandler 解析并转发。

任何一步失败都返回 `130105 签名无效`，不向客户端区分“过期 / 重放 / 不匹配”，避免给攻击者提供探测信息；真实原因可以查服务端日志。

### 4.6 客户端示例（Go）

```go
func signGateway(appID, signKey string, body []byte) string {
	sum := sha256.Sum256(body)
	canonical := strings.Join([]string{
		"v1", appID, "/v1/gateway", hex.EncodeToString(sum[:]),
	}, "\n")

	mac := hmac.New(sha256.New, []byte(signKey))
	mac.Write([]byte(canonical))
	return "v1:" + hex.EncodeToString(mac.Sum(nil))
}
```

body 需要包含 timestamp/nonce：

```go
body, _ := json.Marshal(map[string]any{
	"method":    "pod.list",
	"data":      data,
	"tid":       tid,
	"timestamp": time.Now().UnixMilli(),
	"nonce":     uuid.NewString(),
})
```

关键：`body` 必须是**实际发送的字节**。用 `json.Marshal` 得到 `body` 后，发送时直接写这段字节，不要再让 HTTP 客户端重新序列化。

Java 示例：

```java
byte[] body = mapper.writeValueAsBytes(request);   // request 里包含 timestamp/nonce
String bodyHash = sha256Hex(body);
String canonical = "v1\n" + appId + "\n" + "/v1/gateway" + "\n" + bodyHash;
String sign = "v1:" + hmacSha256Hex(canonical, signKey);
// 发送时设置 X-Sign: sign，并使用 body 原始字节
```

### 4.7 `signKey` 轮换

- 重置 Basic 密钥时，如果应用开启了签名，会同时生成新的 `signKey`；
- 旧密钥记录保留 24 小时（`status=2`），新旧 key 在这个窗口内都能验签；
- 24 小时后旧 key 失效，客户端必须换成新 key；
- `signKey` 只出现在创建/重置响应中，服务端无法找回，丢失只能重置。

> 注意：`/open-admin/v1/app/disable` 在传入 `signRequired=true` 时会生成新的 `signKey` 并直接替换当前生效的密钥，旧 key 不会保留 24 小时。仅做启停时不要传 `signRequired`；需要平滑轮换时使用 `secret/reset`。

---

## 5. 为什么用 HMAC，而不是 `sha256(signKey + data)`

HMAC 是专门设计的消息认证码：

```text
HMAC-SHA256(key, message) = SHA256((key ⊕ opad) || SHA256((key ⊕ ipad) || message))
```

直接拼接再哈希（`sha256(key || message)`）存在长度扩展攻击等风险，且不是标准构造。**签名统一使用 HMAC-SHA256。**

同理，`secret` 只做身份比对，使用 sha256 摘要即可；`signKey` 用于完整性认证，使用 HMAC。

---

## 6. 常见问题

### Q1：前端/客户端发送 `appId` 时要去掉 `app_` 吗？

不要去。服务端全程按完整字符串处理，`app_` 是 ID 的一部分。`secret`、`signKey` 同理。

### Q2：body 里的 `appId` 必须传吗？

不需要。`appId` 来自 Basic 鉴权用户名，body 只放 `method`、`data`、`tid` 以及签名用的 `timestamp`、`nonce`。

### Q3：`data` 可以保持 JSON 对象吗？

可以。签名覆盖的是整个 body 的原始字节，`data` 不需要改成字符串；客户端只要对实际发送的 body 算 `bodyHash` 即可。

### Q4：为什么 `secret` 存哈希，`signKey` 存密文？

- `secret` 只需要“比对”，不需要还原 → 单向哈希；
- `signKey` 需要参与 HMAC 计算，服务端必须拿到明文 → 可逆加密。

### Q5：签名失败为什么都是 `130105`？

避免向攻击者区分“时间戳过期”“nonce 重放”“key 不匹配”等细节。排查时看服务端日志或本地复算 canonical string。

### Q6：HTTPS 下还需要签名吗？

需要。HTTPS 保护传输过程；签名保护的是“请求内容是否被篡改、是否被重放”，两者解决不同问题。

### Q7：为什么 `timestamp` 用毫秒？

与前端 `Date.now()`、Java `System.currentTimeMillis()` 对齐，避免秒/毫秒单位混淆。签名和校验必须使用同一单位。

---

## 7. `appId` 为什么用 UUID，还需要重试吗

旧实现是 `app_` + 12 位 hex，只有 48 位随机数，按生日问题约 1677 万个 ID 就有较高碰撞概率，所以创建逻辑里做了“查重 + 重试”。

现在改为 UUID v4：

- 122 位随机数，约 `5.3 × 10^36` 种取值；
- 要到约 `2^61`（约 2.3 × 10^18）个 ID 才有 50% 碰撞概率，实际业务不可能达到；
- 因此 **不再需要重试循环**，创建逻辑改为生成一次即可；
- `open_app.app_id` 的主键约束仍是最终兜底：真出现重复会拒绝插入，返回创建失败，不会产生两个相同应用。

这也是当前 `CreateAppLogic` 去掉 `maxCreateRetries` 和 `FindByAppId` 预检查的原因。

---

## 8. 代码索引

| 位置 | 说明 |
| --- | --- |
| `security.NewAppID` / `NewSecret` / `NewSignKey` | 生成三类凭据 |
| `security.HashSecret` / `VerifySecret` | secret 摘要与常数时间校验 |
| `security.EncryptSignKey` / `DecryptSignKey` | signKey 加密存储与解密 |
| `security.deriveCipherKey` | 主密钥派生加密/MAC 子密钥 |
| `security.ParseSignHeader` | 解析 `X-Sign: v1:<hex>` |
| `security.SignatureInput` | 签名要素（appId/path/bodyHash） |
| `security.HashBody` | 原始 body 的 SHA-256 摘要 |
| `security.CanonicalString` | canonical string |
| `security.ComputeSign` / `Verify` | HMAC 计算与多 key 校验 |
| `middleware.OpenAuthMiddleware.verifySignature` | HTTP 层验签、时间窗、nonce |
| `logic/app.CreateAppLogic` | 创建应用、生成/加密 signKey |
| `logic/app.ResetAppSecretLogic` | 重置 secret 与 signKey，旧 key 保留 24h |

协议变更时请同步更新：

- 本文档；
- `app/opengtw/internal/security/signature.go`；
- `web/openapi/src/lib/http.ts`（控制台签名实现）；
- `docs/open-api/README.md` 的请求示例。
