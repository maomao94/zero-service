package logic

import (
	"fmt"
	"net/url"
	"time"
	"zero-service/common/djisdk"
	"zero-service/common/mqttx"
	"zero-service/common/tool"
)

// toDrcMqttBroker 将服务 MQTT 配置转换为 DRC MQTT Broker 连接信息。
// 从配置的第一个 Broker 地址中提取 host:port，根据 scheme 判断是否启用 TLS，
// 其余字段直接透传，ExpireTime 设置为 7 天后。
// toDrcMqttBroker 将服务 MQTT 配置转换为 DRC MQTT Broker 连接信息。
// DRC 通道必须使用独立 MQTT 连接，ClientID 每次重新生成，不复用主 MQTT ClientID。
func toDrcMqttBroker(cfg mqttx.MqttConfig) djisdk.DrcMqttBroker {
	uid, _ := tool.SimpleUUID()
	clientID := fmt.Sprintf("dji-cloud-drc-%s", uid)
	broker := djisdk.DrcMqttBroker{
		ClientID:   clientID,
		Username:   cfg.Username,
		Password:   cfg.Password,
		ExpireTime: time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	if len(cfg.Broker) > 0 {
		addr := cfg.Broker[0]
		if u, err := url.Parse(addr); err == nil && u.Host != "" {
			broker.Address = u.Host
			broker.EnableTLS = u.Scheme == "tcps" || u.Scheme == "mqtts"
		} else {
			// 非 URL 格式，直接作为 address 使用
			broker.Address = addr
		}
	}
	return broker
}
