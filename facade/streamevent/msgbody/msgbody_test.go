package msgbody

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"zero-service/common/iec104/types"
	"zero-service/facade/streamevent/streamevent"

	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/encoding/protojson"
)

// TestFromJSONFullMessage 校验一条完整报文转换后的每个字段都不丢失、不变形。
// body、metaData 以原始 JSON 文本透传，其余字段逐一比对。
func TestFromJSONFullMessage(t *testing.T) {
	body := &types.SinglePointInfo{
		Ioa: 2001, Value: true, Qds: 1, QdsDesc: "QDS(00000001)[OV]",
		Ov: true, Nt: true, Time: "2026-06-05 14:30:00.000000",
	}
	data := &types.MsgBody{
		MsgId:    "msg-1",
		Host:     "127.0.0.1",
		Port:     2404,
		Asdu:     "M_SP_NA_1",
		TypeId:   1,
		DataType: 0,
		Coa:      1001,
		Body:     body,
		Time:     "2026-06-05 14:30:00.000000",
		MetaData: map[string]any{"stationId": "station_001"},
		Pm: &types.PointMapping{
			DeviceId: "device_123", DeviceName: "测试设备", TdTableType: "yx,yc",
			Ext1: "alarm", Ext2: "e2", Ext3: "e3", Ext4: "e4", Ext5: "e5",
		},
		Headers: map[string]string{"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"},
		TraceId: "4bf92f3577b34da6a3ce929d0e0e4736",
	}

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal types.MsgBody: %v", err)
	}
	got := FromJSON(string(raw))

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	metaDataJSON, err := json.Marshal(data.MetaData)
	if err != nil {
		t.Fatalf("marshal metaData: %v", err)
	}

	if got.BodyRaw != string(bodyJSON) {
		t.Fatalf("bodyRaw 未原样透传\nwant: %s\ngot : %s", bodyJSON, got.BodyRaw)
	}
	if got.MetaDataRaw != string(metaDataJSON) {
		t.Fatalf("metaDataRaw 未原样透传\nwant: %s\ngot : %s", metaDataJSON, got.MetaDataRaw)
	}

	checks := []struct {
		name string
		want any
		got  any
	}{
		{"msgId", data.MsgId, got.MsgId},
		{"host", data.Host, got.Host},
		{"port", int32(data.Port), got.Port},
		{"asdu", data.Asdu, got.Asdu},
		{"typeId", int32(data.TypeId), got.TypeId},
		{"dataType", int32(data.DataType), got.DataType},
		{"coa", uint32(data.Coa), got.Coa},
		{"time", data.Time, got.Time},
		{"traceId", data.TraceId, got.TraceId},
		{"headers", data.Headers, got.Headers},
		{"pm.deviceId", data.Pm.DeviceId, got.GetPm().GetDeviceId()},
		{"pm.deviceName", data.Pm.DeviceName, got.GetPm().GetDeviceName()},
		{"pm.tdTableType", data.Pm.TdTableType, got.GetPm().GetTdTableType()},
		{"pm.ext1", data.Pm.Ext1, got.GetPm().GetExt1()},
		{"pm.ext2", data.Pm.Ext2, got.GetPm().GetExt2()},
		{"pm.ext3", data.Pm.Ext3, got.GetPm().GetExt3()},
		{"pm.ext4", data.Pm.Ext4, got.GetPm().GetExt4()},
		{"pm.ext5", data.Pm.Ext5, got.GetPm().GetExt5()},
	}
	for _, c := range checks {
		if !reflect.DeepEqual(c.want, c.got) {
			t.Errorf("%s 转换不一致: want %#v, got %#v", c.name, c.want, c.got)
		}
	}

	if got.MsgId == "" {
		t.Fatal("msgId 丢失")
	}
}

// TestFromJSONMapsEveryMsgBodyField 以 types.MsgBody 的 JSON 标签为基准，
// 逐个字段构造最小报文，断言 MsgBody 中存在对应字段且取值一致。
//
// 该测试是新增字段的守卫：types.MsgBody 增加字段而未在 FromJSON 中映射，
// 或者 streamevent.MsgBody 缺少对应字段时，都会直接失败。
func TestFromJSONMapsEveryMsgBodyField(t *testing.T) {
	typ := reflect.TypeOf(types.MsgBody{})
	mapped := make(map[string]struct{}, typ.NumField())

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		pbField := tag
		switch tag {
		case "body":
			pbField = "bodyRaw"
		case "metaData":
			pbField = "metaDataRaw"
		}
		mapped[pbField] = struct{}{}

		t.Run(tag, func(t *testing.T) {
			sample := sampleJSONValue(t, f)

			pb := FromJSON(fmt.Sprintf(`{"%s":%s}`, tag, sample))
			out, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(pb)
			if err != nil {
				t.Fatalf("marshal MsgBody: %v", err)
			}

			got := gjson.GetBytes(out, pbField)
			if !got.Exists() {
				t.Fatalf("streamevent.MsgBody 缺少 %s 字段，types.MsgBody.%s 无法映射", pbField, f.Name)
			}

			// bodyRaw、metaDataRaw 在 pb 中是 JSON 文本字符串，直接按文本比对。
			if pbField == "bodyRaw" || pbField == "metaDataRaw" {
				if got.String() != sample {
					t.Fatalf("%s 未原样透传: want %s, got %s", pbField, sample, got.String())
				}
				return
			}

			var wantVal, gotVal any
			if err := json.Unmarshal([]byte(sample), &wantVal); err != nil {
				t.Fatalf("解析样例值 %s: %v", sample, err)
			}
			if err := json.Unmarshal([]byte(got.Raw), &gotVal); err != nil {
				t.Fatalf("解析转换结果 %s: %v", got.Raw, err)
			}
			if !reflect.DeepEqual(wantVal, gotVal) {
				t.Fatalf("%s 取值不一致: want %s, got %s", pbField, sample, got.Raw)
			}
		})
	}

	// 反向检查：protobuf 契约新增了字段，但 types.MsgBody 没有对应 JSON 字段。
	fields := (&streamevent.MsgBody{}).ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		name := fields.Get(i).JSONName()
		if _, ok := mapped[name]; !ok {
			t.Errorf("streamevent.MsgBody 新增了字段 %s，但 types.MsgBody 没有对应 JSON 字段，需要同步", name)
		}
	}
}

// sampleJSONValue 按字段类型生成一个非零样例值，用于验证该字段被正确映射。
// 数值样例按字段序号生成，保证同 kind 的不同字段取值互不相同，字段接错线会被发现。
// 新增字段时如果出现未覆盖的类型，测试会失败并提示补充样例。
func sampleJSONValue(t *testing.T, f reflect.StructField) string {
	t.Helper()
	index := f.Index[len(f.Index)-1]
	switch f.Type.Kind() {
	case reflect.String:
		return fmt.Sprintf("%q", "v-"+f.Name)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.Itoa(100 + index)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.Itoa(200 + index)
	case reflect.Map:
		return `{"k":"v"}`
	case reflect.Interface:
		return `{"ioa":3}`
	case reflect.Ptr:
		if f.Type != reflect.TypeOf(&types.PointMapping{}) {
			t.Fatalf("字段 %s 的指针类型 %s 未覆盖，请补充样例值", f.Name, f.Type)
		}
		return `{"deviceId":"d","deviceName":"n","tdTableType":"t","ext1":"1","ext2":"2","ext3":"3","ext4":"4","ext5":"5"}`
	}
	t.Fatalf("字段 %s 的类型 %s 未覆盖，请补充样例值", f.Name, f.Type)
	return ""
}
