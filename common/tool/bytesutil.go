package tool

import "github.com/duke-git/lancet/v2/formatter"

// DecimalBytes formats a byte size using decimal units.
func DecimalBytes(size int64, precision ...int) string {
	return formatter.DecimalBytes(float64(size), precision...)
}

// BinaryBytes formats a byte size using binary units.
func BinaryBytes(size int64, precision ...int) string {
	return formatter.BinaryBytes(float64(size), precision...)
}
