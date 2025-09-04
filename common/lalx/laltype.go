package lalx

// FrameData 帧率数据：对应接口返回的 in_frame_per_sec 数组元素（最近32秒每秒视频帧数）
type FrameData struct {
	// 时间戳（秒级Unix时间，如1723513826，必返）
	UnixSec int64 `json:"unix_sec"` // 驼峰转下划线：unixSec → unix_sec
	// 每秒视频帧数（如15，必返）
	V int32 `json:"v"` // 单个字母无需转换，保持原标签
}

// PubSessionInfo 发布者会话信息：对应接口返回的 data.pub（接收推流的详情）
type PubSessionInfo struct {
	// 会话ID（全局唯一标识，如"RTMPPUBSUB1"，必返）
	SessionId string `json:"session_id"` // 驼峰转下划线：sessionId → session_id
	// 推流协议（取值："RTMP"|"RTSP"，必返）
	Protocol string `json:"protocol"` // 无驼峰，保持原标签
	// 基础类型（固定为"PUB"，标识发布者角色，必返）
	BaseType string `json:"base_type"` // 驼峰转下划线：baseType → base_type
	// 推流开始时间（格式化字符串，如"2020-10-11 19:17:41.586"，必返）
	StartTime string `json:"start_time"` // 驼峰转下划线：startTime → start_time
	// 对端地址（客户端IP:端口，如"127.0.0.1:61353"，必返）
	RemoteAddr string `json:"remote_addr"` // 驼峰转下划线：remoteAddr → remote_addr
	// 累计读取数据大小（从推流开始计算，单位字节，必返）
	ReadBytesSum int64 `json:"read_bytes_sum"` // 驼峰转下划线：readBytesSum → read_bytes_sum
	// 累计发送数据大小（从推流开始计算，单位字节，必返）
	WroteBytesSum int64 `json:"wrote_bytes_sum"` // 驼峰转下划线：wroteBytesSum → wrote_bytes_sum
	// 最近5秒总码率（单位kbit/s，对PUB类型等价于read_bitrate_kbits，必返）
	BitrateKbits int32 `json:"bitrate_kbits"` // 驼峰转下划线：bitrateKbits → bitrate_kbits
	// 最近5秒读取数据码率（单位kbit/s，必返）
	ReadBitrateKbits int32 `json:"read_bitrate_kbits"` // 驼峰转下划线：readBitrateKbits → read_bitrate_kbits
	// 最近5秒发送数据码率（单位kbit/s，必返）
	WriteBitrateKbits int32 `json:"write_bitrate_kbits"` // 驼峰转下划线：writeBitrateKbits → write_bitrate_kbits
}

// SubSessionInfo 订阅者会话信息：对应接口返回的 data.subs 数组元素（拉流的详情）
type SubSessionInfo struct {
	// 会话ID（全局唯一标识，如"FLVSUB1"，必返）
	SessionId string `json:"session_id"` // 驼峰转下划线：sessionId → session_id
	// 拉流协议（取值："RTMP"|"FLV"|"TS"，必返）
	Protocol string `json:"protocol"` // 无驼峰，保持原标签
	// 基础类型（固定为"SUB"，标识订阅者角色，必返）
	BaseType string `json:"base_type"` // 驼峰转下划线：baseType → base_type
	// 拉流开始时间（格式化字符串，如"2020-10-11 19:19:21.724"，必返）
	StartTime string `json:"start_time"` // 驼峰转下划线：startTime → start_time
	// 对端地址（客户端IP:端口，如"127.0.0.1:61785"，必返）
	RemoteAddr string `json:"remote_addr"` // 驼峰转下划线：remoteAddr → remote_addr
	// 累计读取数据大小（从拉流开始计算，单位字节，必返）
	ReadBytesSum int64 `json:"read_bytes_sum"` // 驼峰转下划线：readBytesSum → read_bytes_sum
	// 累计发送数据大小（从拉流开始计算，单位字节，必返）
	WroteBytesSum int64 `json:"wrote_bytes_sum"` // 驼峰转下划线：wroteBytesSum → wrote_bytes_sum
	// 最近5秒总码率（单位kbit/s，对SUB类型等价于write_bitrate_kbits，必返）
	BitrateKbits int32 `json:"bitrate_kbits"` // 驼峰转下划线：bitrateKbits → bitrate_kbits
	// 最近5秒读取数据码率（单位kbit/s，必返）
	ReadBitrateKbits int32 `json:"read_bitrate_kbits"` // 驼峰转下划线：readBitrateKbits → read_bitrate_kbits
	// 最近5秒发送数据码率（单位kbit/s，必返）
	WriteBitrateKbits int32 `json:"write_bitrate_kbits"` // 驼峰转下划线：writeBitrateKbits → write_bitrate_kbits
}

