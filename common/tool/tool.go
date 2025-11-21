package tool

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/formatter"
	"github.com/duke-git/lancet/v2/random"
	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/shopspring/decimal"
	"github.com/uber/h3-go/v4"
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

	// 先判断接口实现，避免漏掉指针类型
	if msg, ok := v.(proto.Message); ok {
		return proto.Marshal(msg)
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("nil pointer")
		}
		return convertor.ToBytes(rv.Elem().Interface())
	} else {
		return convertor.ToBytes(v)
	}
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

// polygonToH3GeoPolygon 将orb.Polygon转换为H3库需要的GeoPolygon格式
// 严格按照用户要求处理多边形结构：
// - ring[0]: 电子围栏外环
// - ring[1...]: 电子围栏的洞
func OrbPolygonToH3GeoPolygon(polygon orb.Polygon) (h3.GeoPolygon, error) {
	var geoPolygon h3.GeoPolygon

	if len(polygon) == 0 {
		return geoPolygon, errors.New("多边形至少包含一个外环")
	}

	// --- 处理外环 ---
	outerRing := polygon[0]
	if len(outerRing) < 3 {
		return geoPolygon, errors.New("外环至少需要3个点")
	}

	geoPolygon.GeoLoop = OrbRingToH3LatLng(outerRing)

	// --- 处理洞 ---
	for i := 1; i < len(polygon); i++ {
		holeRing := polygon[i]
		if len(holeRing) < 3 {
			continue // 忽略无效洞
		}
		hole := OrbRingToH3LatLng(holeRing)
		geoPolygon.Holes = append(geoPolygon.Holes, hole)
	}

	return geoPolygon, nil
}

// isPointsEqual 检查两个orb.Point是否相等（用于验证多边形闭合性）
func IsOrbPointsEqual(p1, p2 orb.Point) bool {
	// 考虑浮点精度问题的坐标比较
	const epsilon = 1e-9
	return math.Abs(p1[0]-p2[0]) < epsilon && math.Abs(p1[1]-p2[1]) < epsilon
}

func OrbRingToH3LatLng(ring orb.Ring) []h3.LatLng {
	if !IsOrbPointsEqual(ring[0], ring[len(ring)-1]) {
		ring = append(ring, ring[0])
	}
	res := make([]h3.LatLng, len(ring))
	for i, pt := range ring {
		res[i] = h3.LatLng{Lat: pt[1], Lng: pt[0]}
	}
	return res
}
