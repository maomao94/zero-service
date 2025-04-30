package file

import (
	"context"
	"zero-service/app/file/file"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SignUrlLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 生成文件url
func NewSignUrlLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SignUrlLogic {
	return &SignUrlLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SignUrlLogic) SignUrl(req *types.SignUrlRequest) (resp *types.SignUrlReqly, err error) {
	signUrlRes, err := l.svcCtx.FileRpcCLi.SignUrl(l.ctx, &file.SignUrlReq{
		TenantId:   req.TenantId,
		Code:       req.Code,
		BucketName: req.BucketName,
		Filename:   req.Filename,
		Expires:    req.Expires,
	})
	if err != nil {
		return
	}
	return &types.SignUrlReqly{
		Url: signUrlRes.Url,
	}, nil
}