// PullSessionInfo 中继拉流会话信息：对应接口返回的 data.pull（从其他节点拉流回源的详情）
type PullSessionInfo struct {
	// 会话ID（全局唯一标识，如"RTMPPULL1"，必返）
	SessionId string `json:"session_id"` // 驼峰转下划线：sessionId → session_id
	// 拉流协议（取值："RTMP"|"RTSP"，必返）
	Protocol string `json:"protocol"` // 无驼峰，保持原标签
	// 基础类型（固定为"PULL"，标识中继拉流角色，必返）
	BaseType string `json:"base_type"` // 驼峰转下划线：baseType → base_type
	// 拉流开始时间（格式化字符串，如"2020-10-11 19:20:00.123"，必返）
	StartTime string `json:"start_time"` // 驼峰转下划线：startTime → start_time
	// 对端地址（源节点IP:端口，如"192.168.1.10:1935"，必返）
	RemoteAddr string `json:"remote_addr"` // 驼峰转下划线：remoteAddr → remote_addr
	// 累计读取数据大小（从拉流开始计算，单位字节，必返）
	ReadBytesSum int64 `json:"read_bytes_sum"` // 驼峰转下划线：readBytesSum → read_bytes_sum
	// 累计发送数据大小（从拉流开始计算，单位字节，必返）
	WroteBytesSum int64 `json:"wrote_bytes_sum"` // 驼峰转下划线：wroteBytesSum → wrote_bytes_sum
	// 最近5秒总码率（单位kbit/s，对PULL类型等价于read_bitrate_kbits，必返）
	BitrateKbits int32 `json:"bitrate_kbits"` // 驼峰转下划线：bitrateKbits → bitrate_kbits
	// 最近5秒读取数据码率（单位kbit/s，必返）
	ReadBitrateKbits int32 `json:"read_bitrate_kbits"` // 驼峰转下划线：readBitrateKbits → read_bitrate_kbits
	// 最近5秒发送数据码率（单位kbit/s，必返）
	WriteBitrateKbits int32 `json:"write_bitrate_kbits"` // 驼峰转下划线：writeBitrateKbits → write_bitrate_kbits
}

// PushSessionInfo 中继推流会话信息：对应接口返回的 data.pushs（主动外连转推，暂不提供数据）
type PushSessionInfo struct {
	// 预留字段，后续接口开放时可参考PullSessionInfo补充，当前暂空
}

// GroupData 分组核心数据：对应接口返回的 data 字段（流的完整信息聚合）
type GroupData struct {
	// 流名称（如"test110"，必返）
	StreamName string `json:"stream_name"` // 驼峰转下划线：streamName → stream_name
	// 应用名（如"live"，必返）
	AppName string `json:"app_name"` // 驼峰转下划线：appName → app_name
	// 音频编码格式（如"AAC"，必返）
	AudioCodec string `json:"audio_codec"` // 驼峰转下划线：audioCodec → audio_codec
	// 视频编码格式（如"H264"|"H265"，必返）
	VideoCodec string `json:"video_codec"` // 驼峰转下划线：videoCodec → video_codec
	// 视频宽度（像素，如640，必返）
	VideoWidth int32 `json:"video_width"` // 驼峰转下划线：videoWidth → video_width
	// 视频高度（像素，如360，必返）
	VideoHeight int32 `json:"video_height"` // 驼峰转下划线：videoHeight → video_height
	// 发布者会话信息（推流详情，无推流时为nil，必返）
	Pub *PubSessionInfo `json:"pub"` // 无驼峰，保持原标签（接口返回字段为"pub"）
	// 订阅者会话列表（拉流详情，无拉流时为空切片，必返）
	Subs []*SubSessionInfo `json:"subs"` // 无驼峰，保持原标签（接口返回字段为"subs"）
	// 中继拉流会话信息（回源流详情，无回源时为nil，必返）
	Pull *PullSessionInfo `json:"pull"` // 无驼峰，保持原标签（接口返回字段为"pull"）
	// 中继推流会话列表（外连转推，暂为空数组，必返）
	Pushs []*PushSessionInfo `json:"pushs"` // 无驼峰，保持原标签（接口返回字段为"pushs"）
	// 最近32秒视频帧率统计（每秒视频帧数，必返）
	InFramePerSec []*FrameData `json:"in_frame_per_sec"` // 驼峰转下划线：inFramePerSec → in_frame_per_sec
}

// LalServerData 对应 /api/stat/lal_info 返回的「服务器基础信息」
type LalServerData struct {
	// 版本信息（必返）
	ServerId      string `json:"server_id"`      // 服务器唯一 ID
	BinInfo       string `json:"bin_info"`       // 二进制文件信息
	LalVersion    string `json:"lal_version"`    // lal可执行文件版本信息
	ApiVersion    string `json:"api_version"`    // HTTP API接口版本信息
	NotifyVersion string `json:"notify_version"` // HTTP Notify版本信息
	WebUiVersion  string `json:"web_ui_version"` // Web UI版本信息
	// 时间信息（必返）
	StartTime string `json:"start_time"` // lal进程启动时间
}
