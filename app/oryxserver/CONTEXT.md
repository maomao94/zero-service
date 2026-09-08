# Oryx Relay Context

Oryx Relay 上下文描述从外部源拉流并推送到 SRS/Oryx 目标的运行活动。目标身份、单次运行身份与录制身份彼此独立。

## Language

**Relay**:
从外部源拉流并推送到一个 Relay Target 的运行活动。
_Avoid_: Recording

**Relay Target**:
由应用名与流名 `(app, stream)` 唯一确定的推流目标。
_Avoid_: Relay ID, Recording UUID

**Relay ID**:
一次 Relay 启动时生成的 UUID；同一 Relay Target 再次启动会获得新的 Relay ID。
_Avoid_: Relay Target, Recording UUID, Canonical UID

**Recording UUID**:
Oryx 录制任务的身份，不代表 Relay 或 Relay Target。
_Avoid_: Relay ID
