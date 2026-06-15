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
	// 校验随机字节数范围（避免过短导致冲突，或过长失去"短"的意义）
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

// ------------------------------------------------------
// Token 估算工具
// ------------------------------------------------------

// EstimateTokens 估算文本的 token 数量（近似值）
// 规则：
//   - 中文字符：约 2 tokens/字符
//   - 英文单词：约 1.3 tokens/单词
//   - 标点/空格：约 0.25 tokens/字符
//
// 注意：这是粗略估算，实际 token 数因模型而异。如需精确值请使用 tiktoken 等库。
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	var tokenCount float64
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case r >= 0x4E00 && r <= 0x9FFF: // CJK Unified Ideographs (中文)
			tokenCount += 2.0
		case r >= 0x3000 && r <= 0x303F: // CJK Symbols and Punctuation
			tokenCount += 0.25
		case r >= 0xFF00 && r <= 0xFFEF: // Halfwidth and Fullwidth Forms
			tokenCount += 2.0
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z': // English letters
			// 尝试计算完整单词
			wordLen := 1
			for i+1 < len(runes) && ((runes[i+1] >= 'a' && runes[i+1] <= 'z') || (runes[i+1] >= 'A' && runes[i+1] <= 'Z') || (runes[i+1] >= '0' && runes[i+1] <= '9')) {
				wordLen++
				i++
			}
			tokenCount += 1.3 * float64(wordLen)
			continue
		case r == ' ', r == '\t', r == '\n', r == '\r': // Whitespace
			tokenCount += 0.25
		case r < 128: // ASCII punctuation/symbols
			tokenCount += 0.25
		default: // Other characters (emoji, etc.)
			tokenCount += 2.0
		}
	}

	return int(tokenCount)
}

// EstimateMessagesTokens 估算消息列表的总 token 数（包含消息格式开销）
// 消息格式开销约 4 tokens/条（role + content wrapper）
func EstimateMessagesTokens(messages []string) int {
	total := 0
	for _, msg := range messages {
		total += 4 + EstimateTokens(msg) // 4 tokens overhead per message
	}
	return total
}

// CountSignificantDigits 统计数值字符串的有效数字位数。
// 规则：去掉符号、前导零、小数点后统计剩余数字个数。
// "51.88" -> 4, "0.001234" -> 4, "100" -> 3, "0" -> 0.
func CountSignificantDigits(s string) int {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "+-")
	if idx := strings.IndexAny(s, "eE"); idx >= 0 {
		s = s[:idx]
	}
	s = strings.TrimLeft(s, "0")
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i] + s[i+1:]
	}
	s = strings.TrimLeft(s, "0")
	return len(s)
}
