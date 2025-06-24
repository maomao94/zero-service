package tool

import (
	"github.com/duke-git/lancet/v2/formatter"
	"github.com/shopspring/decimal"
	"os"
	"strings"
)

var oneHundredDecimal decimal.Decimal = decimal.NewFromInt(100)

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
