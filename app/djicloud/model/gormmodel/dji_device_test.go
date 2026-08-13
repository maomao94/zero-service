package gormmodel

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestDjiDeviceModelsDeviceTypeAndNameSchema(t *testing.T) {
	tests := []struct {
		model any
		name  string
	}{
		{model: &DjiDevice{}, name: "DjiDevice"},
		{model: &DjiDeviceTopo{}, name: "DjiDeviceTopo"},
	}
	for _, tt := range tests {
		parsed, err := schema.Parse(tt.model, &sync.Map{}, schema.NamingStrategy{})
		if err != nil {
			t.Fatalf("parse %s schema error = %v", tt.name, err)
		}
		for fieldName, dataType := range map[string]string{"device_type": "varchar(32)", "device_name": "varchar(128)"} {
			field := parsed.LookUpField(fieldName)
			if field == nil || field.DataType != schema.DataType(dataType) || !field.NotNull || !field.HasDefaultValue || field.DefaultValue != DjiDeviceUnknown {
				t.Fatalf("%s.%s field = %+v, want %s not null default %q", tt.name, fieldName, field, dataType, DjiDeviceUnknown)
			}
		}
	}
}
