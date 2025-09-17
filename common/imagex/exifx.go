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

// ImageMeta 图片元数据结构体
type ImageMeta struct {
	Longitude   float64 `json:"longitude"`   // 经度
	Latitude    float64 `json:"latitude"`    // 纬度
	Time        string  `json:"time"`        // 拍摄时间
	ImgHeight   int     `json:"imgHeight"`   // 高
	ImgWidth    int     `json:"imgWidth"`    // 宽
	Altitude    float64 `json:"altitude"`    // 高度
	CameraModel string  `json:"cameraModel"` // 相机型号
}

//func main() {
//	var imgPath string
//	if len(os.Args) > 1 {
//		imgPath = os.Args[1]
//	} else {
//		imgPath = "test.jpg" // 默认测试图片路径
//		fmt.Println("使用默认测试图片路径: test.jpg")
//		fmt.Println("提示: 可指定图片路径运行: go run image_meta.go [图片路径]")
//	}
//
//	meta, err := ExtractImageMeta(imgPath)
//	if err != nil {
//		fmt.Printf("❌ 提取失败: %v\n", err)
//		return
//	}
//
//	// 模拟弹窗效果展示结果
//	printMetadataAsPopup(imgPath, meta)
//}

// 模拟弹窗效果展示元数据
func printMetadataAsPopup(imgPath string, meta ImageMeta) {
	// 计算最长标签名长度，用于对齐
	maxLabelLen := 12
	labels := []string{"经纬度", "拍摄时间", "尺寸", "海拔", "相机型号", "镜头型号", "机身序列号"}
	for _, l := range labels {
		if len(l) > maxLabelLen {
			maxLabelLen = len(l)
		}
	}

	// 打印边框
	border := strings.Repeat("=", 60)
	fmt.Println(border)
	fmt.Printf("📷 图片元数据解析结果 - %s\n", filepath.Base(imgPath))
	fmt.Println(border)

	// 格式化输出各字段
	fmt.Printf("📍 %-*s: (%.6f, %.6f)\n", maxLabelLen, "经纬度", meta.Latitude, meta.Longitude)
	fmt.Printf("⏰ %-*s: %s\n", maxLabelLen, "拍摄时间", meta.Time)
	fmt.Printf("📐 %-*s: %dx%d 像素\n", maxLabelLen, "尺寸", meta.ImgWidth, meta.ImgHeight)
	fmt.Printf("🔼 %-*s: %.2f 米\n", maxLabelLen, "海拔", meta.Altitude)
	fmt.Printf("📌 %-*s: %s\n", maxLabelLen, "相机型号", meta.CameraModel)
	//fmt.Printf("🔍 %-*s: %s\n", maxLabelLen, "镜头型号", meta.LensModel)
	//fmt.Printf("🔢 %-*s: %s\n", maxLabelLen, "机身序列号", meta.BodySerialNumber)

	fmt.Println(border)
	fmt.Println("✅ 解析完成")
}

type Walker struct{}

func (_ Walker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	data, _ := tag.MarshalJSON()
	fmt.Printf("    %v: %v\n", name, string(data))
	return nil
}

func ExtractImageMetaFromBytes(data []byte) (meta ImageMeta, err error) {
	reader := bytes.NewReader(data)
	return ExtractImageMetaReader(reader)
}

func ExtractImageMetaReader(reader io.Reader) (meta ImageMeta, err error) {
	// 解析EXIF
	exif.RegisterParsers(mknote.All...)
	x, err := exif.Decode(reader)
	if err != nil {
		// 无EXIF数据时返回默认值
		if strings.Contains(err.Error(), "no exif data") {
			return meta, nil
		}
		return meta, fmt.Errorf("解析EXIF失败: %w", err)
	}

	// 提取拍摄时间（处理带双引号的情况）
	if dt, err := x.Get(exif.DateTimeOriginal); err == nil {
		cleanTime := strings.Trim(dt.String(), "\"")
		if t, err := time.Parse("2006:01:02 15:04:05", cleanTime); err == nil {
			meta.Time = t.Format("2006-01-02 15:04:05")
		} else {
			meta.Time = cleanTime // 保留清洗后的原始格式
		}
	}

	// 提取经纬度（优化处理逻辑）
	meta.Latitude, err = parseGPSCoordinate(x, exif.GPSLatitude, exif.GPSLatitudeRef)
	if err != nil {
		fmt.Printf("⚠️ 解析纬度警告: %v\n", err)
	}

	meta.Longitude, err = parseGPSCoordinate(x, exif.GPSLongitude, exif.GPSLongitudeRef)
	if err != nil {
		fmt.Printf("⚠️ 解析经度警告: %v\n", err)
	}

	// 提取海拔
	if alt, err := x.Get(exif.GPSAltitude); err == nil {
		if altVal, ok := parseFraction(alt.String()); ok {
			if ref, err := x.Get(exif.GPSAltitudeRef); err == nil && strings.Trim(ref.String(), "\"") == "1" {
				altVal = -altVal
			}
			meta.Altitude = altVal
		}
	}

	// 提取宽度
	if width, err := x.Get(exif.ImageWidth); err == nil {
		if val, ok := parseFractionToInt(width.String()); ok {
			meta.ImgWidth = val
		}
	} else if width, err := x.Get(exif.PixelXDimension); err == nil {
		if val, ok := parseFractionToInt(width.String()); ok {
			meta.ImgWidth = val
		}
	}

	// 提取高度
	if height, err := x.Get(exif.ImageLength); err == nil {
		if val, ok := parseFractionToInt(height.String()); ok {
			meta.ImgHeight = val
		}
	} else if height, err := x.Get(exif.PixelYDimension); err == nil {
		if val, ok := parseFractionToInt(height.String()); ok {
			meta.ImgHeight = val
		}
	}
	// 提取相机信息（去引号处理）
	//if bodySN, err := x.Get(exif.MakerNote); err == nil {
	//	meta.BodySerialNumber = strings.Trim(bodySN.String(), "\"")
	//}
	if model, err := x.Get(exif.Model); err == nil {
		meta.CameraModel = strings.Trim(model.String(), "\"")
	}
	//if lens, err := x.Get(exif.LensModel); err == nil {
	//	meta.LensModel = strings.Trim(lens.String(), "\"")
	//}

	return meta, nil
}

