package file

import (
	"context"
	"errors"
	"io"
	"net/http"
	"zero-service/app/file/file"
	"zero-service/common/tool"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type PutStreamFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 上传快文件-grpc单向流
func NewPutStreamFileLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *PutStreamFileLogic {
	return &PutStreamFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *PutStreamFileLogic) PutStreamFile(req *types.PutFileRequest) (resp *types.GetFileReply, err error) {
	l.r.ParseMultipartForm(maxFileSize)
	defer func() {
		if l.r.MultipartForm != nil {
			_ = l.r.MultipartForm.RemoveAll() // 清理临时文件
		}
	}()
	uploadFile, fileHeader, err := l.r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer uploadFile.Close()
	l.Logger.Infof("upload file: %+v, file size: %s, MIME header: %+v",
		fileHeader.Filename, tool.DecimalBytes(fileHeader.Size), fileHeader.Header)
	// 执行 stream 上传
	stream, err := l.svcCtx.FileRpcCLi.PutStreamFile(context.Background())
	if err != nil {
		l.Logger.Errorf("Failed to create stream: %v", err)
		return nil, err
	}
	defer stream.CloseSend()

	// 逐块读取文件并上传
	buf := make([]byte, partSize)
	partNum := 1
	// 用来记录已上传的字节数
	var uploadedSize int64
	var lastLoggedSize int64 // 记录上次打印日志时的字节数
	for {
		n, err := uploadFile.Read(buf)
		if n > 0 {
			// 更新已上传大小
			uploadedSize += int64(n)
			// 发送文件块到服务器
			chunk := &file.PutStreamFileReq{
				TenantId:    req.TenantId,
				Code:        req.Code,
				BucketName:  req.BucketName,
				Content:     buf[:n],
				Filename:    fileHeader.Filename,
				ContentType: fileHeader.Header.Get("content-type"),
				Size:        fileHeader.Size,
				IsThumb:     req.IsThumb,
			}
			if err := stream.Send(chunk); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				l.Logger.Errorf("Failed to write: %v", err)
				return nil, err
			}
			// 每当上传字节数超过阈值时打印一次进度
			if uploadedSize-lastLoggedSize >= progressLogThreshold || uploadedSize == fileHeader.Size {
				progress := float64(uploadedSize) / float64(fileHeader.Size) * 100
				l.Logger.Infof(
					"Uploading part %d: %s (%.2f%% completed, Uploaded: %s / %s)",
					partNum, tool.DecimalBytes(int64(uploadedSize-lastLoggedSize)), progress, tool.DecimalBytes(uploadedSize), tool.DecimalBytes(fileHeader.Size))
				lastLoggedSize = uploadedSize // 更新上次打印的已上传字节数
			}
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
	if res.IsEnd {
		var file types.File
		_ = copier.Copy(&file, res.File)
		return &types.GetFileReply{
			File: file,
		}, nil
	} else {
		return nil, errors.New("文件上传错误")
	}
}
