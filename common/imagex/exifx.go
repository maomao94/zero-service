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
)

const (
	bodySerialNumberTagID = 0xA431
	exifIFDPointerTagID   = 0x8769
)

// ImageMeta 是业务层对外返回的图片元数据摘要。
// 这些字段来自 EXIF 中相对稳定、业务常用的标准 Tag：GPS 坐标、拍摄时间、像素尺寸、海拔、相机型号和机身序列号。
// BodySerialNumber 对应 EXIF 标准 Tag 0xA431，虽然它位于 ExifIFD 子目录，但业务上已经是固定需要的图片字段，因此默认平铺解析。
// Extra 只保存调用方通过 WithExtraMetaFields 显式指定的其他扩展 Tag，避免把完整 EXIF 原始信息暴露到接口响应中。
type ImageMeta struct {
	Longitude        float64           `json:"longitude"`
	Latitude         float64           `json:"latitude"`
	Time             string            `json:"time"`
	ImgHeight        int               `json:"imgHeight"`
	ImgWidth         int               `json:"imgWidth"`
	Altitude         float64           `json:"altitude"`
	CameraModel      string            `json:"cameraModel"`
	BodySerialNumber string            `json:"bodySerialNumber"`
	Extra            map[string]string `json:"extra"`
}

// ImageMetaOption 控制图片元数据解析行为，例如按需补充指定 EXIF Tag。
type ImageMetaOption func(*imageMetaOptions)

type imageMetaOptions struct {
	extraFields []string
}

// WithExtraMetaFields 指定额外需要提取的 EXIF Tag 名称。
// 新实现基于 dsoprea/go-exif/v3 的标准字段表解析，BodySerialNumber(0xA431) 已被内置识别。
// 支持原始 Tag 名称和常见调用参数格式，例如 BodySerialNumber、Body Serial Number、body_serial_number。
// 返回结果会统一使用 lowerCamelCase 作为 Extra 的 key，例如 bodySerialNumber。
func WithExtraMetaFields(fields ...string) ImageMetaOption {
	return func(opts *imageMetaOptions) {
		opts.extraFields = append(opts.extraFields, fields...)
	}
}

// ExtractImageMetaFromBytes 从图片头部、完整图片字节或原始 EXIF 字节中解析元数据。
// 流式上传场景只会捕获文件头部，只要头部包含完整 APP1/EXIF 段，就可以完成基础元数据解析。
func ExtractImageMetaFromBytes(data []byte, options ...ImageMetaOption) (ImageMeta, error) {
	return ExtractImageMetaReader(bytes.NewReader(data), options...)
}

// ExtractImageMetaReader 从 Reader 中解析图片 EXIF 元数据。
// 这里先调用底层解析器把 IFD0、ExifIFD、GPS IFD 等分散在不同目录里的 Tag 扁平化为 ExifTags。
// 再从这些标准 Tag 中组装业务需要的 ImageMeta：经纬度、拍摄时间、图片宽高、海拔、相机型号、机身序列号和按需扩展字段。
func ExtractImageMetaReader(reader io.Reader, options ...ImageMetaOption) (ImageMeta, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return ImageMeta{}, err
	}
	tags, err := ExtractExifTagsFromBytes(data)
	if err != nil {
		return ImageMeta{}, fmt.Errorf("解析EXIF失败: %w", err)
	}
	if len(tags) == 0 {
		return ImageMeta{}, nil
	}

	opts := buildImageMetaOptions(options...)
	meta := ImageMeta{}
	fillTakenTime(&meta, tags)
	fillGPS(&meta, tags)
	fillAltitude(&meta, tags)
	fillSize(&meta, tags)
	fillCameraModel(&meta, tags)
	fillBodySerialNumber(&meta, tags)
	fillExtra(&meta, tags, opts.extraFields)

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

