package lalx

// 流分组数据结构
type GroupData struct {
	StreamName   string         `json:"stream_name"`
	CreateTimeMs int64          `json:"create_time_ms"`
	Publisher    *SessionInfo   `json:"publisher"`
	Subscribers  []*SessionInfo `json:"subscribers"`
	Pullers      []*SessionInfo `json:"pullers"`
}

// 会话信息数据结构
type SessionInfo struct {
	SessionId    string `json:"session_id"`
	Type         string `json:"type"`
	ClientIp     string `json:"client_ip"`
	CreateTimeMs int64  `json:"create_time_ms"`
	SendBytes    int64  `json:"send_Bytes"`
	RecvBytes    int64  `json:"recv_Bytes"`
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
