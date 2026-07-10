package logic

import (
	"context"
	"fmt"
	"path/filepath"

	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"

	"github.com/zeromicro/go-zero/core/logx"
)

type TestFTPSUploadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTestFTPSUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TestFTPSUploadLogic {
	return &TestFTPSUploadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TestFTPSUploadLogic) TestFTPSUpload(in *ispagent.TestFTPSUploadReq) (*ispagent.TestFTPSUploadRes, error) {
	localPath := in.GetLocalPath()
	if localPath == "" {
		localPath = filepath.Join("local", "test.txt")
	}

	remoteName := filepath.Base(localPath)
	if _, err := l.svcCtx.ModelUploader.UploadFile(l.ctx, localPath, remoteName); err != nil {
		l.Errorf("ftps upload test file failed: %v", err)
		return &ispagent.TestFTPSUploadRes{
			Success: false,
			Error:   fmt.Sprintf("upload failed: %v", err),
		}, nil
	}

	cfg := l.svcCtx.ModelUploader.Config()
	remotePath := filepath.Join(cfg.RemoteDir, remoteName)

	return &ispagent.TestFTPSUploadRes{
		Success:    true,
		RemotePath: remotePath,
	}, nil
}
