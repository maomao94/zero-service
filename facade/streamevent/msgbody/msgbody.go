// Package msgbody 把 ASDU 的 JSON 报文转换为 streamevent gRPC 契约中的 MsgBody。
//
// 推送侧对 Kafka、MQTT 直接使用 JSON 文本，对 gRPC 需要按字段填充 MsgBody；
// 两处实现既容易写重复、又容易在新增字段时漏映射，因此集中在本包并由测试守卫。
//
// body、metaData 不做结构化解析，分别以原始 JSON 文本写入 BodyRaw、MetaDataRaw，
// 消费方可自行选择 json.Unmarshal 转 map 或 protojson.Unmarshal 转强类型。
package msgbody

import (
	"zero-service/facade/streamevent/streamevent"

	"github.com/tidwall/gjson"
)

// FromJSON 把一条 ASDU JSON 报文转换为 gRPC 通道的 MsgBody。
//
// 入参不是合法 JSON 时返回字段为零值的 MsgBody，不返回错误：与推送侧现有行为一致，
// 避免因单条报文异常打断同批次其余消息的推送。
func FromJSON(raw string) *streamevent.MsgBody {
	result := gjson.Parse(raw)
	item := &streamevent.MsgBody{
		MsgId:       result.Get("msgId").String(),
		Host:        result.Get("host").String(),
		Port:        int32(result.Get("port").Int()),
		Asdu:        result.Get("asdu").String(),
		TypeId:      int32(result.Get("typeId").Int()),
		DataType:    int32(result.Get("dataType").Int()),
		Coa:         uint32(result.Get("coa").Int()),
		BodyRaw:     result.Get("body").Raw,
		Time:        result.Get("time").String(),
		MetaDataRaw: result.Get("metaData").String(),
		TraceId:     result.Get("traceId").String(),
		Headers:     headersFromJSON(result.Get("headers")),
	}
	if pm := result.Get("pm"); pm.Exists() {
		item.Pm = &streamevent.PointMapping{
			DeviceId:    pm.Get("deviceId").String(),
			DeviceName:  pm.Get("deviceName").String(),
			TdTableType: pm.Get("tdTableType").String(),
			Ext1:        pm.Get("ext1").String(),
			Ext2:        pm.Get("ext2").String(),
			Ext3:        pm.Get("ext3").String(),
			Ext4:        pm.Get("ext4").String(),
			Ext5:        pm.Get("ext5").String(),
		}
	}
	return item
}

// FromJSONBatch 逐条转换一批 ASDU JSON 报文，返回顺序与入参一致。
func FromJSONBatch(raws []string) []*streamevent.MsgBody {
	items := make([]*streamevent.MsgBody, 0, len(raws))
	for _, raw := range raws {
		items = append(items, FromJSON(raw))
	}
	return items
}

// headersFromJSON 把 trace 传播头还原为 map；缺失时返回 nil，避免下游拿到空 map。
func headersFromJSON(r gjson.Result) map[string]string {
	if !r.Exists() {
		return nil
	}
	m := make(map[string]string, len(r.Map()))
	for k, v := range r.Map() {
		m[k] = v.String()
	}
	return m
}
