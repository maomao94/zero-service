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
}
