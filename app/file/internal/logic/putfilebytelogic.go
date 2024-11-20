package logic

import (
	"context"
	"github.com/jinzhu/copier"
	"io"
	file2 "zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/model"
	"zero-service/ossx"

	"github.com/zeromicro/go-zero/core/logx"
)

type PutFileByteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPutFileByteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PutFileByteLogic {
	return &PutFileByteLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PutFileByteLogic) PutFileByte(stream file2.FileRpc_PutFileByteServer) error {
	// 使用管道实现流式数据写入
	pr, pw := io.Pipe()
	defer pw.Close()

	// 用于存储元信息
	var tenantID, code, bucketName, filename string
	var contentType string

	// 预定义错误通道
	errChan := make(chan error, 1)
	var pbFile file2.File

	// 标记是否已经初始化 OSS 模板
	var initialized bool
	// 从 gRPC 流中逐块读取数据并写入管道
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// 所有数据读取完毕，关闭管道写入
			pw.Close()
			break
		}
		if err != nil {
			l.Logger.Errorf("Failed to read chunk: %v", err)
			return err
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
				return ossErr
			}

			// 启动一个 goroutine，将管道数据写入 OSS
			go func() {
				defer func() {
					close(errChan)
				}()
				// 写入 OSS
				uploadedFile, ossChanErr := ossTemplate.PutObject(tenantID, bucketName, filename, contentType, pr, -1)
				_ = copier.Copy(&pbFile, uploadedFile)
				errChan <- ossChanErr
			}()

			initialized = true
		}

		// 写入文件数据到管道
		if _, err = pw.Write(req.GetContent()); err != nil {
			return err
		}
	}

	// 等待上传完成
	if err := <-errChan; err != nil {
		l.Logger.Errorf("Failed to upload file: %v", err)
		return err
	}
	// 返回上传结果
	return stream.SendAndClose(&file2.PutFileByteRes{
		File: &pbFile,
	})
}
