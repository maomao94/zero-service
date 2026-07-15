package tool

import "github.com/shopspring/decimal"

var oneHundredDecimal decimal.Decimal = decimal.NewFromInt(100)

// Fen2Yuan converts cents to yuan.
func Fen2Yuan(fen int64) float64 {
	y, _ := decimal.NewFromInt(fen).Div(oneHundredDecimal).Truncate(2).Float64()
	return y
}

// Yuan2Fen converts yuan to cents.
func Yuan2Fen(yuan float64) int64 {
	f, _ := decimal.NewFromFloat(yuan).Mul(oneHundredDecimal).Truncate(0).Float64()
	return int64(f)
}
