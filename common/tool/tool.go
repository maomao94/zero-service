package tool

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
	"time"
	"zero-service/common/ctxdata"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/formatter"
	"github.com/duke-git/lancet/v2/random"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
)

var oneHundredDecimal decimal.Decimal = decimal.NewFromInt(100)

func stripBearerPrefixFromTokenString(tok string) (string, error) {
	// Should be a bearer token
	if len(tok) > 6 && strings.ToUpper(tok[0:7]) == "BEARER " {
		return tok[7:], nil
	}
	return tok, nil
}

// ParseToken 解析并验证JWT token，支持所有签名算法，与go-zero保持一致
func ParseToken(tokenString string, secrets ...string) (jwt.MapClaims, error) {
	if len(secrets) == 0 {
		return nil, fmt.Errorf("at least one secret is required")
	}
	tokenString, tokenErr := stripBearerPrefixFromTokenString(tokenString)
	if tokenErr != nil {
		return nil, tokenErr
	}
	var lastErr error
	for _, secret := range secrets {
		token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil {
			lastErr = err
			continue
		}
		if token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				return claims, nil
			}
		}
		lastErr = fmt.Errorf("invalid token with secret: %s", secret)
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("invalid token")
}

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
	return formatter.DecimalBytes(float64(size), precision...)
}

func BinaryBytes(size int64, precision ...int) string {
	return formatter.BinaryBytes(float64(size), precision...)
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

func GetCurrentUserId(ctx context.Context, currentUser interface{}) string {
	if userId := ctxdata.GetUserId(ctx); userId != "" {
		return userId
	}
	if currentUser != nil {
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
	return ""
}

func GetCurrentUserName(ctx context.Context, currentUser interface{}) string {
	if userName := ctxdata.GetUserName(ctx); userName != "" {
		return userName
	}
	if currentUser != nil {
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
	return ""
}

func GetCurrentDeptCode(ctx context.Context, currentUser interface{}) string {
	if deptCode := ctxdata.GetDeptCode(ctx); deptCode != "" {
		return deptCode
	}
	if currentUser != nil {
		v := reflect.ValueOf(currentUser)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return ""
		}
		deptField := v.FieldByName("Dept")
		if !deptField.IsValid() {
			return ""
		}
		if deptField.Kind() != reflect.Slice && deptField.Kind() != reflect.Array {
			return ""
		}
		if deptField.Len() == 0 {
			return ""
		}
		firstDept := deptField.Index(0)
		if firstDept.Kind() == reflect.Ptr {
			firstDept = firstDept.Elem()
		}
		deptCodeField := firstDept.FieldByName("DeptCode")
		if !deptCodeField.IsValid() || deptCodeField.Kind() != reflect.String {
			return ""
		}
		return deptCodeField.String()
	}
	return ""
}

func CalculateOffset(page, pageSize int64) uint {
	if page < 1 {
		page = 1
	}
	return uint((page - 1) * pageSize)
}

type BinaryValues struct {
	Hex    []string // 十六进制表示
	Uint   []uint32 // 无符号整数表示
	Int    []int32  // 有符号整数表示
	Bytes  []byte   // 字节数组表示
	Binary []string // 二进制表示
}

// BytesToBinaryValues 将字节数组每两个字节解析成整数（BigEndian）
func BytesToBinaryValues(data []byte) *BinaryValues {
	n := (len(data) + 1) / 2
	hexVals := make([]string, n)
	uintVals := make([]uint32, n)
	intVals := make([]int32, n)
	binaryVals := make([]string, n)

	for i := 0; i < n; i++ {
		var val uint16
		if 2*i+1 < len(data) {
			val = uint16(data[2*i])<<8 | uint16(data[2*i+1])
		} else {
			val = uint16(data[2*i]) << 8
		}
		hexVals[i] = fmt.Sprintf("0x%04X", val)
		uintVals[i] = uint32(val)
		intVals[i] = int32(int16(val))
		binaryVals[i] = fmt.Sprintf("%016b", val)
	}

	return &BinaryValues{
		Hex:    hexVals,
		Uint:   uintVals,
		Int:    intVals,
		Bytes:  data,
		Binary: binaryVals,
	}
}

// BytesToUint16Slice 将字节数组每两个字节解析成 uint16 切片
func BytesToUint16Slice(data []byte) []uint16 {
	n := (len(data) + 1) / 2
	result := make([]uint16, 0, n)
	for i := 0; i < n; i++ {
		var val uint16
		if 2*i+1 < len(data) {
			val = uint16(data[2*i])<<8 | uint16(data[2*i+1])
		} else {
			val = uint16(data[2*i]) << 8
		}
		result = append(result, val)
	}
	return result
}

// BytesToUint32Slice 将字节数组每两个字节解析成 uint32 切片
func BytesToUint32Slice(data []byte) []uint32 {
	uint16Vals := BytesToUint16Slice(data)
	uint32Vals := make([]uint32, len(uint16Vals))
	for i, v := range uint16Vals {
		uint32Vals[i] = uint32(v)
	}
	return uint32Vals
}

// Uint16SliceToHex 将 uint16 切片转换为十六进制字符串切片
func Uint16SliceToHex(values []uint16) []string {
	hexVals := make([]string, len(values))
	for i, v := range values {
		hexVals[i] = fmt.Sprintf("0x%04X", v)
	}
	return hexVals
}

// BytesToBools 将字节数组按位解析成布尔值切片
func BytesToBools(data []byte, quantity int) []bool {
	bools := make([]bool, quantity)
	for i := 0; i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		bools[i] = (data[byteIndex] & (1 << bitIndex)) != 0
	}
	return bools
}

// BoolsToBytes 将布尔值切片打包成字节数组（每 8 位一字节）
func BoolsToBytes(bools []bool) []byte {
	n := (len(bools) + 7) / 8
	data := make([]byte, n)
	for i, b := range bools {
		if b {
			data[i/8] |= 1 << (i % 8)
		}
	}
	return data
}

// Uint32ToBinaryValues 将单个 uint32 值转换为多种表示形式
func Uint32ToBinaryValues(value uint32) *BinaryValues {
	hexVals := []string{fmt.Sprintf("0x%04X", value)}
	uintVals := []uint32{value}
	intVals := []int32{int32(value)}
	binaryVals := []string{fmt.Sprintf("%016b", value)}

	// 计算字节数组表示，只使用低 16 位
	bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(bytes, uint16(value))

	return &BinaryValues{
		Hex:    hexVals,
		Uint:   uintVals,
		Int:    intVals,
		Bytes:  bytes,
		Binary: binaryVals,
	}
}

// Uint32SliceToBinaryValues 将 uint32 切片转换为多种表示形式
func Uint32SliceToBinaryValues(values []uint32) *BinaryValues {
	hexVals := make([]string, len(values))
	uintVals := make([]uint32, len(values))
	intVals := make([]int32, len(values))
	binaryVals := make([]string, len(values))

	// 计算字节数组表示，每个 uint32 值只使用低 16 位
	bytes := make([]byte, len(values)*2)

	for i, v := range values {
		hexVals[i] = fmt.Sprintf("0x%04X", v)
		uintVals[i] = v
		intVals[i] = int32(v)
		binaryVals[i] = fmt.Sprintf("%016b", v)

		// 使用标准库的 BigEndian 方法转换为字节数组
		binary.BigEndian.PutUint16(bytes[2*i:], uint16(v))
	}

	return &BinaryValues{
		Hex:    hexVals,
		Uint:   uintVals,
		Int:    intVals,
		Bytes:  bytes,
		Binary: binaryVals,
	}
}
