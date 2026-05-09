package logic

import (
	"context"
	"os"
	"path/filepath"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/timex"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/config"
	"zero-service/common/filex"
	"zero-service/common/imagex"
	"zero-service/common/ossx"
	"zero-service/common/tool"
)

// resolveContentType 解析内容类型。若调用方已显式指定则直接返回，否则基于文件头前 512 字节
// 探测 MIME 类型，并在探测成功时记录日志。
func resolveContentType(ctx context.Context, contentType string, head []byte) string {
	detected := ossx.DetectContentType(contentType, head)
	if contentType == "" && detected != "" {
		logx.WithContext(ctx).Infof("Detected Content-Type: %s", detected)
	}
	return detected
}

// buildCaptureOptions 基于上传配置构建 CaptureOptions，不依赖运行时 ContentType 探测。
// 缩略图启用时始终创建临时文件（非图片在上传后立即清理）；头字节 MaxHeadRead 始终按配置设置
// 以支持 EXIF 提取。
func buildCaptureOptions(uploadConf config.UploadConf, isThumb bool) filex.CaptureOptions {
	thumbEnabled := uploadConf.Image.Thumb.Enabled &&
		uploadConf.Image.Thumb.Width > 0 &&
		uploadConf.Image.Thumb.Height > 0
	return filex.CaptureOptions{
		TempDir:     uploadConf.TempDir,
		TempPattern: "upload-*",
		NeedTemp:    thumbEnabled && isThumb,
		MaxHeadRead: uploadConf.Image.MaxExifRead,
	}
}

// processUploadResult 根据 ossx.UploadStream 的结果构建 gRPC 响应 File。
// 完成 pbFile 字段映射、EXIF 元数据提取、异步缩略图调度。
func processUploadResult(
	ctx context.Context,
	uploadConf config.UploadConf,
	result *ossx.StreamUploadResult,
	ossTemplate ossx.OssTemplate,
	tenantID, bucketName, filename string,
	isThumb bool,
	thumbTaskRunner *threading.TaskRunner,
) *file.File {
	if result.File == nil {
		_ = filex.RemoveTempFile(result.TempPath, uploadConf.KeepTempFiles)
		return &file.File{}
	}

	var pbFile file.File
	copier.Copy(&pbFile, result.File) // nolint:errcheck
	pbFile.Md5 = result.File.Md5

	if !filex.IsImageContentType(result.ContentType) {
		_ = filex.RemoveTempFile(result.TempPath, uploadConf.KeepTempFiles)
		return &pbFile
	}

	if exifMeta, err := imagex.ExtractImageMetaFromBytes(result.Head); err == nil {
		var meta file.ImageMeta
		copier.Copy(&meta, &exifMeta) // nolint:errcheck
		pbFile.Meta = &meta
	}

	if !needThumbGeneration(uploadConf, result, isThumb, thumbTaskRunner) {
		_ = filex.RemoveTempFile(result.TempPath, uploadConf.KeepTempFiles)
		return &pbFile
	}

	scheduleThumbGeneration(ctx, uploadConf, &pbFile, result, ossTemplate,
		tenantID, bucketName, filename, thumbTaskRunner)

	return &pbFile
}

func needThumbGeneration(uploadConf config.UploadConf, result *ossx.StreamUploadResult, isThumb bool, thumbTaskRunner *threading.TaskRunner) bool {
	return isThumb &&
		thumbTaskRunner != nil &&
		result.TempPath != "" &&
		uploadConf.Image.Thumb.Enabled &&
		uploadConf.Image.Thumb.Width > 0 &&
		uploadConf.Image.Thumb.Height > 0
}

// scheduleThumbGeneration 在图片上传完成后异步生成缩略图：
//  1. 复制上传临时文件为缩略图源文件
//  2. 按配置清理原始上传临时文件
//  3. 预填响应中的 ThumbLink/ThumbName 字段
//  4. 通过 ThumbTaskRunner 调度异步任务，生成缩略图并上传到 OSS
func scheduleThumbGeneration(
	ctx context.Context,
	uploadConf config.UploadConf,
	pbFile *file.File,
	result *ossx.StreamUploadResult,
	ossTemplate ossx.OssTemplate,
	tenantID, bucketName, filename string,
	thumbTaskRunner *threading.TaskRunner,
) {
	sourcePath := result.TempPath
	thumbSourcePath := filepath.Join(filepath.Dir(sourcePath), filepath.Base(sourcePath)+"_source")
	if err := filex.CopyFile(sourcePath, thumbSourcePath); err != nil {
		logx.WithContext(ctx).Errorf("Failed to copy image temp file: %v", err)
		_ = filex.RemoveTempFile(result.TempPath, uploadConf.KeepTempFiles)
		return
	}
	_ = filex.RemoveTempFile(result.TempPath, uploadConf.KeepTempFiles)

	thumbFilename := "thumb_" + filename
	ossName := tool.GenOssFilename(thumbFilename, "thumb")
	pbFile.ThumbLink = pbFile.Domain + "/" + ossName
	pbFile.ThumbName = ossName

	thumbTaskRunner.Schedule(func() {
		defer os.Remove(thumbSourcePath) // nolint:errcheck
		thumbOutputPath := filepath.Join(filepath.Dir(thumbSourcePath), filepath.Base(thumbSourcePath)+"_thumb")
		bgCtx := context.WithoutCancel(ctx)
		generateAndUploadVariant(bgCtx, ossTemplate, tenantID, bucketName,
			thumbSourcePath, thumbOutputPath, thumbFilename, ossName,
			uploadConf.Image.Thumb.Width, uploadConf.Image.Thumb.Height, "thumb")
	})
}

func generateAndUploadVariant(
	ctx context.Context,
	ossTemplate ossx.OssTemplate,
	tenantID, bucketName,
	sourcePath, outputPath, filename, ossName string,
	width, height int,
	variant string,
) {
	start := timex.Now()
	defer os.Remove(outputPath) // nolint:errcheck

	if err := imagex.FromFileToFile(sourcePath, outputPath, width, height); err != nil {
		logx.WithContext(ctx).Errorf("Failed to generate %s image: %v", variant, err)
		return
	}

	f, err := os.Open(outputPath)
	if err != nil {
		logx.WithContext(ctx).Errorf("Failed to open %s image: %v", variant, err)
		return
	}
	defer f.Close()

	if _, err = ossTemplate.PutObject(ctx, tenantID, bucketName, filename, "image/jpeg", f, -1, "", ossName); err != nil {
		logx.WithContext(ctx).Errorf("Failed to upload %s image: %v", variant, err)
		return
	}
	logx.WithContext(ctx).WithDuration(timex.Since(start)).Infof("%s image finished processing", variant)
}

// streamUploadSession 等旧类型已废弃。
// 实际流式上传请使用 ossx.UploadStream + processUploadResult 的纯函数组合。
