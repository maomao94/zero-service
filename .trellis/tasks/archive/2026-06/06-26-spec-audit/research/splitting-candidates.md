# Research: Splitting Candidates (Files > 300 lines)

- **Query**: Files over ~300 lines covering multiple distinct topics
- **Scope**: internal (spec files)
- **Date**: 2026-06-26

## Findings

### 1. messaging-guidelines.md — 665 lines, 6 distinct scenarios

**File**: `.trellis/spec/backend/messaging-guidelines.md`

**Line count**: 665 lines

**Topics covered**:
1. Kafka trace header propagation (lines 3-72)
2. Kafka bridge module (bridgekafka) (lines 74-318)
3. MQTT request/reply routing with mqttx (lines 319-467)
4. Device heartbeat manager pattern (lines 468-550)
5. Multiple design decisions for mqttx (lines 551-665)

**Recommendation**: Split into 3 files:
- `kafka-guidelines.md` — Kafka trace + bridgekafka configuration (lines 1-318)
- `mqttx-request-reply.md` — MQTT reply router + RequestReply + design decisions (lines 319-665)
- `device-heartbeat-pattern.md` — heartbeat manager pattern (lines 468-550), unless it should stay in messaging

### 2. iec104-control-commands.md — 1022 lines, multiple scenarios

**File**: `.trellis/spec/backend/iec104-control-commands.md`

**Line count**: 1022 lines

**Topics covered**:
1. Typed control command RPC (lines 26-260)
2. Cluster broadcast ACK reply via MQTT (lines 268-498)
3. ACK replyPool + Command Option + Helpers (lines 500-637)
4. Design decisions (lines 660-680)
5. Enum value alignment (lines 682-695)
6. ASDU receipt handling (lines 706-839)
7. Multiple gotchas and conventions

**Recommendation**: Split into 2-3 files:
- `iec104-control-commands.md` — RPC + typeId mappings + command lifecycle (lines 1-260)
- `iec104-cluster-broadcast.md` — cluster broadcast + ACK reply + MQTT routing (lines 268-498)
- `iec104-asdu-handling.md` — ASDU processing + COT handling + gotchas (lines 500-1022)

### 3. gisx-guidelines.md — 676 lines, multiple distinct subsystems

**File**: `.trellis/spec/backend/gisx-guidelines.md`

**Line count**: 676 lines

**Topics covered**:
1. common/gisx/ package boundaries (lines 5-25)
2. Coordinate system conventions (lines 26-38)
3. GEOS tool layer — Docker/CGO build contract (lines 39-334)
4. FenceStore interface pattern + recall index (lines 341-510)
5. app/gis/ service architecture (lines 369-632)
6. Proto conventions (lines 593-609)
7. Common traps (lines 611-633)
8. Test coverage (lines 634-676)

**Recommendation**: Consider splitting:
- `gisx-geos-guidelines.md` — GEOS tool layer + Docker/CGO (lines 39-334)
- `gis-service-guidelines.md` — app/gis/ architecture (lines 369-632)
- Keep `gisx-guidelines.md` for common/gisx/ boundaries + FenceStore + coordinate conventions

## Caveats / Not Found

- The large file sizes are partially due to the detailed Scenario format (Scope/Signatures/Contracts/Validation/Good-Base-Bad/Tests/Wrong-Correct), which is intentionally verbose. Splitting should preserve this structure.
