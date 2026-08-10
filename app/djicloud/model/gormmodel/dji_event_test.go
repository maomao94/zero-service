package gormmodel

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestDockFlightTaskTrackIdAllowsNullWithoutDatabaseDefault(t *testing.T) {
	models := []struct {
		name  string
		model any
	}{
		{name: "dock flight task", model: &DjiDockFlightTask{}},
		{name: "dock device flight task state", model: &DjiDockDeviceFlightTaskState{}},
	}

	for _, tt := range models {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := schema.Parse(tt.model, &sync.Map{}, schema.NamingStrategy{})
			if err != nil {
				t.Fatalf("parse schema error = %v", err)
			}
			field := parsed.LookUpField("track_id")
			if field == nil {
				t.Fatal("track_id field not found")
			}
			if field.NotNull {
				t.Fatal("track_id must allow null because GaussDB PG treats empty string as null")
			}
			if field.HasDefaultValue {
				t.Fatal("track_id must not depend on database default")
			}
		})
	}
}

func TestDjiHmsAlertStoresTextAlarmIDAndDeviceTypeName(t *testing.T) {
	parsed, err := schema.Parse(&DjiHmsAlert{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse schema error = %v", err)
	}
	if field := parsed.LookUpField("alarm_id"); field == nil || field.DataType != "varchar(64)" || field.TagSettings["TYPE"] != "varchar(64)" {
		t.Fatalf("alarm_id field = %+v, want varchar(64)", field)
	}
	if field := parsed.LookUpField("device_type_name"); field == nil || field.DataType != "varchar(128)" || field.TagSettings["TYPE"] != "varchar(128)" {
		t.Fatalf("device_type_name field = %+v, want varchar(128)", field)
	}
	for _, name := range []string{"message_key", "tid", "bid", "trace_id"} {
		if field := parsed.LookUpField(name); field == nil {
			t.Fatalf("%s field not found", name)
		}
	}
	if field := parsed.LookUpField("gimbal_position"); field != nil {
		t.Fatalf("HMS must not persist protocol-absent gimbal_position field = %+v", field)
	}
	for _, name := range []string{"component_index", "sensor_index", "gimbal_index", "lidar_index", "lte_index"} {
		if field := parsed.LookUpField(name); field != nil {
			t.Fatalf("obsolete persisted field %s = %+v", name, field)
		}
	}
	if field := parsed.LookUpField("imminent"); field == nil || field.TagSettings["COMMENT"] != "是否即时告警" {
		t.Fatalf("imminent field = %+v, want immediacy comment", field)
	}
}
