package lalx

// 流分组数据结构 - 根据实际JSON格式更新
type GroupData struct {
	StreamName    string         `json:"stream_name"`
	AppName       string         `json:"app_name"`
	AudioCodec    string         `json:"audio_codec"`
	VideoCodec    string         `json:"video_codec"`
	VideoWidth    int            `json:"video_width"`
	VideoHeight   int            `json:"video_height"`
	Pub           *SessionInfo   `json:"pub"`
	Subs          []*SessionInfo `json:"subs"`
	Pull          *SessionInfo   `json:"pull"`
	Pushs         []interface{}  `json:"pushs"`
	InFramePerSec []FrameData    `json:"in_frame_per_sec"`
}

// 会话信息数据结构 - 根据实际JSON格式更新
type SessionInfo struct {
	SessionId         string `json:"session_id"`
	Protocol          string `json:"protocol"`
	BaseType          string `json:"base_type"`
	StartTime         string `json:"start_time"`
	RemoteAddr        string `json:"remote_addr"`
	ReadBytesSum      int64  `json:"read_bytes_sum"`
	WroteBytesSum     int64  `json:"wrote_bytes_sum"`
	BitrateKbits      int    `json:"bitrate_kbits"`
	ReadBitrateKbits  int    `json:"read_bitrate_kbits"`
	WriteBitrateKbits int    `json:"write_bitrate_kbits"`
}

// 帧率数据结构
type FrameData struct {
	UnixSec int64 `json:"unix_sec"`
	V       int   `json:"v"`
}

// 服务器信息数据结构
type LalServerData struct {
	ServerId      string `json:"server_Id"`
	BinInfo       string `json:"bin_info"`
	LalVersion    string `json:"lal_version"`
	ApiVersion    string `json:"api_version"`
	NotifyVersion string `json:"notify_version"`
	StartTime     string `json:"start_time"`
}
