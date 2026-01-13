package tool

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/formatter"
	"github.com/duke-git/lancet/v2/random"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
)

var oneHundredDecimal decimal.Decimal = decimal.NewFromInt(100)

// Base62 编码实现
var base62Chars = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

// 分转元
func Fen2Yuan(fen int64) float64 {
	y, _ := decimal.NewFromInt(fen).Div(oneHundredDecimal).Truncate(2).Float64()
	return y
}

// 元转分
func Yuan2Fen(yuan float64) int64 {
	f, _ := decimal.NewFromFloat(yuan).Mul(oneHundredDecimal).Truncate(0).Float64()
	return int64(f)
}

func DecimalBytes(size int64, precision ...int) string {
	return formatter.DecimalBytes(float64(size))
}

func MayReplaceLocalhost(host string) string {
	if os.Getenv("IS_DOCKER") != "" {
		return strings.Replace(strings.Replace(host,
			"localhost", "host.docker.internal", 1),
			"127.0.0.1", "host.docker.internal", 1)
	}
	return host
}
func ToProtoBytes(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("input is nil")
	}

	rv := reflect.ValueOf(v)
	var msg proto.Message
	var ok bool
	switch rv.Kind() {
	case reflect.Ptr:
		if rv.IsNil() {
			return nil, fmt.Errorf("nil pointer")
		}
		msg, ok = v.(proto.Message)
	case reflect.Struct:
		msg, ok = rv.Addr().Interface().(proto.Message)
	default:
		ok = false
	}
	if ok {
		return proto.Marshal(msg)
	}
	if rv.Kind() == reflect.Ptr {
		return convertor.ToBytes(rv.Elem().Interface())
	}
	return convertor.ToBytes(v)
}

func GenOssFilename(filename, pathPrefix string) string {
	u, _ := uuid.NewUUID()
	return pathPrefix + "/" + time.Now().Format("20060102") + "/" +
		strings.ReplaceAll(u.String(), "-", "") +
		path.Ext(filename)
}

// SimpleUUID 生成不带 "-" 的 UUID v4
func SimpleUUID() (string, error) {
	uid, err := random.UUIdV4()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(uid, "-", ""), nil
}

func GenSecondTS() int64 {
	return time.Now().Unix() // 示例：1734429580（对应2025-12-17 09:59:40）
}

// 2. 毫秒级时间戳（推荐，int64，范围：1970~2262，低并发无重复）
func GenMilliTS() int64 {
	return time.Now().UnixMilli() // 示例：1734429580020（对应2025-12-17 09:59:40.020）
}

// 3. 微秒级时间戳（超高精度，int64，几乎无重复）
func GenMicroTS() int64 {
	return time.Now().UnixMicro() // 示例：1734429580020123
}

// EncodeBase62 对字节数组进行Base62编码，输出短字符串
func EncodeBase62(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// 用大整数处理字节数据，避免溢出
	num := new(big.Int).SetBytes(data)
	base := big.NewInt(62)
	zero := big.NewInt(0)
	var result []byte

	for num.Cmp(zero) > 0 {
		var rem big.Int
		num.DivMod(num, base, &rem) // 除以62取余数
		result = append([]byte{base62Chars[rem.Int64()]}, result...)
	}

	// 处理数据为0的极端情况（随机数几乎不可能为0）
	if len(result) == 0 {
		return "0"
	}
	return string(result)
}

// ShortPath 生成指定长度的短路径（通过控制随机字节数）
// 参数：randomBytesLen 随机字节数（建议5-8，越小越短但冲突风险越高）
// 返回：短路径字符串、原始唯一标识（十六进制）、错误
func ShortPath(randomBytesLen int) (shortPath string, uniqueID string, err error) {
	// 校验随机字节数范围（避免过短导致冲突，或过长失去“短”的意义）
	if randomBytesLen < 3 || randomBytesLen > 10 {
		return "", "", fmt.Errorf("randomBytesLen 建议3-10，当前 %d", randomBytesLen)
	}

	// 生成指定长度的高质量随机字节（ crypto/rand 比UUID更直接控制长度）
	randomBytes := random.RandBytes(randomBytesLen)
	// 原始唯一标识用十六进制表示（用于存储/溯源，长度为 2*randomBytesLen）
	uniqueID = hex.EncodeToString(randomBytes)

	// Base62编码随机字节，得到短路径
	shortPath = EncodeBase62(randomBytes)

	return shortPath, uniqueID, nil
}

// PrintGoVersion 打印当前Go版本信息
func PrintGoVersion() {
	fmt.Printf("Go Version: %s\n", runtime.Version())
}

// GetCurrentUserId 安全获取当前用户ID，避免CurrentUser为nil时panic
// 如果CurrentUser为nil或UserId为空，返回空字符串
func GetCurrentUserId(currentUser interface{}) string {
	if currentUser == nil {
		return ""
	}

	// 使用反射获取CurrentUser对象的UserId字段
	v := reflect.ValueOf(currentUser)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	userIdField := v.FieldByName("UserId")
	if !userIdField.IsValid() {
		return ""
	}
	switch userIdField.Kind() {
	case reflect.String:
		return userIdField.String()
	default:
		return ""
	}
}

func GetCurrentUserName(currentUser interface{}) string {
	if currentUser == nil {
		return ""
	}
	v := reflect.ValueOf(currentUser)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	userNameField := v.FieldByName("UserName")
	if !userNameField.IsValid() {
		return ""
	}
	switch userNameField.Kind() {
	case reflect.String:
		return userNameField.String()
	default:
		return ""
	}
}
