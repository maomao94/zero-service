package gormx

import (
	"strings"
	"testing"
)

func TestQueryPageNormalizesInvalidParams(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})
	if err := db.Create(&pageTestModel{Name: "a"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var list []pageTestModel
	page, err := QueryPage(db.Model(&pageTestModel{}).Order("id ASC"), 0, 0, &list)
	if err != nil {
		t.Fatalf("query page error = %v", err)
	}
	if page.Page != DefaultPage || page.PageSize != DefaultPageSize {
		t.Fatalf("page params = %d/%d, want %d/%d", page.Page, page.PageSize, DefaultPage, DefaultPageSize)
	}
	if page.Total != 1 || len(page.Data) != 1 {
		t.Fatalf("page data = total %d len %d, want total 1 len 1", page.Total, len(page.Data))
	}
}

func TestCursorPageReturnsNextCursor(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})
	for _, name := range []string{"a", "b", "c"} {
		if err := db.Create(&pageTestModel{Name: name}).Error; err != nil {
			t.Fatalf("create error = %v", err)
		}
	}

	var list []pageTestModel
	page, err := CursorPage(db.Model(&pageTestModel{}), "", 2, "id", &list)
	if err != nil {
		t.Fatalf("cursor page error = %v", err)
	}
	if !page.HasMore {
		t.Fatalf("has more should be true")
	}
	if page.NextCursor != "2" {
		t.Fatalf("next cursor = %q, want 2", page.NextCursor)
	}
	if len(page.Data) != 2 {
		t.Fatalf("data length = %d, want 2", len(page.Data))
	}
}

func TestCursorPageRejectsUnsafeOrderColumn(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})
	var list []pageTestModel

	_, err := CursorPage(db.Model(&pageTestModel{}), "", 2, "id;drop table users", &list)
	if err == nil || !strings.Contains(err.Error(), "invalid cursor order column") {
		t.Fatalf("error = %v, want invalid cursor order column", err)
	}
}
