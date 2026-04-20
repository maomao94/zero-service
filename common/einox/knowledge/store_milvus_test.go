package knowledge

import (
	"testing"

	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

func TestFloatVectorDimFromSchema(t *testing.T) {
	sch := entity.NewSchema().
		WithField(entity.NewField().WithName("vec").WithDataType(entity.FieldTypeFloatVector).WithDim(384))
	d, err := floatVectorDimFromSchema(sch, "vec")
	if err != nil || d != 384 {
		t.Fatalf("got dim=%d err=%v", d, err)
	}
	if _, err := floatVectorDimFromSchema(sch, "missing"); err == nil {
		t.Fatal("expected error for missing field")
	}
}
