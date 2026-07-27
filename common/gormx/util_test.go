package gormx

import (
	"math"
	"testing"
)

func TestNewPageParams(t *testing.T) {
	tests := []struct {
		name         string
		page         int64
		pageSize     int64
		wantPage     int64
		wantPageSize int64
	}{
		{name: "defaults", page: 0, pageSize: 0, wantPage: DefaultPage, wantPageSize: DefaultPageSize},
		{name: "negative defaults", page: -1, pageSize: -1, wantPage: DefaultPage, wantPageSize: DefaultPageSize},
		{name: "explicit", page: 2, pageSize: 50, wantPage: 2, wantPageSize: 50},
		{name: "max page size", page: 1, pageSize: MaxPageSize + 1, wantPage: 1, wantPageSize: MaxPageSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPageParams(tt.page, tt.pageSize)
			if got.Page() != tt.wantPage || got.PageSize() != tt.wantPageSize {
				t.Fatalf("page params = %d/%d, want %d/%d", got.Page(), got.PageSize(), tt.wantPage, tt.wantPageSize)
			}
		})
	}
}

func TestPageParamsOffset(t *testing.T) {
	if offset, ok := (PageParams{}).Offset(); !ok || offset != 0 {
		t.Fatalf("zero-value offset = %d/%t, want 0/true", offset, ok)
	}

	offset, ok := NewPageParams(2, 10).Offset()
	if !ok || offset != 10 {
		t.Fatalf("offset = %d/%t, want 10/true", offset, ok)
	}

	if offset, ok := NewPageParams(math.MaxInt64, MaxPageSize).Offset(); ok || offset != 0 {
		t.Fatalf("overflow offset = %d/%t, want 0/false", offset, ok)
	}
}

func TestPageParamsTotalPagesAndContains(t *testing.T) {
	params := NewPageParams(2, 10)
	if got := params.TotalPages(11); got != 2 {
		t.Fatalf("total pages = %d, want 2", got)
	}
	if !params.Contains(11) {
		t.Fatal("page 2 should be inside a two-page result")
	}
	if NewPageParams(3, 10).Contains(11) {
		t.Fatal("page 3 should be outside a two-page result")
	}

	large := NewPageParams(1, MaxPageSize)
	want := int64(math.MaxInt64/MaxPageSize + 1)
	if got := large.TotalPages(math.MaxInt64); got != want {
		t.Fatalf("large total pages = %d, want %d", got, want)
	}
}
