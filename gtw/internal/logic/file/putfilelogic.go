package file

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"zero-service/app/file/file"
	"zero-service/common/tool"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/dromara/carbon/v2"
	"github.com/google/uuid"
	"github.com/jinzhu/copier"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxFileSize = 10 << 20 // 10 MB

const partSize = 3 * 1024 * 1024 // 每个分片的大小 3MB

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
	putFileRes, err := l.svcCtx.FileRpcCLi.PutFile(l.ctx, &file.PutFileReq{
		TenantId:    req.TenantId,
		Code:        req.Code,
		BucketName:  req.BucketName,
		Path:        path,
		Filename:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("content-type"),
		IsThumb:     req.IsThumb,
	})
	if err != nil {
		return nil, err
	}
	var file types.File
	_ = copier.Copy(&file, putFileRes.File)
	return &types.GetFileReply{
		File: file,
	}, nil
}
