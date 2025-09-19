package logic

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	media "zero-service/common/mediax"
	"zero-service/common/ossx"
	"zero-service/model"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type CaptureVideoStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCaptureVideoStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CaptureVideoStreamLogic {
	return &CaptureVideoStreamLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CaptureVideoStreamLogic) CaptureVideoStream(in *file.CaptureVideoStreamReq) (*file.CaptureVideoStreamRes, error) {
	ossTemplate, ossErr := ossx.Template(
		in.TenantId, in.Code,
		l.svcCtx.Config.Oss.TenantMode,
		func(tenantId, code string) (*model.Oss, error) {
			return l.svcCtx.OssModel.FindOneByTenantIdOssCode(l.ctx, tenantId, code)
		},
	)
	if ossErr != nil {
		return nil, ossErr
	}
	shot, err := media.NewScreenshotter(in.StreamUrl)
	if err != nil {
		return nil, err
	}
	cutFilePath, err := shot.CaptureFrameToFile(l.ctx, -1, shot.GenerateTempFilePath("/opt/data/capture", ".jpg"))
	if err != nil {
		return nil, err
	}
	defer func() {
		if removeErr := os.Remove(cutFilePath); removeErr != nil {
			logx.Errorf("Failed to remove temporary file: %v", removeErr)
		}
	}()
	cutfile, err := os.Open(cutFilePath)
	if err != nil {
		return nil, fmt.Errorf("打开截图文件失败: %w", err)
	}
	defer cutfile.Close() // 确保文件最终关闭
	fileInfo, err := cutfile.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}
	// 计算文件MD5（关键修改）
	hash := md5.New()
	if _, err := io.Copy(hash, cutfile); err != nil {
		return nil, fmt.Errorf("计算MD5失败: %w", err)
	}
	fileMD5 := hex.EncodeToString(hash.Sum(nil)) // 转换为16进制字符串
	// 重置文件指针到开头
	if _, err := cutfile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("重置文件指针失败: %w", err)
	}
	fileName := filepath.Base(cutFilePath)
	contentType := "image/jpeg"   // JPG图片的MIME类型
	objectSize := fileInfo.Size() // 文件大小
	var pbFile file.File
	uploadedFile, ossPutErr := ossTemplate.PutObject(context.Background(), in.TenantId, in.BucketName, fileName, contentType, cutfile, objectSize, in.PathPrefix)
	if ossPutErr != nil {
		return nil, ossPutErr
	}
	_ = copier.Copy(&pbFile, uploadedFile)
	pbFile.Md5 = fileMD5
	l.Logger.Infof("File uploaded to OSS: %s success", uploadedFile.Name)
	return &file.CaptureVideoStreamRes{
		File: &pbFile,
	}, nil
}
