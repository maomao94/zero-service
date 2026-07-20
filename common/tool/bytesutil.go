package tool

import (
	"encoding/hex"
	"fmt"

	"github.com/duke-git/lancet/v2/formatter"
)

// HexBytesFormat controls how HexBytes renders raw bytes.
type HexBytesFormat int

const (
	// HexLowerCompact renders bytes like hex.EncodeToString, for example "680e".
	HexLowerCompact HexBytesFormat = iota
	// HexUpperCompact renders compact upper-case hex, for example "680E".
	HexUpperCompact
	// HexUpperSpace renders upper-case bytes separated by spaces, for example "68 0E".
	HexUpperSpace
)

// HexBytes formats raw bytes as hex text.
func HexBytes(raw []byte, format HexBytesFormat) string {
	switch format {
	case HexUpperCompact:
		return fmt.Sprintf("%X", raw)
	case HexUpperSpace:
		return fmt.Sprintf("% X", raw)
	default:
		return hex.EncodeToString(raw)
	}
}

// DecimalBytes formats a byte size using decimal units.
func DecimalBytes(size int64, precision ...int) string {
	return formatter.DecimalBytes(float64(size), precision...)
}

// BinaryBytes formats a byte size using binary units.
func BinaryBytes(size int64, precision ...int) string {
	return formatter.BinaryBytes(float64(size), precision...)
}
