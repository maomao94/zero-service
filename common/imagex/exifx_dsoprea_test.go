package imagex

import "testing"

func TestDsopreaExtractExifTagsFromBytesReadsBodySerialNumber(t *testing.T) {
	tags, err := ExtractExifTagsFromBytes(buildExifWithBodySerialNumber(t, "BODY-DSOPREA-123"))
	if err != nil {
		t.Fatalf("ExtractExifTagsFromBytes() error = %v", err)
	}

	tag, ok := tags.FindByName("BodySerialNumber")
	if !ok {
		t.Fatal("BodySerialNumber tag not found")
	}
	if tag.ID != bodySerialNumberTagID {
		t.Fatalf("BodySerialNumber tag id = 0x%x, want 0x%x", tag.ID, bodySerialNumberTagID)
	}
	if tag.Value != "BODY-DSOPREA-123" {
		t.Fatalf("BodySerialNumber value = %q, want %q", tag.Value, "BODY-DSOPREA-123")
	}
}

func TestDsopreaExtractExifTagsFromBytesReturnsEmptyForNoExif(t *testing.T) {
	tags, err := ExtractExifTagsFromBytes([]byte("not exif"))
	if err != nil {
		t.Fatalf("ExtractExifTagsFromBytes() error = %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("tag length = %d, want 0", len(tags))
	}
}

func TestSelectExifTagValuePrefersFullFormattedValue(t *testing.T) {
	got := selectExifTagValue("[117, 47, 11.47]", "117")
	if got != "[117, 47, 11.47]" {
		t.Fatalf("selectExifTagValue() = %q, want %q", got, "[117, 47, 11.47]")
	}
}