// ExtractImageMeta 提取图片元数据
func ExtractImageMeta(imgPath string) (meta ImageMeta, err error) {
	// 检查文件格式
	ext := strings.ToLower(filepath.Ext(imgPath))
	if ext != ".jpg" && ext != ".jpeg" {
		return meta, errors.New("仅支持JPG/JPEG文件")
	}

	// 打开文件
	f, err := os.Open(imgPath)
	if err != nil {
		return meta, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()
	return ExtractImageMetaReader(f)
}

// parseGPSCoordinate 优化的经纬度解析函数，专门处理["36/1","40/1","342293/10000"]格式
func parseGPSCoordinate(x *exif.Exif, coordTag, refTag exif.FieldName) (float64, error) {
	// 获取坐标原始值
	coordVal, err := x.Get(coordTag)
	if err != nil {
		return 0, fmt.Errorf("获取坐标值失败: %w", err)
	}

	// 获取参考方向（N/S/E/W）
	refVal, err := x.Get(refTag)
	if err != nil {
		return 0, fmt.Errorf("获取参考方向失败: %w", err)
	}

	// 1. 清洗原始数据：去除引号、方括号和前后空格
	cleanCoord := strings.Trim(coordVal.String(), "\"[] ")
	cleanRef := strings.Trim(refVal.String(), "\" ")

	// 2. 拆分度分秒组件（支持逗号或空格分隔）
	var parts []string
	if strings.Contains(cleanCoord, ",") {
		// 处理数组格式："36/1","40/1","342293/10000" → 拆分为["36/1", "40/1", "342293/10000"]
		parts = strings.Split(cleanCoord, ",")
		// 进一步清洗每个组件的引号和空格
		for i, part := range parts {
			parts[i] = strings.Trim(part, "\" ")
		}
	} else {
		// 处理空格分隔格式：36/1 40/1 34.2293
		parts = strings.Fields(cleanCoord)
	}

	// 验证组件数量是否为3（度、分、秒）
	if len(parts) != 3 {
		return 0, fmt.Errorf("无效的度分秒格式，需包含3个组件，实际得到 %d 个", len(parts))
	}

	// 3. 解析度、分、秒（带详细错误信息）
	deg, ok1 := parseFraction(parts[0])
	if !ok1 {
		return 0, fmt.Errorf("度解析失败: %s（格式应为数字或分数，如36或36/1）", parts[0])
	}

	min, ok2 := parseFraction(parts[1])
	if !ok2 {
		return 0, fmt.Errorf("分解析失败: %s（格式应为数字或分数，如40或40/1）", parts[1])
	}

	sec, ok3 := parseFraction(parts[2])
	if !ok3 {
		return 0, fmt.Errorf("秒解析失败: %s（格式应为数字或分数，如34.2293或342293/10000）", parts[2])
	}

	// 4. 转换为十进制坐标
	decimal := deg + min/60 + sec/3600

	// 5. 根据参考方向调整正负值
	switch cleanRef {
	case "S", "W":
		decimal = -decimal
	case "N", "E":
		// 保持正值
	default:
		return 0, fmt.Errorf("无效的参考方向: %s（应为N/S/E/W）", cleanRef)
	}

	return mathutil.RoundToFloat(decimal, 6), nil
}

// parseFraction 解析分数格式（如"36/1"、"342293/10000"）或普通数字（如"34.2293"）
func parseFraction(s string) (float64, bool) {
	// 去除可能残留的引号和空格
	s = strings.Trim(s, "\" ")

	// 处理纯数字格式
	if !strings.Contains(s, "/") {
		val, err := strconv.ParseFloat(s, 64)
		return val, err == nil
	}

	// 处理分数格式
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, false
	}

	num, err1 := strconv.ParseFloat(parts[0], 64)
	den, err2 := strconv.ParseFloat(parts[1], 64)

	// 防止除零错误
	if err1 != nil || err2 != nil || den == 0 {
		return 0, false
	}

	return num / den, true
}

// parseFractionToInt 解析分数格式为整数（如"5184/1" → 5184）
func parseFractionToInt(s string) (int, bool) {
	val, ok := parseFraction(s)
	if !ok {
		return 0, false
	}
	return int(val + 0.5), true // 四舍五入
}
