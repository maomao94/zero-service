# Live Context

Live 上下文描述业务会议与 LiveKit 媒体资源之间的对应关系。业务会议身份与平台分配的资源身份必须区分。

## Language

**Meeting**:
一次业务会议的记录，不等同于承载音视频的 LiveKit Room。
_Avoid_: Room

**Room**:
承载会议实时音视频的 LiveKit 媒体资源。
_Avoid_: Meeting

**Meeting Number**:
业务会议号，同时作为对应 LiveKit Room 的名称。
_Avoid_: Meeting Code, Room SID

**Meeting Code**:
用户输入的 9 位会议码。
_Avoid_: Meeting Number, Room SID

**Room SID**:
LiveKit 为 Room 分配的平台身份，不是业务会议号。
_Avoid_: Meeting Number

**Participant Identity**:
参与者在一个 Room 内的唯一身份。
_Avoid_: Meeting Number, Room SID
