package imagex

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestNormalizeExtraMetaField(t *testing.T) {
	cases := map[string]string{
		"bodySerialNumber":   "bodySerialNumber",
		"body_serial_number": "bodySerialNumber",
		"Body Serial Number": "bodySerialNumber",
		"BodySerialNumber":   "bodySerialNumber",
		"lens-model":         "lensModel",
	}

	for in, want := range cases {
		if got := normalizeExtraMetaField(in); got != want {
			t.Fatalf("normalizeExtraMetaField(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractImageMetaFromBytesWithExtraFieldsReturnsEmptyForNoExif(t *testing.T) {
	got, err := ExtractImageMetaFromBytes([]byte("not exif"), WithExtraMetaFields("bodySerialNumber"))
	if err != nil {
		t.Fatalf("ExtractImageMetaFromBytes() error = %v", err)
	}
	if len(got.Extra) != 0 {
		t.Fatalf("extra meta length = %d, want 0", len(got.Extra))
	}
}

func TestExtractImageMetaFromBytesReadsBodySerialNumberFromExifIFD(t *testing.T) {
	got, err := ExtractImageMetaFromBytes(buildExifWithBodySerialNumber(t, "BODY-123456"), WithExtraMetaFields("Body Serial Number"))
	if err != nil {
		t.Fatalf("ExtractImageMetaFromBytes() error = %v", err)
	}
	if got.Extra["bodySerialNumber"] != "BODY-123456" {
		t.Fatalf("bodySerialNumber = %q, want %q", got.Extra["bodySerialNumber"], "BODY-123456")
	}
}

func TestExtraMetaWalkerBodySerialNumberAlias(t *testing.T) {
	walker := newExtraMetaWalker([]string{"body_serial_number"})
	if !walker.wanted["bodySerialNumber"] {
		t.Fatal("body_serial_number should map to bodySerialNumber")
	}
}

func buildExifWithBodySerialNumber(t *testing.T, bodySerialNumber string) []byte {
	t.Helper()

	var buf bytes.Buffer
	write := func(data any) {
		if err := binary.Write(&buf, binary.LittleEndian, data); err != nil {
			t.Fatalf("write exif data: %v", err)
		}
	}

	buf.WriteString("Exif\x00\x00")
	buf.WriteString("II")
	write(uint16(42))
	write(uint32(8))

	write(uint16(1))
	write(uint16(exifIFDPointerTagID))
	write(uint16(4))
	write(uint32(1))
	write(uint32(26))
	write(uint32(0))

	value := append([]byte(bodySerialNumber), 0)
	valueOffset := uint32(26 + 2 + 12 + 4)

	write(uint16(1))
	write(uint16(bodySerialNumberTagID))
	write(uint16(2))
	write(uint32(len(value)))
	write(valueOffset)
	write(uint32(0))
	buf.Write(value)

	return buf.Bytes()
}
