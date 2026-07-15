package isp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrRootNameMismatch 在服务端校验收到消息的 RootName 与预期不符时返回，用于断开非法连接。
var ErrRootNameMismatch = errors.New("isp: root name mismatch")


// xmlMessage 对应 ISP 协议 XML 报文结构，用于标准库 xml 编解码。
// Command 字段使用 omitempty：Command=0 时省略 <Command> 元素，与 Java 侧行为一致。
type xmlMessage struct {
	XMLName     xml.Name  `xml:""`
	SendCode    string    `xml:"SendCode"`
	ReceiveCode string    `xml:"ReceiveCode"`
	Type        string    `xml:"Type"`
	Code        string    `xml:"Code"`
	Command     string    `xml:"Command,omitempty"`
	Time        string    `xml:"Time"`
	Items       []xmlItem `xml:"Items>Item"`
}

// xmlItem 对应 <Item attr="value"/>，属性列表动态映射。
type xmlItem struct {
	Attrs []xml.Attr `xml:",any,attr"`
}

// BuildXML 将 Message 序列化为 ISP XML 字节（不含帧头尾）。
// rootName 用于覆盖消息中未设置的根元素名称。
func BuildXML(msg *Message, rootName string) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("isp: 空消息")
	}
	root := NormalizeRootName(firstNonEmpty(msg.RootName, rootName))
	cmdStr := ""
	if msg.Command != 0 {
		cmdStr = strconv.FormatInt(int64(msg.Command), 10)
	}
	xm := xmlMessage{
		XMLName:     xml.Name{Local: root},
		SendCode:    msg.SendCode,
		ReceiveCode: msg.ReceiveCode,
		Type:        strconv.FormatInt(int64(msg.Type), 10),
		Code:        msg.Code,
		Command:     cmdStr,
		Time:        msg.Time,
		Items:       make([]xmlItem, 0, len(msg.Items)),
	}
	for _, item := range msg.Items {
		xi := xmlItem{Attrs: make([]xml.Attr, 0, len(item))}
		for k, v := range item {
			xi.Attrs = append(xi.Attrs, xml.Attr{Name: xml.Name{Local: k}, Value: v})
		}
		xm.Items = append(xm.Items, xi)
	}
	buf := &bytes.Buffer{}
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(buf)
	enc.Indent("", "  ")
	if err := enc.Encode(xm); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ParseXML 将 ISP XML 字节解析为 Message。
// 入站消息默认 SessionSource 为服务端，调用方可按需覆盖。
func ParseXML(raw []byte) (*Message, error) {
	var xm xmlMessage
	if err := xml.Unmarshal(raw, &xm); err != nil {
		return nil, err
	}
	typ, err := parseInt32(strings.TrimSpace(xm.Type), "Type")
	if err != nil {
		return nil, err
	}
	cmd, err := parseInt32(strings.TrimSpace(xm.Command), "Command")
	if err != nil {
		return nil, err
	}
	msg := &Message{
		RootName:      xm.XMLName.Local,
		SendCode:      strings.TrimSpace(xm.SendCode),
		ReceiveCode:   strings.TrimSpace(xm.ReceiveCode),
		Type:          typ,
		Code:          strings.TrimSpace(xm.Code),
		Command:       cmd,
		Time:          strings.TrimSpace(xm.Time),
		RawXML:        string(raw),
		SessionSource: SessionSourceServer,
		Items:         make([]Item, 0, len(xm.Items)),
	}
	for _, xi := range xm.Items {
		item := make(Item, len(xi.Attrs))
		for _, attr := range xi.Attrs {
			item[attr.Name.Local] = attr.Value
		}
		msg.Items = append(msg.Items, item)
	}
	return msg, nil
}

// ValidateRootName 校验收到的 XML 根元素名称是否与预期一致。
// 对标 Java 侧 SipHandlerInterceptor 中 rootNodeName 的校验逻辑。
func ValidateRootName(expected, actual string) error {
	if actual == "" {
		return nil
	}
	if !IsValidRootName(actual) {
		return fmt.Errorf("%w: 不支持的根元素 %q", ErrRootNameMismatch, actual)
	}
	if NormalizeRootName(expected) != NormalizeRootName(actual) {
		return fmt.Errorf("%w: 期望 %q, 实际 %q", ErrRootNameMismatch, expected, actual)
	}
	return nil
}

// IsValidRootName 校验根元素是否为合法的 PatrolHost 或 PatrolDevice。
func IsValidRootName(root string) bool {
	switch root {
	case RootPatrolHost, RootPatrolDevice:
		return true
	default:
		return false
	}
}

// parseInt32 将字段字符串转为 int32，空字符串返回 0。
func parseInt32(s, field string) (int32, error) {
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("isp: 无效字段 %s=%q: %w", field, s, err)
	}
	return int32(v), nil
}
