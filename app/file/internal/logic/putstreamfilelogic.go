package logic

import (
	"bytes"
	"context"
	"io"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/filex"
	"zero-service/common/ossx"
)

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

type grpcUploadReader struct {
	stream file.FileRpc_PutStreamFileServer
	buf    bytes.Buffer
}

// newGrpcUploadReader 将 gRPC 分片流包装成标准 io.Reader，便于复用统一上传流程。
func newGrpcUploadReader(stream file.FileRpc_PutStreamFileServer, firstContent []byte) io.Reader {
	r := &grpcUploadReader{stream: stream}
	r.buf.Write(firstContent)
	return r
}

// Read 在缓冲耗尽时继续从 gRPC 流读取分片，向下游提供连续字节流。
func (r *grpcUploadReader) Read(p []byte) (int, error) {
	for r.buf.Len() == 0 {
		req, err := r.stream.Recv()
		if err != nil {
			return 0, err
		}
		r.buf.Write(req.GetContent())
	}
	return r.buf.Read(p)
}

func (l *PutStreamFileLogic) PutStreamFile(stream file.FileRpc_PutStreamFileServer) error {
	if err := stream.Context().Err(); err != nil {
		return err
	}

	firstReq, err := stream.Recv()
	if err != nil {
		if err != io.EOF {
			l.Logger.Errorf("Failed to read from stream: %v", err)
		}
		return err
	}

	isThumb := firstReq.GetIsThumb()
	tenantID := firstReq.GetTenantId()
	if tenantID == "" {
		tenantID = "000000"
	}

	ossTemplate, err := l.svcCtx.GetOssTemplate(l.ctx, tenantID, firstReq.GetCode())
	if err != nil {
		return err
	}

	contentType := ossx.DetectContentType(firstReq.GetContentType(), firstReq.GetContent())

	result, err := ossx.UploadStream(l.ctx, ossx.StreamUploadRequest{
		Template:       ossTemplate,
		TenantID:       tenantID,
		BucketName:     firstReq.GetBucketName(),
		Filename:       firstReq.GetFilename(),
		ContentType:    contentType,
		Reader:         newGrpcUploadReader(stream, firstReq.GetContent()),
		Size:           firstReq.GetSize(),
		PathPrefix:     firstReq.GetPathPrefix(),
		CaptureOptions: buildCaptureOptions(l.svcCtx.Config.Upload, isThumb && filex.IsImageContentType(contentType)),
	})
	if err != nil {
		return err
	}

	pbFile := processUploadResult(l.ctx, l.svcCtx.Config.Upload, result, ossTemplate,
		tenantID, firstReq.GetBucketName(), firstReq.GetFilename(), isThumb,
		l.svcCtx.ThumbTaskRunner)

	return stream.SendAndClose(&file.PutStreamFileRes{
		File:  pbFile,
		IsEnd: true,
		Size:  result.Size,
	})
}
