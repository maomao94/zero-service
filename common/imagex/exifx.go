package imagex

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/mathutil"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/rwcarlsen/goexif/tiff"
)

const (
	bodySerialNumberTagID = 0xA431
	exifIFDPointerTagID   = 0x8769
	bodySerialNumberField = exif.FieldName("BodySerialNumber")
)

var extraExifFields = map[uint16]exif.FieldName{
	bodySerialNumberTagID: bodySerialNumberField,
}

func init() {
	exif.RegisterParsers(extraExifParser{})
}

// extraExifParser 补充 goexif 当前 fields.go 未内置、但业务需要读取的标准 EXIF Tag。
// goexif 的 Decode 流程会先解析 TIFF/IFD 结构，再依次执行 RegisterParsers 注册的 Parser。
// Parser 需要把关心的 Tag ID 映射成 FieldName，并通过 x.LoadTags 写入 x.main；后续 x.Get/x.Walk 才能读取到这些字段。
// BodySerialNumber 是 EXIF 2.3 标准字段，Tag ID 为 0xA431，Java metadata-extractor 展示名为 Body Serial Number。
// 当前 goexif 版本没有把 0xA431 放进内置 fields 表，所以这里通过自定义 Parser 默认注入。
type extraExifParser struct{}

func (extraExifParser) Parse(x *exif.Exif) error {
	if x == nil || x.Tiff == nil || len(x.Tiff.Dirs) == 0 {
		return nil
	}

	x.LoadTags(x.Tiff.Dirs[0], extraExifFields, false)
	loadExtraExifSubDir(x, exif.ExifIFDPointer, extraExifFields)
	return nil
}

// loadExtraExifSubDir 按 IFD 指针 Tag 读取子 IFD，并把额外字段加载进 goexif 的字段索引。
// EXIF 的目录是分层结构：IFD0 里通常只存相机型号、方向、缩略图指针以及 ExifIFDPointer(0x8769) 等基础信息。
// 大量拍摄参数和机身信息会放在 ExifIFDPointer 指向的 Exif SubIFD 中，BodySerialNumber(0xA431) 就属于这一层。
// goexif 内置解析器也会读取 ExifIFD，但只会加载 fields.go 已声明的 Tag；未声明的 0xA431 会被跳过。
// 因此这里复用 goexif 的 Raw TIFF 数据和字节序，按 offset 解出子 IFD，再用额外字段表补充加载。
func loadExtraExifSubDir(x *exif.Exif, ptr exif.FieldName, fields map[uint16]exif.FieldName) {
	tag, err := x.Get(ptr)
	if err != nil {
		return
	}
	offset, err := tag.Int64(0)
	if err != nil {
		return
	}
	r := bytes.NewReader(x.Raw)
	if _, err = r.Seek(offset, 0); err != nil {
		return
	}
	subDir, _, err := tiff.DecodeDir(r, x.Tiff.Order)
	if err != nil {
		return
	}
	x.LoadTags(subDir, fields, false)
}

type ImageMeta struct {
	Longitude   float64           `json:"longitude"`
	Latitude    float64           `json:"latitude"`
	Time        string            `json:"time"`
	ImgHeight   int               `json:"imgHeight"`
	ImgWidth    int               `json:"imgWidth"`
	Altitude    float64           `json:"altitude"`
	CameraModel string            `json:"cameraModel"`
	Extra       map[string]string `json:"extra"`
}

// ImageMetaOption 控制图片元数据解析行为，例如按需补充指定 EXIF Tag。
type ImageMetaOption func(*imageMetaOptions)

type imageMetaOptions struct {
	extraFields []string
}

// WithExtraMetaFields 指定额外需要提取的 EXIF Tag 名称。
// goexif 默认只加载内置 fields 表中的 Tag，工具包会额外注册 BodySerialNumber(0xA431) 等需要补充的 Tag ID。
// 支持原始 Tag 名称和常见调用参数格式，例如 BodySerialNumber、Body Serial Number、body_serial_number。
// 返回结果会统一使用 lowerCamelCase 作为 Extra 的 key，例如 bodySerialNumber。
func WithExtraMetaFields(fields ...string) ImageMetaOption {
	return func(opts *imageMetaOptions) {
		opts.extraFields = append(opts.extraFields, fields...)
	}
}

// ExtractImageMetaFromBytes 从图片头部或完整图片字节中解析 EXIF 元数据。
func ExtractImageMetaFromBytes(data []byte, options ...ImageMetaOption) (ImageMeta, error) {
	return ExtractImageMetaReader(bytes.NewReader(data), options...)
}