// fillTakenTime 优先读取 DateTimeOriginal，缺失时退回 DateTime。
// EXIF 时间通常是 2006:01:02 15:04:05 格式，对外统一转换为 2006-01-02 15:04:05。
func fillTakenTime(meta *ImageMeta, tags ExifTags) {
	tag, ok := firstExifTagByName(tags, "DateTimeOriginal", "DateTime")
	if !ok {
		return
	}
	cleanTime := strings.Trim(tag.Value, "\"")
	if t, err := time.Parse("2006:01:02 15:04:05", cleanTime); err == nil {
		meta.Time = t.Format("2006-01-02 15:04:05")
		return
	}
	meta.Time = cleanTime
}

// fillGPS 从 GPSLatitude/GPSLongitude 和对应 Ref 字段中解析十进制度坐标。
// GPS 原始值是度、分、秒三段，Ref 决定方向：N/E 为正，S/W 为负。
func fillGPS(meta *ImageMeta, tags ExifTags) {
	if latitude, err := parseGPSCoordinate(tags, "GPSLatitude", "GPSLatitudeRef"); err == nil {
		meta.Latitude = latitude
	}
	if longitude, err := parseGPSCoordinate(tags, "GPSLongitude", "GPSLongitudeRef"); err == nil {
		meta.Longitude = longitude
	}
}

// fillAltitude 解析 GPSAltitude，并根据 GPSAltitudeRef 修正海拔正负。
// GPSAltitude 在 EXIF 里是 RATIONAL 类型，原始值可能是 135976/1000。
// dsoprea 的 Formatted 会尽量保留完整展示值，单值字段也可能被格式化成 [135.976] 或 [135976/1000]。
// 所以这里不直接 ParseFloat，而是交给 parseFraction 统一清洗括号、引号、分数格式后再转成 float64。
// GPSAltitudeRef 为 1 表示低于海平面；它本身也可能被格式化成 [1]，因此同样使用 parseFraction 解析后判断。
func fillAltitude(meta *ImageMeta, tags ExifTags) {
	alt, ok := firstExifTagByName(tags, "GPSAltitude")
	if !ok {
		return
	}
	altVal, ok := parseFraction(alt.Value)
	if !ok {
		return
	}
	if ref, ok := firstExifTagByName(tags, "GPSAltitudeRef"); ok {
		if refVal, ok := parseFraction(ref.Value); ok && refVal == 1 {
			altVal = -altVal
		}
	}
	meta.Altitude = altVal
}

// fillSize 读取图片像素尺寸。
// 不同设备写入像素尺寸的位置不完全一致：
// ImageWidth/ImageLength 常见于 IFD0，PixelXDimension/PixelYDimension 和 ExifImageWidth/ExifImageLength 常见于 ExifIFD。
// 这里按“图片主尺寸字段优先、EXIF 像素尺寸兜底”的顺序读取，避免不同相机、手机、航拍设备的字段差异导致宽高为 0。
// dsoprea 对 SHORT/LONG 单值也可能输出 [5184] 这种格式，最终会通过 parseFractionToInt 统一清洗后转成 int。
func fillSize(meta *ImageMeta, tags ExifTags) {
	if width, ok := firstTagInt(tags, "ImageWidth", "PixelXDimension", "ExifImageWidth"); ok {
		meta.ImgWidth = width
	}
	if height, ok := firstTagInt(tags, "ImageLength", "PixelYDimension", "ExifImageLength"); ok {
		meta.ImgHeight = height
	}
}

// fillCameraModel 读取相机型号，对应 EXIF Model 字段。
func fillCameraModel(meta *ImageMeta, tags ExifTags) {
	model, ok := firstExifTagByName(tags, "Model")
	if !ok {
		return
	}
	meta.CameraModel = cleanExifStringValue(model.Value)
}

// fillBodySerialNumber 默认提取机身序列号，对应 EXIF 标准 Tag BodySerialNumber(0xA431)。
// 这个字段在目录结构上属于 ExifIFD，不在 IFD0；底层已经通过 universal search 扁平化所有 IFD，所以这里直接按标准 TagName 读取。
// 业务返回需要平铺字段，不再要求调用方通过 option 指定；option 仍保留给其他临时扩展字段使用。
func fillBodySerialNumber(meta *ImageMeta, tags ExifTags) {
	tag, ok := firstExifTagByName(tags, "BodySerialNumber")
	if !ok {
		return
	}
	meta.BodySerialNumber = cleanExifStringValue(tag.Value)
}

