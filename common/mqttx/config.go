package mqttx

// MqttConfig MQTT 客户端配置
type MqttConfig struct {
	// Broker MQTT 服务器地址列表，如 "tcp://localhost:1883"
	Broker []string
	// ClientID 客户端标识，不填则自动生成
	ClientID string `json:",optional"`
	// Username 用户名（可选）
	Username string `json:",optional"`
	// Password 密码（可选）
	Password string `json:",optional"`
	// Qos 服务质量等级 0=最多一次, 1=至少一次, 2=恰好一次，默认 1
	Qos byte `json:",optional"`
	// Timeout 连接和操作超时时间（毫秒），默认 30000
	Timeout int64 `json:",default=30000"`
	// KeepAlive 心跳间隔（毫秒），默认 60000
	KeepAlive int64 `json:",default=60000"`
	// SubscribeTopics 初始化时需要订阅的主题列表
	SubscribeTopics []string `json:",optional"`
	// TLS 加密连接配置，Broker 使用 ssl:// / tls:// / mqtts:// 等加密协议时生效
	TLS TLSConfig `json:",optional"`
}

// TLSConfig MQTT TLS 连接配置。
// 仅需校验自签服务端证书时配置 CAFile；需要双向认证时同时配置 CertFile 与 KeyFile。
type TLSConfig struct {
	// CAFile 服务端证书校验用 CA 证书路径（PEM），为空时使用系统根证书
	CAFile string `json:",optional"`
	// CertFile 客户端证书路径（PEM），与 KeyFile 成对配置
	CertFile string `json:",optional"`
	// KeyFile 客户端私钥路径（PEM），与 CertFile 成对配置
	KeyFile string `json:",optional"`
	// ServerName 服务端证书校验域名，为空时使用 Broker 地址中的 host
	ServerName string `json:",optional"`
	// InsecureSkipVerify 跳过服务端证书校验，仅限内网自签证书排障使用
	InsecureSkipVerify bool `json:",optional"`
}

// isEmpty 判断是否未配置任何 TLS 参数，未配置时不覆盖 paho 的默认 TLS 行为。
func (c TLSConfig) isEmpty() bool {
	return c.CAFile == "" && c.CertFile == "" && c.KeyFile == "" &&
		c.ServerName == "" && !c.InsecureSkipVerify
}

// ClientOptions holds configuration for creating a Client.
type ClientOptions struct {
	onReady      func(Client)
	replyRouters []replyRouterRegistration
}

type replyRouterRegistration struct {
	topicTemplate string
	handler       ConsumeHandler
}

// ClientOption configures ClientOptions.
type ClientOption func(*ClientOptions)

// WithOnReady 设置首次连接成功时的回调（仅执行一次）
// 通常在此回调中注册消息处理器
func WithOnReady(fn func(Client)) ClientOption {
	return func(o *ClientOptions) {
		o.onReady = fn
	}
}

// WithReplyRouter registers a reply router that takes priority in dispatch.
// topicTemplate may contain MQTT wildcards such as + or #.
// Registration happens before the mqttx.Client exists, so MQTT subscription is
// deferred to the connection event via restoreSubscriptions; use
// Client.AddReplyRouter when the client already exists (it subscribes immediately).
// 发送侧需自己持有同一个 router 实例并调用其 RequestReply。
func WithReplyRouter[T any](topicTemplate string, router *ReplyRouter[T]) ClientOption {
	return func(o *ClientOptions) {
		if router != nil {
			o.replyRouters = append(o.replyRouters, replyRouterRegistration{
				topicTemplate: topicTemplate,
				handler:       router,
			})
		}
	}
}
