package oryxx

// RecordQueryData 录制配置查询结果（record/query 的 data）
type RecordQueryData struct {
	// 是否录制所有流
	All bool `json:"all"`
	// 录制根目录
	Home string `json:"home"`
	// 录制 glob 模式列表
	Globs []string `json:"globs"`
	// 后处理拷贝目录
	ProcessCpDir string `json:"processCpDir"`
}

// RecordFile 录制文件信息（record/files 的 data 数组元素）
type RecordFile struct {
	// 录制任务 UUID
	UUID string `json:"uuid"`
	// 虚拟主机
	Vhost string `json:"vhost"`
	// 应用名
	App string `json:"app"`
	// 流名
	Stream string `json:"stream"`
	// 是否正在处理
	Progress bool `json:"progress"`
	// 最后更新时间
	Update string `json:"update"`
	// 文件数量
	NN int32 `json:"nn"`
	// 总时长（秒）
	Duration float64 `json:"duration"`
	// 总大小（字节）
	Size uint64 `json:"size"`
}

// HooksQueryData 回调配置查询结果（hooks/query 的 data）
type HooksQueryData struct {
	// 请求配置（JSON 字符串）
	Req string `json:"req"`
	// 响应配置（JSON 字符串）
	Res string `json:"res"`
	// 回调 target 地址
	Target string `json:"target"`
	// 回调透传凭证
	Opaque string `json:"opaque"`
	// 是否监听所有事件
	All bool `json:"all"`
	// 回调 host
	Host string `json:"host"`
}

// HooksApplyReq 应用回调配置请求体（hooks/apply）
type HooksApplyReq struct {
	// 回调 target 地址
	Target string `json:"target"`
	// 回调透传凭证
	Opaque string `json:"opaque"`
	// 是否监听所有事件
	All bool `json:"all"`
	// 回调 host
	Host string `json:"host"`
}

// DvrQueryData DVR 云录制配置查询结果（dvr/query 的 data）
type DvrQueryData struct {
	// 是否录制所有流
	All bool `json:"all"`
	// 是否已配置云存储密钥
	Secret bool `json:"secret"`
}

// DvrFile DVR 云录制文件信息（dvr/files 的 data 数组元素）
type DvrFile struct {
	// 录制任务 UUID
	UUID string `json:"uuid"`
	// 虚拟主机
	Vhost string `json:"vhost"`
	// 应用名
	App string `json:"app"`
	// 流名
	Stream string `json:"stream"`
	// 是否正在处理
	Progress bool `json:"progress"`
	// 最后更新时间
	Update string `json:"update"`
	// 文件数量
	NN int32 `json:"nn"`
	// 总时长（秒）
	Duration float64 `json:"duration"`
	// 总大小（字节）
	Size uint64 `json:"size"`
	// 云存储 bucket（仅 DVR）
	Bucket string `json:"bucket"`
	// 云存储 region（仅 DVR）
	Region string `json:"region"`
}

// ==================== SRS HTTP API DTO（Oryx 代理 /api/v1/*）====================

// SrsKbps 码率（kb/s，近 30s）
type SrsKbps struct {
	// 接收码率（kb/s，近 30s）
	Recv30s float64 `json:"recv_30s"`
	// 发送码率（kb/s，近 30s）
	Send30s float64 `json:"send_30s"`
}

// SrsStream SRS 流信息（/api/v1/streams 元素；video/audio 未推流时为 null）
type SrsStream struct {
	// 流 id
	ID int64 `json:"id"`
	// 流名（不含前缀，如 obs）
	Name string `json:"name"`
	// 虚拟主机
	Vhost string `json:"vhost"`
	// 应用名
	App string `json:"app"`
	// 推流地址（不含 query）
	TcUrl string `json:"tcUrl"`
	// 完整 url（含 query 参数）
	URL string `json:"url"`
	// 存活时长（ms）
	LiveMs int64 `json:"live_ms"`
	// 客户端数量
	Clients int32 `json:"clients"`
	// 帧数
	Frames int64 `json:"frames"`
	// 发送字节数
	SendBytes uint64 `json:"send_bytes"`
	// 接收字节数
	RecvBytes uint64 `json:"recv_bytes"`
	// 码率（近 30s）
	Kbps SrsKbps `json:"kbps"`
	// 推流状态
	Publish struct {
		// 是否有活跃推流
		Active bool `json:"active"`
		// 推流客户端 id（未推流时为空）
		Cid string `json:"cid"`
	} `json:"publish"`
	// 视频编码信息（无视频时 null）
	Video *struct {
		// 编解码器，如 HEVC
		Codec string `json:"codec"`
		// 视频编码 profile
		Profile string `json:"profile"`
		// 视频编码 level
		Level string `json:"level"`
		// 视频宽
		Width int32 `json:"width"`
		// 视频高
		Height int32 `json:"height"`
	} `json:"video"`
	// 音频编码信息（无音频时 null）
	Audio *struct {
		// 编解码器，如 AAC
		Codec string `json:"codec"`
		// 采样率（Hz）
		SampleRate int32 `json:"sample_rate"`
		// 声道数
		Channel int32 `json:"channel"`
		// 音频编码 profile
		Profile string `json:"profile"`
	} `json:"audio"`
}