// ExtractImageMetaReader 从 Reader 中解析图片 EXIF 元数据。
// 默认提取经纬度、拍摄时间、图片宽高、海拔、相机型号；通过 WithExtraMetaFields 可补充指定 Tag。
func ExtractImageMetaReader(reader io.Reader, options ...ImageMetaOption) (ImageMeta, error) {
	opts := buildImageMetaOptions(options...)
	exif.RegisterParsers(mknote.All...)
	x, err := exif.Decode(reader)
	if err != nil {
		if isNoExifError(err) {
			return ImageMeta{}, nil
		}
		return ImageMeta{}, fmt.Errorf("解析EXIF失败: %w", err)
	}

	meta := ImageMeta{}
	fillTakenTime(&meta, x)
	fillGPS(&meta, x)
	fillAltitude(&meta, x)
	fillSize(&meta, x)
	fillCameraModel(&meta, x)
	fillExtra(&meta, x, opts.extraFields)

	return meta, nil
}

// ExtractImageMeta 从 JPG/JPEG 文件中解析图片 EXIF 元数据。
func ExtractImageMeta(imgPath string, options ...ImageMetaOption) (ImageMeta, error) {
	ext := strings.ToLower(filepath.Ext(imgPath))
	if ext != ".jpg" && ext != ".jpeg" {
		return ImageMeta{}, errors.New("仅支持JPG/JPEG文件")
	}

	f, err := os.Open(imgPath)
	if err != nil {
		return ImageMeta{}, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	return ExtractImageMetaReader(f, options...)
}

func buildImageMetaOptions(options ...ImageMetaOption) imageMetaOptions {
	var opts imageMetaOptions
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}
	return opts
}

func isNoExifError(err error) bool {
	return err == io.EOF || strings.Contains(err.Error(), "no exif data")
}

func fillTakenTime(meta *ImageMeta, x *exif.Exif) {
	dt, err := x.Get(exif.DateTimeOriginal)
	if err != nil {
		return
	}
	cleanTime := strings.Trim(dt.String(), "\"")
	if t, err := time.Parse("2006:01:02 15:04:05", cleanTime); err == nil {
		meta.Time = t.Format("2006-01-02 15:04:05")
		return
	}
	meta.Time = cleanTime
}

func fillGPS(meta *ImageMeta, x *exif.Exif) {
	if latitude, err := parseGPSCoordinate(x, exif.GPSLatitude, exif.GPSLatitudeRef); err == nil {
		meta.Latitude = latitude
	}
	if longitude, err := parseGPSCoordinate(x, exif.GPSLongitude, exif.GPSLongitudeRef); err == nil {
		meta.Longitude = longitude
	}
}

func fillAltitude(meta *ImageMeta, x *exif.Exif) {
	alt, err := x.Get(exif.GPSAltitude)
	if err != nil {
		return
	}
	altVal, ok := parseFraction(alt.String())
	if !ok {
		return
	}
	if ref, err := x.Get(exif.GPSAltitudeRef); err == nil && strings.Trim(ref.String(), "\"") == "1" {
		altVal = -altVal
	}
	meta.Altitude = altVal
}

func fillSize(meta *ImageMeta, x *exif.Exif) {
	if width, ok := firstTagInt(x, exif.ImageWidth, exif.PixelXDimension); ok {
		meta.ImgWidth = width
	}
	if height, ok := firstTagInt(x, exif.ImageLength, exif.PixelYDimension); ok {
		meta.ImgHeight = height
	}
}

func fillCameraModel(meta *ImageMeta, x *exif.Exif) {
	model, err := x.Get(exif.Model)
	if err != nil {
		return
	}
	meta.CameraModel = strings.Trim(model.String(), "\"")
}

// fillExtra 遍历 EXIF 并只提取调用方指定的 Tag，避免把全部 EXIF 原始数据暴露到响应中。
func fillExtra(meta *ImageMeta, x *exif.Exif, fields []string) {
	walker := newExtraMetaWalker(fields)
	if len(walker.wanted) == 0 {
		return
	}
	if err := x.Walk(walker); err == nil && len(walker.values) > 0 {
		meta.Extra = walker.values
	}
}