// fillExtra 只提取调用方指定的 Tag，避免把全部 EXIF 原始数据暴露到响应中。
// 匹配时会把调用参数和 EXIF TagName 都归一化为 lowerCamelCase，因此 Body Serial Number、BodySerialNumber、body_serial_number 都能匹配。
// Extra 的 key 也沿用归一化后的字段名，方便接口调用方稳定读取，例如 bodySerialNumber。
func fillExtra(meta *ImageMeta, tags ExifTags, fields []string) {
	wanted := make(map[string]bool, len(fields))
	for _, field := range fields {
		if normalized := normalizeExtraMetaField(field); normalized != "" {
			wanted[normalized] = true
		}
	}
	if len(wanted) == 0 {
		return
	}

	values := make(map[string]string, len(wanted))
	for _, tag := range tags {
		field := normalizeExtraMetaField(tag.Name)
		if !wanted[field] || tag.Value == "" {
			continue
		}
		values[field] = cleanExifStringValue(tag.Value)
	}
	if len(values) > 0 {
		meta.Extra = values
	}
}

// cleanExifStringValue 清理 EXIF 字符串字段的展示外壳。
// dsoprea 对 ASCII 字段通常会返回 BODY-123 这类可直接使用的值，但部分字段可能带引号或首尾空白。
// 字符串字段不同于数值字段，不需要去掉方括号，避免误伤真实序列号内容。
func cleanExifStringValue(value string) string {
	return strings.TrimSpace(strings.Trim(value, "\""))
}

func firstExifTagByName(tags ExifTags, names ...string) (ExifTag, bool) {
	for _, name := range names {
		if tag, ok := tags.FindByName(name); ok {
			return tag, true
		}
	}
	return ExifTag{}, false
}

func firstTagInt(tags ExifTags, names ...string) (int, bool) {
	for _, name := range names {
		tag, ok := tags.FindByName(name)
		if !ok {
			continue
		}
		if val, ok := parseFractionToInt(tag.Value); ok {
			return val, true
		}
	}
	return 0, false
}

// normalizeExtraMetaField 将不同输入格式统一为 lowerCamelCase，用于匹配 EXIF Tag 和响应 Extra key。
// 这里不维护字段名 switch，而是按空格、下划线、短横线拆词，避免后续每新增一个扩展字段都要改 hardcode 映射。
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

// parseGPSCoordinate 解析 GPS 度分秒坐标，并结合 N/S/E/W 方向转换为十进制度。
// dsoprea/go-exif/v3 扁平化后可能输出 [36, 40, 34.23]，也可能输出 36/1 40/1 3423/100 这类分数字符串。
// 因此这里同时兼容逗号分隔和空白分隔，并把每段都交给 parseFraction 统一处理。
func parseGPSCoordinate(tags ExifTags, coordName, refName string) (float64, error) {
	coordVal, ok := firstExifTagByName(tags, coordName)
	if !ok {
		return 0, fmt.Errorf("获取坐标值失败: %s", coordName)
	}

	refVal, ok := firstExifTagByName(tags, refName)
	if !ok {
		return 0, fmt.Errorf("获取参考方向失败: %s", refName)
	}

	cleanCoord := strings.Trim(coordVal.Value, "\"[] ")
	cleanRef := strings.Trim(refVal.Value, "\" ")

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
// EXIF 中很多数值以有理数保存，例如 135976/1000；dsoprea 格式化后也可能已经是 135.976。
// 由于我们在底层优先保留 Formatted 完整值，单值字段也可能带数组外壳，例如 [5184]、[135.976]、[135976/1000]。
// 这里先去掉引号和方括号，再统一解析数字或分数，避免宽高、海拔这类单值字段因为展示格式差异解析成 0。
func parseFraction(s string) (float64, bool) {
	s = strings.Trim(s, "\"[] ")
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
