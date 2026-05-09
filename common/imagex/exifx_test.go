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
	got, err := ExtractImageMetaFromBytes(buildExifWithBodySerialNumber(t, "BODY-123456"))
	if err != nil {
		t.Fatalf("ExtractImageMetaFromBytes() error = %v", err)
	}
	if got.BodySerialNumber != "BODY-123456" {
		t.Fatalf("BodySerialNumber = %q, want %q", got.BodySerialNumber, "BODY-123456")
	}
}

func TestExtractImageMetaFromBytesKeepsBodySerialNumberExtraCompatibility(t *testing.T) {
	got, err := ExtractImageMetaFromBytes(buildExifWithBodySerialNumber(t, "BODY-ID-123"), WithExtraMetaFields("BodySerialNumber"))
	if err != nil {
		t.Fatalf("ExtractImageMetaFromBytes() error = %v", err)
	}
	if got.BodySerialNumber != "BODY-ID-123" {
		t.Fatalf("BodySerialNumber = %q, want %q", got.BodySerialNumber, "BODY-ID-123")
	}
	if got.Extra["bodySerialNumber"] != "BODY-ID-123" {
		t.Fatalf("bodySerialNumber extra = %q, want %q", got.Extra["bodySerialNumber"], "BODY-ID-123")
	}
}

func TestParseGPSCoordinateSupportsDsopreaFormattedTuple(t *testing.T) {
	tags := ExifTags{
		{Name: "GPSLatitude", Value: "[36, 40, 34.23]"},
		{Name: "GPSLatitudeRef", Value: "N"},
		{Name: "GPSLongitude", Value: "[117, 47, 11.47]"},
		{Name: "GPSLongitudeRef", Value: "E"},
	}

	latitude, err := parseGPSCoordinate(tags, "GPSLatitude", "GPSLatitudeRef")
	if err != nil {
		t.Fatalf("parseGPSCoordinate latitude error = %v", err)
	}
	if latitude != 36.676175 {
		t.Fatalf("latitude = %v, want %v", latitude, 36.676175)
	}

	longitude, err := parseGPSCoordinate(tags, "GPSLongitude", "GPSLongitudeRef")
	if err != nil {
		t.Fatalf("parseGPSCoordinate longitude error = %v", err)
	}
	if longitude != 117.786519 {
		t.Fatalf("longitude = %v, want %v", longitude, 117.786519)
	}
}

func TestFillSizeSupportsDsopreaSingleValueTuple(t *testing.T) {
	meta := ImageMeta{}
	tags := ExifTags{
		{Name: "PixelXDimension", Value: "[5184]"},
		{Name: "PixelYDimension", Value: "[3888]"},
	}

	fillSize(&meta, tags)

	if meta.ImgWidth != 5184 {
		t.Fatalf("ImgWidth = %d, want %d", meta.ImgWidth, 5184)
	}
	if meta.ImgHeight != 3888 {
		t.Fatalf("ImgHeight = %d, want %d", meta.ImgHeight, 3888)
	}
}

func TestFillAltitudeSupportsDsopreaSingleValueTuple(t *testing.T) {
	meta := ImageMeta{}
	tags := ExifTags{
		{Name: "GPSAltitude", Value: "[135.976]"},
	}

	fillAltitude(&meta, tags)

	if meta.Altitude != 135.976 {
		t.Fatalf("Altitude = %v, want %v", meta.Altitude, 135.976)
	}
}

func TestFillAltitudeSupportsBracketedBelowSeaLevelRef(t *testing.T) {
	meta := ImageMeta{}
	tags := ExifTags{
		{Name: "GPSAltitude", Value: "[135.976]"},
		{Name: "GPSAltitudeRef", Value: "[1]"},
	}

	fillAltitude(&meta, tags)

	if meta.Altitude != -135.976 {
		t.Fatalf("Altitude = %v, want %v", meta.Altitude, -135.976)
	}
}

func TestFillSizeSupportsExifImageDimensionNames(t *testing.T) {
	meta := ImageMeta{}
	tags := ExifTags{
		{Name: "ExifImageWidth", Value: "[5184]"},
		{Name: "ExifImageLength", Value: "[3888]"},
	}

	fillSize(&meta, tags)

	if meta.ImgWidth != 5184 {
		t.Fatalf("ImgWidth = %d, want %d", meta.ImgWidth, 5184)
	}
	if meta.ImgHeight != 3888 {
		t.Fatalf("ImgHeight = %d, want %d", meta.ImgHeight, 3888)
	}
}

func TestNormalizeExtraMetaFieldAlias(t *testing.T) {
	if got := normalizeExtraMetaField("body_serial_number"); got != "bodySerialNumber" {
		t.Fatalf("body_serial_number should map to bodySerialNumber, got %q", got)
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
