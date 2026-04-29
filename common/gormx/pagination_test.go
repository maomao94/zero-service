package gormx

import "testing"

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
