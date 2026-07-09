package modelxml

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

type parsedModel struct {
	XMLName xml.Name     `xml:""`
	Items   []parsedItem `xml:"Item"`
}

type parsedItem struct {
	Attrs []xml.Attr `xml:",any,attr"`
}

func TestWriteDeviceModelEscapesAttributes(t *testing.T) {
	var buf bytes.Buffer
	err := WriteDeviceModel(&buf, []DevicePointModel{{
		StationName: "500kV变电站",
		StationCode: "Nanwang500KV",
		DeviceID:    "1000001",
		DeviceInfo:  `{"name":"A&B","quote":"\""}`,
		VideoPos:    `[{"device_code":"cam&1"}]`,
	}})
	if err != nil {
		t.Fatalf("WriteDeviceModel() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `<Device_Model>`) || !strings.Contains(out, `</Device_Model>`) {
		t.Fatalf("missing Device_Model root: %s", out)
	}
	for _, want := range []string{`&#34;name&#34;`, `A&amp;B`, `cam&amp;1`} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	model := decodeModel(t, buf.Bytes())
	if model.XMLName.Local != "Device_Model" {
		t.Fatalf("root = %s, want Device_Model", model.XMLName.Local)
	}
	attrs := attrMap(model.Items[0])
	for name, want := range map[string]string{
		"station_name": "500kV变电站",
		"device_id":    "1000001",
		"device_info":  `{"name":"A&B","quote":"\""}`,
		"video_pos":    `[{"device_code":"cam&1"}]`,
		"point_type":   "",
		"label_attri":  "",
	} {
		if got := attrs[name]; got != want {
			t.Fatalf("attr %s = %q, want %q\n%s", name, got, want, out)
		}
	}
}

func TestWriteDeviceModelStreamsItems(t *testing.T) {
	items := make([]DevicePointModel, 128)
	for i := range items {
		items[i].DeviceID = "id"
	}
	if err := WriteDeviceModel(io.Discard, items); err != nil {
		t.Fatalf("WriteDeviceModel() error = %v", err)
	}
}

func TestWriteDeviceModelHasProtocolFieldNames(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteDeviceModel(&buf, []DevicePointModel{{}}); err != nil {
		t.Fatalf("WriteDeviceModel() error = %v", err)
	}
	attrs := attrMap(decodeModel(t, buf.Bytes()).Items[0])
	for _, name := range []string{
		"station_name", "station_code", "area_id", "area_name", "bay_id", "bay_name",
		"main_device_id", "main_device_name", "component_id", "component_name",
		"device_id", "device_name", "device_type", "meter_type", "appearance_type",
		"save_type_list", "recognition_type_list", "phase", "device_info", "data_type",
		"lower_value", "upper_value", "video_pos", "point_type", "label_attri",
	} {
		if _, ok := attrs[name]; !ok {
			t.Fatalf("missing protocol attr %q in %+v", name, attrs)
		}
	}
}

func decodeModel(t *testing.T, data []byte) parsedModel {
	t.Helper()
	var model parsedModel
	if err := xml.Unmarshal(data, &model); err != nil {
		t.Fatalf("generated XML is invalid: %v\n%s", err, data)
	}
	if len(model.Items) == 0 {
		t.Fatalf("generated XML has no Item: %s", data)
	}
	return model
}

func attrMap(item parsedItem) map[string]string {
	attrs := make(map[string]string, len(item.Attrs))
	for _, attr := range item.Attrs {
		attrs[attr.Name.Local] = attr.Value
	}
	return attrs
}

func TestWritePatrolDeviceModelFields(t *testing.T) {
	var buf bytes.Buffer
	err := WritePatrolDeviceModel(&buf, []PatrolDeviceModel{{
		PatrolDeviceName:      "xgrobot",
		PatrolDeviceCode:      "Q1_P484",
		StationName:           "500kV变电站",
		StationCode:           "Nanwang500KV",
		Manufacturer:          "联想",
		Type:                  "1",
		MountPatrolDeviceCode: "nvr001",
	}})
	if err != nil {
		t.Fatalf("WritePatrolDeviceModel() error = %v", err)
	}
	model := decodeModel(t, buf.Bytes())
	if model.XMLName.Local != "PatrolDevice_Model" {
		t.Fatalf("root = %s, want PatrolDevice_Model", model.XMLName.Local)
	}
	attrs := attrMap(model.Items[0])
	for name, want := range map[string]string{
		"patroldevice_name":       "xgrobot",
		"patroldevice_code":       "Q1_P484",
		"station_name":            "500kV变电站",
		"station_code":            "Nanwang500KV",
		"manufacturer":            "联想",
		"type":                    "1",
		"mount_patroldevice_code": "nvr001",
	} {
		if got := attrs[name]; got != want {
			t.Fatalf("attr %s = %q, want %q\n%s", name, got, want, buf.String())
		}
	}
	if strings.Contains(buf.String(), "robots_code") {
		t.Fatalf("unexpected non-standard robots_code field:\n%s", buf.String())
	}
}
