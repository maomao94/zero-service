package lalx

// GroupData 对应 /api/stat/group 或 /api/stat/all_group 返回的「单流分组详情」
// 完全对齐 lal 官方 JSON 结构，补充状态、会话统计等关键字段
type GroupData struct {
	// 基础标识
	StreamName string `json:"stream_name"` // 流名称（唯一标识，必返）
	AppName    string `json:"app_name"`    // 应用名（如 live，部分场景返回，非必返）
	Status     string `json:"status"`      // 分组状态（active=活跃/inactive=非活跃，必返）

	// 音视频编码信息（必返，无对应流时为空字符串）
	AudioCodec  string `json:"audio_codec"`  // 音频编码（如 aac、mp3）
	VideoCodec  string `json:"video_codec"`  // 视频编码（如 h264、h265）
	VideoWidth  int    `json:"video_width"`  // 视频宽度（像素，无视频时为 0）
	VideoHeight int    `json:"video_height"` // 视频高度（像素，无视频时为 0）

	// 会话列表（按类型区分，无对应会话时为 nil 或空切片）
	Pub   SessionInfo   `json:"pub"`   // 发布者会话（单会话，无则为 nil）
	Subs  []SessionInfo `json:"subs"`  // 订阅者会话列表（多会话，无则为空切片）
	Pull  SessionInfo   `json:"pull"`  // 中继拉流会话（单会话，无则为 nil）
	Pushs []SessionInfo `json:"pushs"` // 中继推流会话列表（多会话，无则为空切片，替换原 interface{} 确保类型安全）

	// 会话统计（必返，反映分组负载）
	PubCount          int `json:"pub_count"`           // 发布者数量（0/1，多发布场景可能大于 1）
	SubCount          int `json:"sub_count"`           // 订阅者数量
	PullCount         int `json:"pull_count"`          // 中继拉流数量
	TotalSessionCount int `json:"total_session_count"` // 总会话数（pub+sub+pull+push）

	// 帧率数据（必返，最近一段时间的帧统计）
	InFramePerSec []FrameData `json:"in_frame_per_sec"` // 输入帧统计（每秒视频/音频帧数）

	// 时间信息（必返）
	CreateTimeMs int64 `json:"create_time_ms"` // 分组创建时间戳（毫秒级 Unix 时间）
}

// SessionInfo 对应各类会话（pub/sub/pull/push）的详情
// 补充存活时长、客户端地址拆分等字段，修正比特率类型
type SessionInfo struct {
	SessionId string `json:"session_id"` // 会话唯一 ID（如 FLVSUB1、RTMPPULL1，必返）
	Protocol  string `json:"protocol"`   // 协议类型（如 rtmp、flv、hls、rtsp，必返）
	BaseType  string `json:"base_type"`  // 会话类型（pub=发布者/sub=订阅者/pull=拉流/push=推流，必返）
	StartTime string `json:"start_time"` // 会话启动时间（格式化字符串，如 "2024-09-04 15:30:00"，必返）
	AliveSec  int    `json:"alive_sec"`  // 会话存活时长（秒，必返，补充原缺失字段）

	// 地址信息（拆分 IP 和端口，兼容官方返回的两种格式：remote_addr 是 "ip:port" 字符串，client_ip/client_port 是单独字段）
	RemoteAddr string `json:"remote_addr"` // 客户端地址（ip:port 格式，必返）
	ClientIp   string `json:"client_ip"`   // 客户端 IP（单独字段，部分接口返回，非必返）
	ClientPort int    `json:"client_port"` // 客户端端口（单独字段，部分接口返回，非必返）

	// 流量统计（必返，int64 避免溢出）
	ReadBytesSum  int64 `json:"read_bytes_sum"`  // 累计读取字节数（从客户端到服务端）
	WroteBytesSum int64 `json:"wrote_bytes_sum"` // 累计写入字节数（从服务端到客户端）

	// 比特率（当前实时比特率，单位 kbps，必返，修正原字段注释）
	BitrateKbits      int `json:"bitrate_kbits"`       // 总比特率（read+write）
	ReadBitrateKbits  int `json:"read_bitrate_kbits"`  // 读取比特率（客户端→服务端）
	WriteBitrateKbits int `json:"write_bitrate_kbits"` // 写入比特率（服务端→客户端）
}

// FrameData 对应每秒输入帧统计（视频+音频，补充原缺失的音频帧数）
type FrameData struct {
	UnixSec     int64 `json:"unix_sec"` // 时间戳（秒级 Unix 时间，必返）
	VideoFrames int   `json:"v"`        // 视频帧数（每秒，必返，原 V 修正为 VideoFrames 更清晰）
	AudioFrames int   `json:"a"`        // 音频帧数（每秒，必返，补充原缺失字段）
}

// LalServerData 对应 /api/stat/lal_info 返回的「服务器基础信息」
// 修正 JSON 标签错误（如 server_Id→server_id），补充运行时长、负载等关键字段
type LalServerData struct {
	// 版本信息（必返）
	ServerId      string `json:"server_id"`      // 服务器唯一 ID
	BinInfo       string `json:"bin_info"`       // 二进制文件信息
	LalVersion    string `json:"lal_version"`    // lal可执行文件版本信息
	ApiVersion    string `json:"api_version"`    // HTTP API接口版本信息
	NotifyVersion string `json:"notify_version"` // HTTP Notify版本信息
	// 时间信息（必返）
	StartTime string `json:"start_time"` // lal进程启动时间
}