func firstTagInt(x *exif.Exif, names ...exif.FieldName) (int, bool) {
	for _, name := range names {
		tag, err := x.Get(name)
		if err != nil {
			continue
		}
		if val, ok := parseFractionToInt(tag.String()); ok {
			return val, true
		}
	}
	return 0, false
}

type extraMetaWalker struct {
	wanted map[string]bool
	values map[string]string
}

func newExtraMetaWalker(fields []string) *extraMetaWalker {
	wanted := make(map[string]bool, len(fields))
	for _, field := range fields {
		if normalized := normalizeExtraMetaField(field); normalized != "" {
			wanted[normalized] = true
		}
	}
	return &extraMetaWalker{
		wanted: wanted,
		values: make(map[string]string, len(wanted)),
	}
}

func (w *extraMetaWalker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	field := normalizeExtraMetaField(string(name))
	if !w.wanted[field] {
		return nil
	}
	if value := cleanExifTagValue(tag); value != "" {
		w.values[field] = value
	}
	return nil
}

// normalizeExtraMetaField 将不同输入格式统一为 lowerCamelCase，用于匹配 EXIF Tag 和响应 Extra key。
func normalizeExtraMetaField(field string) string {
	parts := strings.FieldsFunc(field, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if b.Len() == 0 {
			b.WriteString(strings.ToLower(part[:1]))
			if len(part) > 1 {
				b.WriteString(part[1:])
			}
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}

	return b.String()
}

// cleanExifTagValue 将 EXIF Tag 值转成适合放入响应的字符串。
func cleanExifTagValue(tag *tiff.Tag) string {
	if tag == nil {
		return ""
	}
	if tag.Format() == tiff.StringVal {
		if value, err := tag.StringVal(); err == nil {
			return strings.TrimSpace(strings.Trim(value, "\""))
		}
	}
	return strings.TrimSpace(strings.Trim(tag.String(), "\""))
}

// parseGPSCoordinate 解析 GPS 度分秒坐标，并结合 N/S/E/W 方向转换为十进制度。
func parseGPSCoordinate(x *exif.Exif, coordTag, refTag exif.FieldName) (float64, error) {
	coordVal, err := x.Get(coordTag)
	if err != nil {
		return 0, fmt.Errorf("获取坐标值失败: %w", err)
	}

	refVal, err := x.Get(refTag)
	if err != nil {
		return 0, fmt.Errorf("获取参考方向失败: %w", err)
	}

	cleanCoord := strings.Trim(coordVal.String(), "\"[] ")
	cleanRef := strings.Trim(refVal.String(), "\" ")

	var parts []string
	if strings.Contains(cleanCoord, ",") {
		parts = strings.Split(cleanCoord, ",")
		for i, part := range parts {
			parts[i] = strings.Trim(part, "\" ")
		}
	} else {
		parts = strings.Fields(cleanCoord)
	}

	if len(parts) != 3 {
		return 0, fmt.Errorf("无效的度分秒格式，需包含3个组件，实际得到 %d 个", len(parts))
	}

	deg, ok := parseFraction(parts[0])
	if !ok {
		return 0, fmt.Errorf("度解析失败: %s（格式应为数字或分数，如36或36/1）", parts[0])
	}

	min, ok := parseFraction(parts[1])
	if !ok {
		return 0, fmt.Errorf("分解析失败: %s（格式应为数字或分数，如40或40/1）", parts[1])
	}

	sec, ok := parseFraction(parts[2])
	if !ok {
		return 0, fmt.Errorf("秒解析失败: %s（格式应为数字或分数，如34.2293或342293/10000）", parts[2])
	}

	decimal := deg + min/60 + sec/3600
	switch cleanRef {
	case "S", "W":
		decimal = -decimal
	case "N", "E":
	default:
		return 0, fmt.Errorf("无效的参考方向: %s（应为N/S/E/W）", cleanRef)
	}

	return mathutil.RoundToFloat(decimal, 6), nil
}

// parseFraction 支持 EXIF 常见分数字符串和普通数字字符串。
func parseFraction(s string) (float64, bool) {
	s = strings.Trim(s, "\" ")
	if !strings.Contains(s, "/") {
		val, err := strconv.ParseFloat(s, 64)
		return val, err == nil
	}

	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, false
	}

	num, err1 := strconv.ParseFloat(parts[0], 64)
	den, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil || den == 0 {
		return 0, false
	}

	return num / den, true
}

func parseFractionToInt(s string) (int, bool) {
	val, ok := parseFraction(s)
	if !ok {
		return 0, false
	}
	return int(val), true
}
