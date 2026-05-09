package logic

import (
	"net/url"
	"time"

	"zero-service/common/djisdk"
	"zero-service/common/mqttx"
)

// toDrcMqttBroker 将服务 MQTT 配置转换为 DRC MQTT Broker 连接信息。
// 从配置的第一个 Broker 地址中提取 host:port，根据 scheme 判断是否启用 TLS，
// 其余字段直接透传，ExpireTime 设置为 7 天后。
func toDrcMqttBroker(cfg mqttx.MqttConfig) djisdk.DrcMqttBroker {
	broker := djisdk.DrcMqttBroker{
		ClientID:   cfg.ClientID,
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
