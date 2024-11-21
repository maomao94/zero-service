package logic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/threading"
	"io"
	"net/http"
	"zero-service/model"
	"zero-service/ossx"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PutChunkFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPutChunkFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PutChunkFileLogic {
	return &PutChunkFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PutChunkFileLogic) PutChunkFile(stream file.FileRpc_PutChunkFileServer) error {
	// 使用管道实现流式数据写入
	pr, pw := io.Pipe()
	defer pw.Close()

	// 用于存储元信息
	var tenantID, code, bucketName, filename string
	var contentType string

	var pbFile file.File

	// 标记是否已经初始化 OSS 模板
	var initialized bool
	// 存储用于探测内容类型的缓冲区
	var contentBuffer []byte

	errChan := make(chan error, 1)
	defer close(errChan)
	errReadChan := make(chan error, 1)
	defer close(errReadChan)
	go threading.RunSafe(func() {
		// 从 gRPC 流中逐块读取数据并写入管道
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				pw.Close()
				break
			}
			if err != nil && err != io.EOF {
				l.Logger.Errorf("Failed to read from stream: %v", err)
				errReadChan <- err
				break
			}
			// 解析消息中的元数据（仅需要解析一次）
			if !initialized {
				tenantID = req.GetTenantId()
				code = req.GetCode()
				bucketName = req.GetBucketName()
				filename = req.GetFilename()
				contentType = req.GetContentType()

				// 动态获取 OSS 模板
				var ossErr error
				ossTemplate, ossErr := ossx.Template(
					tenantID, code,
					l.svcCtx.Config.Oss.TenantMode,
					func(tenantId, code string) (*model.Oss, error) {
						return l.svcCtx.OssModel.FindOneByTenantIdOssCode(l.ctx, tenantId, code)
					},
				)
				if ossErr != nil {
					l.Logger.Errorf("Failed to get OSS template: %v", ossErr)
					errReadChan <- ossErr
					break
				}

				// 启动一个 goroutine，将管道数据写入 OSS
				go threading.RunSafe(func() {
					// 写入 OSS
					uploadedFile, ossChanErr := ossTemplate.PutObject(tenantID, bucketName, filename, contentType, pr, -1)
					_ = copier.Copy(&pbFile, uploadedFile)
					errChan <- ossChanErr
				})
				initialized = true
			}

			// 试图探测文件内容类型（在收到的第一部分数据上进行）
			if len(contentBuffer) < 512 {
				contentBuffer = append(contentBuffer, req.GetContent()...)

				// 如果已经读取到足够的数据，探测内容类型
				if len(contentBuffer) >= 512 {
					contentType = http.DetectContentType(contentBuffer[:512])
					l.Logger.Infof("Detected Content-Type: %s", contentType)
				}
			}

			// 写入文件数据到管道
			_, err = pw.Write(req.GetContent())
			if err != nil {
				l.Logger.Errorf("Failed to write to pipe: %v", err)
				errChan <- err
				break
			}
		}
	})
	select {
	case err := <-errReadChan:
		return err
	case ossErr := <-errChan:
		if ossErr != nil {
			l.Logger.Errorf("Failed to upload file to OSS: %v", ossErr)
			return ossErr
		}
	}
	// 返回上传结果
	return stream.SendAndClose(&file.PutChunkFileRes{
		File: &pbFile,
	})
}
