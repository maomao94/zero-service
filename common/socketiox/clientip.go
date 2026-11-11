package socketiox

import (
	"net"
	"net/http"
	"strings"

	"github.com/doquangtan/socketio/v4/engineio"
)

const (
	// clientIPHeaderForwarded 是代理链追加的客户端地址头，最左侧为原始客户端。
	clientIPHeaderForwarded = "X-Forwarded-For"
	// clientIPHeaderReal 是部分代理（如 nginx）用对端地址覆盖写入的客户端地址头。
	clientIPHeaderReal = "X-Real-Ip"
	// clientIPHeaderInternal 是内部传递用的客户端地址头。
	// HTTP handler 解析出 IP 后写入它，Session 在握手时读取，避免同一份解析逻辑分散。
	clientIPHeaderInternal = "X-Socketiox-Client-Ip"
)

// clientIPFromRequest 提取 HTTP 请求的客户端 IP，提取顺序参考 go-zero rest 的
// HTTP 访问日志（httpx.GetRemoteAddr）：优先 X-Forwarded-For，其次连接的对端地址；
// 这里额外兼容只会写 X-Real-Ip 的代理，并去掉端口以便按 IP 归并同一客户端的多次连接。
// 该值仅用于日志定位，不可用于安全校验。
func clientIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if ip := firstForwardedIP(r.Header.Values(clientIPHeaderForwarded)); ip != "" {
		return ip
	}
	if ip := normalizeClientIP(r.Header.Get(clientIPHeaderReal)); ip != "" {
		return ip
	}
	return normalizeClientIP(r.RemoteAddr)
}

// handshakeClientIP 读取 handler 写入的内部头，兼容直接使用 Server.HttpHandler()
// 时缺失内部头的情况，最后回退到握手自身的 Address 字段。
func handshakeClientIP(h engineio.Handshake) string {
	if ip := normalizeClientIP(h.Headers.Get(clientIPHeaderInternal)); ip != "" {
		return ip
	}
	if ip := normalizeClientIP(h.Headers.Get(clientIPHeaderReal)); ip != "" {
		return ip
	}
	if ip := firstForwardedIP(h.Headers.Values(clientIPHeaderForwarded)); ip != "" {
		return ip
	}
	return normalizeClientIP(h.Address)
}

// firstForwardedIP 返回 X-Forwarded-For 中最左侧可解析的 IP。
func firstForwardedIP(values []string) string {
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if ip := normalizeClientIP(part); ip != "" {
				return ip
			}
		}
	}
	return ""
}

// normalizeClientIP 去掉端口与空白并校验为合法 IP，非法内容返回空串，
// 避免把不可信的头内容原样写进日志。
func normalizeClientIP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	raw = strings.Trim(raw, "[]")
	ip := net.ParseIP(raw)
	if ip == nil {
		return ""
	}
	return ip.String()
}
