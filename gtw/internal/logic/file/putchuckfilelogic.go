package file

import (
	"context"
	"github.com/jinzhu/copier"
	"io"
	"net/http"
	"zero-service/common/tool"
	"zero-service/file/file"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PutChuckFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 上传大文件
func NewPutChuckFileLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *PutChuckFileLogic {
	return &PutChuckFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *PutChuckFileLogic) PutChuckFile(req *types.PutFileRequest) (resp *types.GetFileReply, err error) {
	l.r.ParseMultipartForm(maxFileSize)
	uploadFile, fileHeader, err := l.r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer uploadFile.Close()
	l.Logger.Infof("upload file: %+v, file size: %s, MIME header: %+v",
		fileHeader.Filename, tool.FormatFileSize(fileHeader.Size), fileHeader.Header)

	// 执行 stream 上传
	stream, err := l.svcCtx.FileRpcCLi.PutFileByte(context.Background())
	if err != nil {
		l.Logger.Errorf("Failed to create stream: %v", err)
		return nil, err
	}

	// 逐块读取文件并上传
	buf := make([]byte, partSize)
	partNum := 1
	var uploadedSize int64 = 0
	for {
		n, err := uploadFile.Read(buf)
		if n > 0 {
			// 更新已上传大小
			uploadedSize += int64(n)
			// 发送文件块到服务器
			chunk := &file.PutFileByteReq{
				TenantId:    req.TenantId,
				Code:        req.Code,
				BucketName:  req.BucketName,
				Content:     buf[:n],
				Filename:    fileHeader.Filename,
				ContentType: fileHeader.Header.Get("content-type"),
			}
			if err := stream.Send(chunk); err != nil {
				l.Logger.Errorf("Failed to send chunk: %v", err)
				return nil, err
			}

			// 打印当前分片上传进度
			progress := float64(uploadedSize) / float64(fileHeader.Size) * 100
			l.Logger.Infof(
				"Uploading part %d: %s (%.2f%% completed, Uploaded: %s / %s)",
				partNum, tool.FormatFileSize(int64(n)), progress, tool.FormatFileSize(uploadedSize), tool.FormatFileSize(fileHeader.Size))
			partNum++
		}

		if err == io.EOF {
			break // 文件读取完毕
		}
		if err != nil {
			l.Logger.Errorf("Failed to read file: %v", err)
			return nil, err
		}
	}

	// 完成上传并接收服务器响应
	res, err := stream.CloseAndRecv()
	if err != nil {
		l.Logger.Errorf("Failed to receive response: %v", err)
		return nil, err
	}
	var file types.File
	_ = copier.Copy(&file, res.File)
	return &types.GetFileReply{
		File: file,
	}, nil
}
