package logic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	media "zero-service/common/mediax"

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
	ossTemplate, err := l.svcCtx.GetOssTemplate(l.ctx, in.TenantId, in.Code)
	if err != nil {
		return nil, err
	}
	shot, err := media.NewScreenshotter(in.StreamUrl)
	if err != nil {
		return nil, err
	}
	cutFilePath, err := shot.CaptureFrameToFile(l.ctx, -1, shot.GenerateTempFilePath(l.svcCtx.Config.Upload.TempDir, ".jpg"))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.Remove(cutFilePath); err != nil {
			logx.WithContext(l.ctx).Errorf("Failed to remove temp file %s: %v", cutFilePath, err)
		}
	}()

	cutFile, err := os.Open(cutFilePath)
	if err != nil {
		return nil, fmt.Errorf("open capture file failed: %w", err)
	}
	defer cutFile.Close()

	fileInfo, err := cutFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat capture file failed: %w", err)
	}

	uploadedFile, err := ossTemplate.PutObject(
		l.ctx, in.TenantId, in.BucketName, filepath.Base(cutFilePath), "image/jpeg",
		cutFile, fileInfo.Size(), in.PathPrefix,
	)
	if err != nil {
		return nil, err
	}

	var pbFile file.File
	copier.Copy(&pbFile, uploadedFile) // nolint:errcheck
	l.Logger.Infof("File uploaded to OSS: %s success", uploadedFile.Name)

	return &file.CaptureVideoStreamRes{
		File: &pbFile,
	}, nil
}
