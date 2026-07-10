package logic

import (
	"context"
	"fmt"

	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"
	"zero-service/common/tool"

	"github.com/jlaffaye/ftp"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListFTPSDirectoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListFTPSDirectoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFTPSDirectoryLogic {
	return &ListFTPSDirectoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

var entryTypeMapping = map[ftp.EntryType]ispagent.EntryType{
	ftp.EntryTypeFile:   ispagent.EntryType_ENTRY_TYPE_FILE,
	ftp.EntryTypeFolder: ispagent.EntryType_ENTRY_TYPE_FOLDER,
	ftp.EntryTypeLink:   ispagent.EntryType_ENTRY_TYPE_LINK,
}

func (l *ListFTPSDirectoryLogic) ListFTPSDirectory(in *ispagent.ListFTPSDirectoryReq) (*ispagent.ListFTPSDirectoryRes, error) {
	subPath := in.GetPath()
	ftpsEntries, err := l.svcCtx.ModelUploader.ListDir(l.ctx, subPath)
	if err != nil {
		l.Errorf("ftps list %q failed: %v", subPath, err)
		return &ispagent.ListFTPSDirectoryRes{
			Success: false,
			Error:   fmt.Sprintf("list %q failed: %v", subPath, err),
		}, nil
	}

	entries := make([]*ispagent.FTPSDirEntry, 0, len(ftpsEntries))
	for _, e := range ftpsEntries {
		entries = append(entries, &ispagent.FTPSDirEntry{
			Name:        e.Name,
			IsDir:       e.IsDir,
			Size:        e.Size,
			EntryType:   entryTypeMapping[e.Type],
			SizeDisplay: tool.BinaryBytes(int64(e.Size)),
		})
	}

	return &ispagent.ListFTPSDirectoryRes{
		Success: true,
		Entries: entries,
	}, nil
}
