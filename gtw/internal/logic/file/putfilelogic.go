package file

import (
	"bufio"
	"context"
	"fmt"
	"github.com/golang-module/carbon/v2"
	"github.com/google/uuid"
	"github.com/jinzhu/copier"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"zero-service/file/file"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxFileSize = 10 << 20 // 10 MB

const partSize = 5 * 1024 * 1024 // 每个分片的大小 5MB

type PutFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 上传文件
func NewPutFileLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *PutFileLogic {
	return &PutFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *PutFileLogic) PutFile(req *types.PutFileRequest) (resp *types.GetFileReply, err error) {
	l.r.ParseMultipartForm(maxFileSize)
	uploadFile, fileHeader, err := l.r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer uploadFile.Close()
	l.Logger.Infof("upload file: %+v, file size: %d, MIME header: %+v",
		fileHeader.Filename, fileHeader.Size, fileHeader.Header)

	// 执行 stream 上传
	if true {
		stream, err := l.svcCtx.FileRpcCLi.PutFileByte(context.Background())
		if err != nil {
			l.Logger.Errorf("Failed to create stream: %v", err)
			return nil, err
		}

		// 逐块读取文件并上传
		buf := make([]byte, partSize)
		partNum := 1
		for {
			n, err := uploadFile.Read(buf)
			if n > 0 {
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
				fmt.Printf("Uploading part %d: %d bytes\n", partNum, n)
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
		resp, err := stream.CloseAndRecv()
		if err != nil {
			l.Logger.Errorf("Failed to receive response: %v", err)
			return nil, err
		}
		var file types.File
		_ = copier.Copy(&file, resp.File)
		return &types.GetFileReply{
			File: file,
		}, nil
	} else {
		// 执行普通上传
		typeFile := "tempFile"
		dayStr := carbon.Now().Format("20060102")
		dirPath := l.svcCtx.Config.NfsRootPath + "/" + typeFile + "/" + dayStr
		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			return nil, err
		}
		u, _ := uuid.NewUUID()
		path := dirPath + "/" + strings.Replace(fmt.Sprintf("%s", u), "-", "", -1) + path.Ext(fileHeader.Filename)
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		buffer := bufio.NewReader(uploadFile)
		_, err = io.Copy(f, buffer)
		if err != nil {
			return nil, err
		}
		putFileResp, err := l.svcCtx.FileRpcCLi.PutFile(l.ctx, &file.PutFileReq{
			TenantId:    req.TenantId,
			Code:        req.Code,
			BucketName:  req.BucketName,
			Path:        path,
			Filename:    fileHeader.Filename,
			ContentType: fileHeader.Header.Get("content-type"),
		})
		if err != nil {
			return nil, err
		}
		var file types.File
		_ = copier.Copy(&file, putFileResp.File)
		return &types.GetFileReply{
			File: file,
		}, nil
	}
}
