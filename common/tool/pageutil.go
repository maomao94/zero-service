package tool

// CalculateOffset returns the zero-based pagination offset for the given page and page size.
func CalculateOffset(page, pageSize int64) uint {
	if page < 1 {
		page = 1
	}
	return uint((page - 1) * pageSize)
}
