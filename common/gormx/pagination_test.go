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

func TestQueryPageSelectAllDoesNotBreakCount(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})
	for _, name := range []string{"a", "b"} {
		if err := db.Create(&pageTestModel{Name: name}).Error; err != nil {
			t.Fatalf("create error = %v", err)
		}
	}

	var list []pageTestModel
	page, err := QueryPage(db.Model(&pageTestModel{}).Select("*").Order("id ASC"), 1, 1, &list)
	if err != nil {
		t.Fatalf("query page error = %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("total = %d, want 2", page.Total)
	}
	if len(page.Data) != 1 || page.Data[0].Name != "a" {
		t.Fatalf("page data = %+v, want first row a", page.Data)
	}
}

func TestQueryPageFindSelectPreserved(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})
	if err := db.Create(&pageTestModel{Name: "a"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	var list []pageTestModel
	page, err := QueryPage(db.Model(&pageTestModel{}).Select("name"), 1, 1, &list)
	if err != nil {
		t.Fatalf("query page error = %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("total = %d, want 1", page.Total)
	}
	if len(page.Data) != 1 || page.Data[0].Name != "a" {
		t.Fatalf("page data = %+v, want selected name", page.Data)
	}
	if page.Data[0].ID != 0 {
		t.Fatalf("selected ID = %d, want 0 because find select should remain name-only", page.Data[0].ID)
	}
}

func TestQueryPageDataReturnsData(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})
	for _, name := range []string{"a", "b", "c"} {
		if err := db.Create(&pageTestModel{Name: name}).Error; err != nil {
			t.Fatalf("create error = %v", err)
		}
	}

	got, err := QueryPageData[pageTestModel](db.Model(&pageTestModel{}).Order("id ASC"), 1, 2)
	if err != nil {
		t.Fatalf("query page data error = %v", err)
	}
	if len(got) != 2 || got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("page 1 data = %+v, want [a b]", got)
	}
}

func TestQueryPageDataReturnsSecondPage(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})
	for _, name := range []string{"a", "b", "c"} {
		if err := db.Create(&pageTestModel{Name: name}).Error; err != nil {
			t.Fatalf("create error = %v", err)
		}
	}

	got, err := QueryPageData[pageTestModel](db.Model(&pageTestModel{}).Order("id ASC"), 2, 2)
	if err != nil {
		t.Fatalf("query page data error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "c" {
		t.Fatalf("page 2 data = %+v, want [c]", got)
	}
}

func TestQueryPageDataNormalizesInvalidParams(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})
	if err := db.Create(&pageTestModel{Name: "a"}).Error; err != nil {
		t.Fatalf("create error = %v", err)
	}

	got, err := QueryPageData[pageTestModel](db.Model(&pageTestModel{}).Order("id ASC"), 0, 0)
	if err != nil {
		t.Fatalf("query page data error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("data length = %d, want 1", len(got))
	}
}

func TestQueryPageDataMaxPageSize(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})
	for i := 0; i < MaxPageSize+10; i++ {
		if err := db.Create(&pageTestModel{Name: "x"}).Error; err != nil {
			t.Fatalf("create error = %v", err)
		}
	}

	got, err := QueryPageData[pageTestModel](db.Model(&pageTestModel{}).Order("id ASC"), 1, MaxPageSize+50)
	if err != nil {
		t.Fatalf("query page data error = %v", err)
	}
	if len(got) > MaxPageSize {
		t.Fatalf("data length = %d, want <= %d", len(got), MaxPageSize)
	}
}

func TestQueryPageDataEmptyTable(t *testing.T) {
	db := openTestDB(t, &pageTestModel{})

	got, err := QueryPageData[pageTestModel](db.Model(&pageTestModel{}).Order("id ASC"), 1, 10)
	if err != nil {
		t.Fatalf("query page data error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("data length = %d, want 0", len(got))
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
