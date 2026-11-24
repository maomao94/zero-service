package logic

import (
	"context"
	"testing"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/gormx"
	"zero-service/model/gormmodel"

	"gorm.io/gorm"
)

func newOssTestServiceContext(t *testing.T) (*gormx.DB, *svc.ServiceContext) {
	t.Helper()
	db, err := gormx.Open("file:" + t.Name() + "?mode=memory&cache=shared&parseTime=true")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if raw, err := db.DB.DB(); err == nil {
			_ = raw.Close()
		}
	})
	if err := db.AutoMigrate(&gormmodel.Oss{}); err != nil {
		t.Fatal(err)
	}
	return db, &svc.ServiceContext{DB: db}
}

func createOssConfig(t *testing.T, db *gormx.DB) gormmodel.Oss {
	t.Helper()
	oss := gormmodel.Oss{
		TenantId:   "t1",
		OssCode:    "code",
		Endpoint:   "old-endpoint",
		AccessKey:  "old-ak",
		BucketName: "old-bucket",
		Remark:     "old-remark",
	}
	oss.Id = "oss-pk"
	if err := db.Create(&oss).Error; err != nil {
		t.Fatal(err)
	}
	return oss
}

// injectOssDeleteAfterLoad 在更新逻辑读完 OSS 配置之后、写回之前，模拟另一路并发删除。
func injectOssDeleteAfterLoad(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	deleted := false
	err := db.Callback().Query().After("gorm:query").Register("test:delete_oss_after_load", func(tx *gorm.DB) {
		if deleted || tx.Statement.Table != "oss" {
			return
		}
		deleted = true
		if err := db.Delete(&gormmodel.Oss{}, "id = ?", id).Error; err != nil {
			t.Errorf("注入并发删除失败: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateOssWritesRequestedColumns(t *testing.T) {
	db, svcCtx := newOssTestServiceContext(t)
	oss := createOssConfig(t, db)

	if _, err := NewUpdateOssLogic(context.Background(), svcCtx).UpdateOss(&file.UpdateOssReq{
		Id:         oss.Id,
		TenantId:   "t1",
		Category:   2,
		OssCode:    "code",
		Endpoint:   "new-endpoint",
		AccessKey:  "new-ak",
		SecretKey:  "new-sk",
		BucketName: "new-bucket",
		AppId:      "new-app",
		Region:     "new-region",
		Remark:     "new-remark",
		Status:     1,
	}); err != nil {
		t.Fatal(err)
	}

	var got gormmodel.Oss
	if err := db.Where("id = ?", oss.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Endpoint != "new-endpoint" || got.AccessKey != "new-ak" || got.BucketName != "new-bucket" ||
		got.Remark != "new-remark" || got.Status != 1 {
		t.Fatalf("更新后的配置不符合预期: %+v", got)
	}
}

func TestUpdateOssOnConcurrentlyDeletedConfigDoesNotResurrect(t *testing.T) {
	db, svcCtx := newOssTestServiceContext(t)
	oss := createOssConfig(t, db)
	injectOssDeleteAfterLoad(t, db.DB, oss.Id)

	if _, err := NewUpdateOssLogic(context.Background(), svcCtx).UpdateOss(&file.UpdateOssReq{
		Id:       oss.Id,
		TenantId: "t1",
		OssCode:  "code",
		Endpoint: "new-endpoint",
	}); err != nil {
		t.Fatalf("UpdateOss() error = %v, want nil（0 行按幂等保存成功处理）", err)
	}

	var got gormmodel.Oss
	if err := db.Unscoped().Where("id = ?", oss.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	// 并发删除的记录不能被更新逻辑复活，也不能被写入新值。
	if !got.Deleted() {
		t.Fatal("并发删除的 OSS 配置被更新逻辑复活")
	}
	if got.Endpoint != "old-endpoint" {
		t.Fatalf("endpoint = %q, want old-endpoint", got.Endpoint)
	}
}

func TestOssDetailAndDeleteAcceptNonNumericID(t *testing.T) {
	db, svcCtx := newOssTestServiceContext(t)
	oss := createOssConfig(t, db)
	ctx := context.Background()

	detail, err := NewOssDetailLogic(ctx, svcCtx).OssDetail(&file.OssDetailReq{Id: oss.Id})
	if err != nil {
		t.Fatalf("OssDetail() error = %v", err)
	}
	if detail.Oss == nil || detail.Oss.Id != oss.Id {
		t.Fatalf("OssDetail() oss = %+v, want id %q", detail.Oss, oss.Id)
	}

	if _, err := NewDeleteOssLogic(ctx, svcCtx).DeleteOss(&file.DeleteOssReq{Id: oss.Id}); err != nil {
		t.Fatalf("DeleteOss() error = %v", err)
	}
	var got gormmodel.Oss
	if err := db.Unscoped().Where("id = ?", oss.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if !got.Deleted() {
		t.Fatal("DeleteOss() 未删除目标记录")
	}
}
