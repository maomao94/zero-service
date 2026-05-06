package gormmodel

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"zero-service/common/gormx"

	"gorm.io/gorm"
)

func TestOssGormModelFindOneByTenantIdOssCode(t *testing.T) {
	ctx := context.Background()
	db, err := gormx.Open("file::memory:?cache=shared&parseTime=true")
	if err != nil {
		t.Fatalf("open sqlite db error = %v", err)
	}
	err = db.AutoMigrate(&Oss{})
	if err != nil {
		t.Fatalf("auto migrate error = %v", err)
	}

	active := &Oss{
		TenantId:   "tenant-a",
		Category:   1,
		OssCode:    "main",
		Endpoint:   "http://127.0.0.1:9000",
		AccessKey:  "access",
		SecretKey:  "secret",
		BucketName: "bucket",
		Status:     2,
	}
	err = db.WithContext(ctx).Create(active).Error
	if err != nil {
		t.Fatalf("create active oss error = %v", err)
	}
	deleted := &Oss{
		TenantId:   "tenant-a",
		Category:   1,
		OssCode:    "deleted",
		Endpoint:   "http://127.0.0.1:9000",
		AccessKey:  "access",
		SecretKey:  "secret",
		BucketName: "bucket",
		Status:     2,
	}
	err = db.WithContext(ctx).Create(deleted).Error
	if err != nil {
		t.Fatalf("create deleted oss error = %v", err)
	}
	err = db.WithContext(ctx).Delete(deleted).Error
	if err != nil {
		t.Fatalf("soft delete oss error = %v", err)
	}

	var got struct {
		Id         int64
		BucketName string
	}
	err = db.WithContext(ctx).Model(&Oss{}).
		Select("id", "bucket_name").
		Where("tenant_id = ? AND oss_code = ?", "tenant-a", "main").
		First(&got).Error
	if err != nil {
		t.Fatalf("find active oss error = %v", err)
	}
	if got.Id != active.Id || got.BucketName != "bucket" {
		t.Fatalf("find active oss = %+v, want id=%d bucket=bucket", got, active.Id)
	}

	var rawCreateTime sql.NullString
	err = db.WithContext(ctx).Raw("SELECT create_time FROM oss WHERE id = ?", active.Id).Scan(&rawCreateTime).Error
	if err != nil {
		t.Fatalf("scan raw create_time error = %v", err)
	}
	if !rawCreateTime.Valid || rawCreateTime.String == "" {
		t.Fatalf("raw create_time = %+v, want valid timestamp string", rawCreateTime)
	}

	var deletedGot struct{ Id int64 }
	err = db.WithContext(ctx).Model(&Oss{}).
		Select("id").
		Where("tenant_id = ? AND oss_code = ?", "tenant-a", "deleted").
		First(&deletedGot).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("find deleted oss error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}
