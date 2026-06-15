package mqttx

// MqttConfig MQTT 客户端配置
type MqttConfig struct {
	// Broker MQTT 服务器地址列表，如 "tcp://localhost:1883"
	Broker []string `json:",optional"`
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
	// AutoSubscribe 添加处理器时是否自动订阅，默认 true
	AutoSubscribe bool `json:",default=true"`
	// SubscribeTopics 初始化时需要订阅的主题列表
	SubscribeTopics []string `json:",optional"`
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
// Unlike AddHandler, this only registers the handler; MQTT subscription is deferred
// to the connection event via restoreSubscriptions.
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
