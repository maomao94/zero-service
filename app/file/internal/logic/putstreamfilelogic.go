package logic

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"zero-service/common/imagex"
	"zero-service/common/ossx"
	"zero-service/common/tool"
	"zero-service/model"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/timex"
)

const maxExifRead = 64 * 1024 // 64KB

const progressLogThreshold = 100 * 1024 * 1024 // 100MB
type PutStreamFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPutStreamFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PutStreamFileLogic {
	return &PutStreamFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PutStreamFileLogic) PutStreamFile(stream file.FileRpc_PutStreamFileServer) error {
	// 使用管道实现流式数据写入
	pr, pw := io.Pipe()
	err := os.MkdirAll("/opt/data/temp", os.ModePerm)
	if err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp("/opt/data/temp", "upload-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	// 用于存储元信息
	var tenantID, code, bucketName, filename string
	var contentType string
	var size int64
	var ossTemplate ossx.OssTemplate

	var pbFile file.File

	// 标记是否已经初始化 OSS 模板
	var initialized bool
	// 存储用于探测内容类型的缓冲区
	var contentBuffer []byte
	// 用来存储EXIF信息
	var exifBuf []byte
	// 用来记录已上传的字节数
	var writeSize int64
	var isThumb bool

	// 进度日志相关变量
	var lastLoggedSize int64 // 上次记录日志时的已上传字节数
	var partNum int          // 块编号

	errOssChan := make(chan error, 1)
	var errRead error
	// 从 gRPC 流中逐块读取数据并写入管道
	for {
		if initialized {
			if writeSize >= size {
				break
			}
		}
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil && err != io.EOF {
			l.Logger.Errorf("Failed to read from stream: %v", err)
			errRead = err
			break
		}
		// 解析消息中的元数据（仅需要解析一次）
		if !initialized {
			tenantID = req.GetTenantId()
			code = req.GetCode()
			bucketName = req.GetBucketName()
			filename = req.GetFilename()
			contentType = req.GetContentType()
			size = req.GetSize()
			isThumb = req.GetIsThumb()

			// 动态获取 OSS 模板
			var ossErr error
			ossTemplate, ossErr = ossx.Template(
				tenantID, code,
				l.svcCtx.Config.Oss.TenantMode,
				func(tenantId, code string) (*model.Oss, error) {
					return l.svcCtx.OssModel.FindOneByTenantIdOssCode(l.ctx, tenantId, code)
				},
			)
			if ossErr != nil {
				l.Logger.Errorf("Failed to get OSS template: %v", ossErr)
				errRead = ossErr
				break
			}

			// 启动一个 goroutine，将管道数据写入 OSS
			go threading.RunSafe(func() {
				defer func() {
					pr.Close()
					close(errOssChan)
				}()
				// 写入 OSS
				uploadedFile, ossPutErr := ossTemplate.PutObject(l.ctx, tenantID, bucketName, filename, contentType, pr, size)
				_ = copier.Copy(&pbFile, uploadedFile)
				if ossPutErr != nil {
					l.Logger.Errorf("Failed to write to OSS: %v", ossPutErr)
				}
				if ossPutErr == nil {
					l.Logger.Infof("File uploaded to OSS: %s success", filename)
				}
				errOssChan <- ossPutErr

			})
			initialized = true
			l.Logger.Infof("Start receiving file: %s, total size: %s", filename, tool.DecimalBytes(size))
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

		if strings.HasPrefix(contentType, "image/") {
			// 缓存前 maxExifRead 字节用于 EXIF
			if len(exifBuf) < maxExifRead {
				need := maxExifRead - len(exifBuf)
				if len(contentBuffer) > need {
					exifBuf = append(exifBuf, contentBuffer[:need]...)
				} else {
					exifBuf = append(exifBuf, contentBuffer...)
				}
			}
			multi := io.MultiWriter(pw, tmpFile)
			_, err = multi.Write(req.GetContent())
			if err != nil {
				l.Logger.Errorf("Failed to write to multi writer: %v", err)
				errRead = err
				break
			}
		} else {
			// 写入文件数据到管道
			_, err = pw.Write(req.GetContent())
			if err != nil {
				l.Logger.Errorf("Failed to write to pipe: %v", err)
				errRead = err
				break
			}
		}

		// 更新已上传字节数和块编号
		chunkSize := int64(len(req.GetContent()))
		writeSize += chunkSize
		partNum++

		// 上传进度日志，达到阈值或完成时打印
		if writeSize-lastLoggedSize >= progressLogThreshold || writeSize == size {
			progress := float64(writeSize) / float64(size) * 100
			l.Logger.Infof(
				"Received part %d: %s (%.2f%% completed, Received: %s / %s)",
				partNum, tool.DecimalBytes(writeSize-lastLoggedSize), progress,
				tool.DecimalBytes(writeSize), tool.DecimalBytes(size))
			lastLoggedSize = writeSize
		}
	}

	// 关闭写管道
	pw.Close()

	if initialized {
		// 等待上传完成
		if err := <-errOssChan; err != nil {
			return err
		}
		if errRead != nil {
			return errRead
		}

		if strings.HasPrefix(contentType, "image/") {
			exifMeta, err := imagex.ExtractImageMetaFromBytes(exifBuf)
			if err == nil {
				var meta file.ImageMeta
				_ = copier.Copy(&meta, &exifMeta)
				pbFile.Meta = &meta
			}
			if isThumb {
				go threading.RunSafe(func() {
					thumbStart := timex.Now()
					thumbPath := tmpFile.Name() + "_thumb.jpg"
					// 生成缩略图
					if err = imagex.FromFileToFile(tmpFile.Name(), thumbPath, 300, 300); err == nil {
						// 上传缩略图
						f, _ := os.Open(thumbPath)
						defer f.Close()
						thumbFile, err := ossTemplate.PutObject(context.Background(), tenantID, bucketName, "thumb_"+filename, "image/jpeg", f, -1)
						if err != nil {
							l.Logger.Errorf("Failed to upload thumbnail: %v", err)
						}
						os.Remove(thumbPath)
						pbFile.ThumbLink = thumbFile.Link
						pbFile.ThumbName = thumbFile.Name
					} else {
						l.Logger.Errorf("Failed to generate thumbnail: %v", err)
					}
					duration := timex.Since(thumbStart)
					l.Logger.WithDuration(duration).Infof("thumb finished processing")
				})
			}
		}

		// 新增：上传完成日志
		l.Logger.Infof("File %s uploaded completely. Total received: %s",
			filename, tool.DecimalBytes(writeSize))

		// 返回上传结果
		return stream.SendAndClose(&file.PutStreamFileRes{
			File:  &pbFile,
			IsEnd: true,
			Size:  writeSize,
		})
	} else {
		if errRead == nil {
			errRead = io.EOF
		}
		return errRead
	}
}
