package imagex

import (
	"bytes"
	"errors"
	"io"
	"strings"

	dsopreaexif "github.com/dsoprea/go-exif/v3"
	log "github.com/dsoprea/go-logging"
)

// ExifTag 是工具层统一后的 EXIF Tag 表示。
// dsoprea/go-exif/v3 解析出的字段较多，这里只保留业务解析需要关注的 IFD 路径、Tag ID、Tag 名称、类型和值。
// 上层代码只依赖这个轻量结构，后续如果替换 EXIF 解析库，不需要把第三方库类型继续向外扩散。
type ExifTag struct {
	IFDPath string
	ID      uint16
	Name    string
	Type    string
	Value   string
}

// ExifTags 是扁平化后的 EXIF Tag 列表。
// EXIF 原始结构按 IFD 分层存储，扁平化后更适合按 TagName 或 TagID 做业务字段提取。
type ExifTags []ExifTag

// FindByName 按 EXIF 标准字段名查找 Tag，例如 BodySerialNumber、DateTimeOriginal。
func (tags ExifTags) FindByName(name string) (ExifTag, bool) {
	for _, tag := range tags {
		if tag.Name == name {
			return tag, true
		}
	}
	return ExifTag{}, false
}

// FindByID 按 EXIF Tag ID 查找 Tag，例如 BodySerialNumber 的 ID 为 0xA431。
func (tags ExifTags) FindByID(id uint16) (ExifTag, bool) {
	for _, tag := range tags {
		if tag.ID == id {
			return tag, true
		}
	}
	return ExifTag{}, false
}

// ExtractExifTagsFromBytes 从图片字节或原始 EXIF 字节中解析全部可识别 Tag。
// 常规上传场景传入的是 JPG/JPEG 头部或完整文件字节，会先通过 SearchAndExtractExif 定位 APP1/EXIF 段。
// 单测或工具调用也可能直接传入原始 EXIF 数据；当 dsoprea 提示未找到 EXIF 包装段时，会把入参当作 raw EXIF 继续解析。
// dsoprea/go-exif/v3 的标准字段表已包含 BodySerialNumber(0xA431)，因此不再需要像旧 goexif 那样手动注册字段。
func ExtractExifTagsFromBytes(data []byte) (ExifTags, error) {
	if len(data) == 0 {
		return nil, nil
	}
	rawExif, err := dsopreaexif.SearchAndExtractExif(data)
	if err != nil {
		if isDsopreaNoExifError(err) {
			rawExif = data
		} else {
			return nil, err
		}
	}
	return extractExifTagsFromRaw(rawExif)
}

// ExtractExifTagsReader 从 Reader 中读取图片或原始 EXIF 数据，并复用字节解析流程。
func ExtractExifTagsReader(reader io.Reader) (ExifTags, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return ExtractExifTagsFromBytes(data)
}

// extractExifTagsFromRaw 将 raw EXIF 数据扁平化为 ExifTags。
// 第三个参数传 true 表示开启 universal search，让 dsoprea 自动遍历 IFD0、ExifIFD、GPS IFD 等子目录。
// 这样 BodySerialNumber 这类位于 ExifIFD 的字段，以及 GPSLatitude/GPSLongitude 这类位于 GPS IFD 的字段，都能被统一收集。
func extractExifTagsFromRaw(rawExif []byte) (ExifTags, error) {
	tags, _, err := dsopreaexif.GetFlatExifDataUniversalSearchWithReadSeeker(bytes.NewReader(rawExif), nil, true)
	if err != nil {
		if isDsopreaNoExifError(err) {
			return nil, nil
		}
		return nil, err
	}

	result := make(ExifTags, 0, len(tags))
	for _, tag := range tags {
		value := selectExifTagValue(tag.Formatted, tag.FormattedFirst)
		if value == "" {
			continue
		}
		result = append(result, ExifTag{
			IFDPath: tag.IfdPath,
			ID:      tag.TagId,
			Name:    tag.TagName,
			Type:    tag.TagTypeName,
			Value:   value,
		})
	}
	return result, nil
}

// selectExifTagValue 选择对业务最友好的 Tag 展示值。
// dsoprea 同时提供 Formatted 和 FormattedFirst：Formatted 会保留完整值，FormattedFirst 只取第一个元素。
// GPS 坐标是度、分、秒三段值，如果误用 FormattedFirst，会只剩下度数，导致坐标解析失败，因此这里优先使用完整的 Formatted。
func selectExifTagValue(formatted, formattedFirst string) string {
	formatted = strings.TrimSpace(formatted)
	formattedFirst = strings.TrimSpace(formattedFirst)
	if formatted == "" {
		return formattedFirst
	}
	return formatted
}

// isDsopreaNoExifError 兼容 dsoprea 直接返回和 go-logging 包装后的 ErrNoExif。
// 没有 EXIF 在图片解析里属于可接受结果，上层会返回空元数据，而不是把它当作业务错误。
func isDsopreaNoExifError(err error) bool {
	return errors.Is(err, dsopreaexif.ErrNoExif) || log.Is(err, dsopreaexif.ErrNoExif)
}
