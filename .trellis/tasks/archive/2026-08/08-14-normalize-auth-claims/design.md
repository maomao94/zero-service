# Claim 规范化设计

## 边界

本任务只改**进程内 claim 值 → string 的转换语义**（类型白名单 + 精度 + 非法值跳过）。不改变：
- wire key（JWT claim / gRPC metadata / MCP `_meta` 键名与顺序）；
- Authorization 跨服务传播；
- metadata 重复/冲突行为（`enforce-authorization-policy` 处理）；
- `b64:` codec；
- typed-key 结构（前一任务已归档，不复盘）。

## 用户确认的决策

1. **白名单宽松兼容**：`user-id` 接受 `string`、`int`/`int64`/`uint`/`uint64`、整数值 `float64`（≤2^53 精确）、`json.Number`。现有 zerorpc(int64)/socketpush(string) 签发继续可用。
2. **非法值忽略跳过**：bool、数组、对象、分数 float、超大 float 在无错误通道边界（网关 `BridgeJWTClaims`、MCP `_meta` 恢复、claims 映射）**忽略**，视同缺失，不写入 typed key；不新增网关 401 错误通道。
3. **精度**：float64 整数值 >2^53（MCP/socket `tool.ParseToken` 路径已在解析时舍入）**拒绝**；`json.Number`（go-zero 网关路径）天然精确，保留字面量。

## 统一转换函数（authctx）

新增包内核心转换，替代纯 lancet 宽松路径：

```go
// normalizeClaimString converts a claim value to a safe string form.
// Accepts string, integer types, integer-valued float64 ≤2^53, and json.Number.
// Returns "" (skipped) for bool, arrays, maps, fractional/oversized float64,
// nil, and other types.
func normalizeClaimString(v any) string
```

规则：
| 输入 | 结果 |
|---|---|
| `string` | 原样返回 |
| `int`/`int8..64`/`uint`/`uint8..64` | `strconv.FormatInt/FormatUint` 精确十进制 |
| `float64`/`float32` 整数值（`math.Trunc(v)==v`）且 `0 ≤ |v| ≤ 2^53` | `strconv.FormatInt(int64(v))` |
| `float64` 整数值 > 2^53 | `""`（已舍入，无法精确恢复，忽略） |
| `float64` 分数值 | `""`（分数 id 无意义，忽略） |
| `json.Number` | 校验正则 `^[0-9]+$` 整数，返回字面量；否则 `""` |
| bool/数组/对象/struct/nil | `""`（忽略） |

注：int 类型实际不由仓库签发方产生（签发走 JWT JSON），但单元测试与直接调用可能传入，保留精确支持。

## 落地点

| 位置 | 现状 | 改动 |
|---|---|---|
| `common/authctx/claims.go` `ClaimString` | `convertor.ToString` 宽松 | 改调 `normalizeClaimString` |
| `common/authctx/context.go` `toStringClaim` | `convertor.ToString` 宽松 | 改调 `normalizeClaimString`（或复用 `ClaimString` 等价逻辑） |
| `common/mcpx/context_meta.go` `ExtractFromMeta` | `ClaimString`（随之变） | 无需改代码，行为随 `ClaimString` |
| `common/mcpx/auth.go` `NewDualTokenVerifier` | `ClaimString`（`UserID`）；`Extra` 原始值 | `UserID` 随之精确；`Extra` 保留原始值（wire/调试），不转换 |
| `common/authctx/claims.go` `ApplyClaimMapping` | 原始值拷贝 | 保持（只拷贝，不转换；转换发生在读取侧） |

## 兼容性

- **zerorpc int64**：签发 JSON number → go-zero 网关 `json.Number` 精确 → `normalizeClaimString` 输出字面量 ✓；MCP/socket 路径 `float64`（小 int64 精确）✓。
- **socketpush string**：string 直通 ✓。
- **现有合法 token**：行为不变；非法类型不再产生 `"true"`/`["a"]`/`"1.5"` 进身份。
- **网关空 user-id 处理不变**：aigtw/gtw 遇到空仍返回未授权（与忽略语义一致）。

## 测试更新

锁定宽松行为的契约测试需按新矩阵更新：
- `claims_test.go`：bool→`""`（原 `"true"`）、分数→`""`（原 `"1.5"`）；float64 整数、int64、string、json.Number 保留精确断言。
- `context_test.go`：bool/数组→`""`（原转换）。
- `mcpx/context_meta_test.go`：`float64(12)` dept-code → `"12"`（整数值保留）；新增分数/超大跳过用例。
- `mcpx/auth_test.go`：`Extra[user-id]` 保留原始 `float64(42)`；`UserID == "42"` 不变。
- 新增 `normalizeClaimString` 单元测试覆盖全矩阵（string/int/int64/uint/float 整数/float 分数/float>2^53/json.Number 整数/json.Number 非整数/bool/数组/对象/nil/缺失）。

## 禁止混合（audit §8.2）

- 不混合 typed-key 切换（已完成）。
- 不收缩 Authorization 传播。
- 不强制 transport 冲突（`grpcx` first-value 行为不变）。
- 不改 wire key、`b64:`、方法策略。
