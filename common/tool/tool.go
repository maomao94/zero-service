package tool

import (
	"context"
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
		lastErr = fmt.Errorf("invalid token")
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

// 字节 → uint16（核心） → int16（有符号）
type BinaryValues struct {
	Hex    []string `json:"hex"`    // 16位十六进制
	Uint16 []uint16 `json:"uint16"` // 核心：无符号16位（唯一真值）
	Int16  []int16  `json:"int16"`  // 有符号16位（由uint16转换）
	Bytes  []byte   `json:"bytes"`  // 原始字节（源头）
	Binary []string `json:"binary"` // 16位二进制
}

// ------------------------------
// 字节 → uint16
// ------------------------------
func BytesToUint16Slice(data []byte) []uint16 {
	n := (len(data) + 1) / 2
	result := make([]uint16, 0, n)

	for i := 0; i < n; i++ {
		idx := i * 2
		if idx+1 < len(data) {
			// 常规组合：高8位 + 低8位
			result = append(result, uint16(data[idx])<<8|uint16(data[idx+1]))
		} else {
			// 奇数长度：最后一个字节补 0
			result = append(result, uint16(data[idx])<<8)
		}
	}
	return result
}

// ------------------------------
// uint16 → 字节
// ------------------------------
func Uint16SliceToBytes(values []uint16) []byte {
	bytes := make([]byte, len(values)*2)
	for i, v := range values {
		bytes[2*i] = byte(v >> 8)
		bytes[2*i+1] = byte(v & 0xFF)
	}
	return bytes
}

// ------------------------------
// uint16 ↔ int16（有符号负数转换）
// ------------------------------
func Uint16ToInt16(u uint16) int16 {
	return int16(u)
}

func Uint16SliceToInt16Slice(values []uint16) []int16 {
	intVals := make([]int16, len(values))
	for i, v := range values {
		intVals[i] = Uint16ToInt16(v)
	}
	return intVals
}

// ------------------------------
// uint16 → uint32 / int32
// 给 grpc 对接用，不污染核心结构
// ------------------------------
func Uint16ToUint32(u uint16) uint32 {
	return uint32(u)
}

func Uint16ToInt32(u uint16) int32 {
	return int32(int16(u))
}

func Uint16SliceToUint32Slice(values []uint16) []uint32 {
	res := make([]uint32, len(values))
	for i, v := range values {
		res[i] = Uint16ToUint32(v)
	}
	return res
}

func Uint16SliceToInt32Slice(values []uint16) []int32 {
	res := make([]int32, len(values))
	for i, v := range values {
		res[i] = Uint16ToInt32(v)
	}
	return res
}

func Int16SliceToInt32Slice(values []int16) []int32 {
	res := make([]int32, len(values))
	for i, v := range values {
		res[i] = int32(v)
	}
	return res
}

// ------------------------------
// 字节 → 完整 BinaryValues
// ------------------------------
func BytesToBinaryValues(data []byte) *BinaryValues {
	uint16Vals := BytesToUint16Slice(data)
	int16Vals := Uint16SliceToInt16Slice(uint16Vals)
	n := len(uint16Vals)

	hexVals := make([]string, n)
	binVals := make([]string, n)

	for i := range uint16Vals {
		val := uint16Vals[i]
		hexVals[i] = fmt.Sprintf("0x%04X", val)
		binVals[i] = fmt.Sprintf("%016b", val)
	}

	return &BinaryValues{
		Hex:    hexVals,
		Uint16: uint16Vals,
		Int16:  int16Vals,
		Bytes:  data,
		Binary: binVals,
	}
}

// ------------------------------
// uint16 数组 → BinaryValues
// ------------------------------
func Uint16SliceToBinaryValues(values []uint16) *BinaryValues {
	int16Vals := Uint16SliceToInt16Slice(values)
	n := len(values)

	hexVals := make([]string, n)
	binVals := make([]string, n)

	for i := range values {
		val := values[i]
		hexVals[i] = fmt.Sprintf("0x%04X", val)
		binVals[i] = fmt.Sprintf("%016b", val)
	}

	return &BinaryValues{
		Hex:    hexVals,
		Uint16: values,
		Int16:  int16Vals,
		Bytes:  Uint16SliceToBytes(values),
		Binary: binVals,
	}
}

// ------------------------------------------------------
// 字节 ↔ 布尔位
// ------------------------------------------------------
func BytesToBools(data []byte, quantity int) []bool {
	bools := make([]bool, quantity)
	for i := 0; i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		bools[i] = (data[byteIndex] & (1 << bitIndex)) != 0
	}
	return bools
}

// ------------------------------------------------------
// 布尔位 ↔ 字节
// ------------------------------------------------------
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