// SrsClient SRS 客户端信息（/api/v1/clients 元素）
type SrsClient struct {
	// 客户端 id
	ID int64 `json:"id"`
	// 虚拟主机
	Vhost string `json:"vhost"`
	// 流名
	Stream string `json:"stream"`
	// 客户端 IP
	IP string `json:"ip"`
	// 页面 URL（HTTP 拉流等场景）
	PageURL string `json:"pageUrl"`
	// swf 播放器 URL
	SwfURL string `json:"swfUrl"`
	// 推流地址（含 query）
	TcURL string `json:"tcUrl"`
	// 完整 url
	URL string `json:"url"`
	// 客户端名
	Name string `json:"name"`
	// 类型：publish/play 等
	Type string `json:"type"`
	// 是否推流
	Publish bool `json:"publish"`
	// 存活时长（秒）
	Alive float64 `json:"alive"`
	// 发送字节数
	SendBytes uint64 `json:"send_bytes"`
	// 接收字节数
	RecvBytes uint64 `json:"recv_bytes"`
	// 码率（近 30s）
	Kbps SrsKbps `json:"kbps"`
}

// SrsVhost SRS 虚拟主机信息（/api/v1/vhosts 元素）
type SrsVhost struct {
	// 虚拟主机 id
	ID string `json:"id"`
	// 虚拟主机名
	Name string `json:"name"`
	// 是否启用
	Enabled bool `json:"enabled"`
	// 客户端数量
	Clients int32 `json:"clients"`
	// 流数量
	Streams int32 `json:"streams"`
	// 发送字节数
	SendBytes uint64 `json:"send_bytes"`
	// 接收字节数
	RecvBytes uint64 `json:"recv_bytes"`
	// 码率（近 30s）
	Kbps SrsKbps `json:"kbps"`
	// HLS 配置
	HLS struct {
		// 是否启用 HLS
		Enabled bool `json:"enabled"`
		// 分片时长（秒）
		Fragment float64 `json:"fragment"`
	} `json:"hls"`
}

// SrsSelfInfo SRS 进程自身状态（summaries.self）
type SrsSelfInfo struct {
	// SRS 版本号
	Version string `json:"version"`
	// 进程 pid
	Pid string `json:"pid"`
	// 父进程 pid
	Ppid string `json:"ppid"`
	// 启动参数
	Argv string `json:"argv"`
	// 工作目录
	Cwd string `json:"cwd"`
	// 内存占用（KB）
	MemKbyte int64 `json:"mem_kbyte"`
	// 内存占比（%）
	MemPercent float64 `json:"mem_percent"`
	// CPU 占比（%）
	CPUPercent float64 `json:"cpu_percent"`
	// 运行时长（秒）
	SrsUptime float64 `json:"srs_uptime"`
}

// SrsSystem SRS 系统整体状态（summaries.system；字段名与 SRS 源码保持一致）
type SrsSystem struct {
	// CPU 占比（%）
	CPUPercent float64 `json:"cpu_percent"`
	// 磁盘读速率（KB/s）
	DiskReadKBps float64 `json:"disk_read_KBps"`
	// 磁盘写速率（KB/s）
	DiskWriteKBps float64 `json:"disk_write_KBps"`
	// 磁盘忙碌率（%）
	DiskBusyPercent float64 `json:"disk_busy_percent"`
	// 内存总量（KB）
	MemRamKbyte int64 `json:"mem_ram_kbyte"`
	// 内存占比（%）
	MemRamPercent float64 `json:"mem_ram_percent"`
	// 交换内存总量（KB）
	MemSwapKbyte int64 `json:"mem_swap_kbyte"`
	// 交换内存占比（%）
	MemSwapPercent float64 `json:"mem_swap_percent"`
	// CPU 逻辑核数
	Cpus int32 `json:"cpus"`
	// 在线 CPU 核数
	CpusOnline int32 `json:"cpus_online"`
	// 系统运行时长（秒）
	Uptime float64 `json:"uptime"`
	// 空闲时长（秒，源码拼写为 ilde_time，保持原样以正确解析）
	IdleTime float64 `json:"ilde_time"`
	// 1 分钟负载
	Load1m float64 `json:"load_1m"`
	// 5 分钟负载
	Load5m float64 `json:"load_5m"`
	// 15 分钟负载
	Load15m float64 `json:"load_15m"`
	// 网络采样时间（系统）
	NetSampleTime float64 `json:"net_sample_time"`
	// 网络接收字节数（系统）
	NetRecvBytes uint64 `json:"net_recv_bytes"`
	// 网络发送字节数（系统）
	NetSendBytes uint64 `json:"net_send_bytes"`
	// 网络接收字节数（自上次统计）
	NetRecviBytes uint64 `json:"net_recvi_bytes"`
	// 网络发送字节数（自上次统计）
	NetSendiBytes uint64 `json:"net_sendi_bytes"`
	// 网络采样时间（SRS）
	SrsSampleTime float64 `json:"srs_sample_time"`
	// SRS 接收字节数（系统）
	SrsRecvBytes uint64 `json:"srs_recv_bytes"`
	// SRS 发送字节数（系统）
	SrsSendBytes uint64 `json:"srs_send_bytes"`
	// 系统 socket 连接数
	ConnSys int64 `json:"conn_sys"`
	// 系统 ESTABLISHED 连接数
	ConnSysET int64 `json:"conn_sys_et"`
	// 系统 TIME_WAIT 连接数
	ConnSysTW int64 `json:"conn_sys_tw"`
	// 系统 UDP 连接数
	ConnSysUDP int64 `json:"conn_sys_udp"`
	// SRS 自身连接数
	ConnSrs int64 `json:"conn_srs"`
}
