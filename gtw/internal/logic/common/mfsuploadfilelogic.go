package common

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"zero-service/common/imagex"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/random"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/timex"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxFileSize = 10 << 20 // 10 MB

type MfsUploadFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
	w      http.ResponseWriter
}

// 上传文件
func NewMfsUploadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request, w http.ResponseWriter) *MfsUploadFileLogic {
	return &MfsUploadFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
		w:      w,
	}
}

func (l *MfsUploadFileLogic) MfsUploadFile(req *types.UploadFileRequest) (resp *types.UploadFileReply, err error) {
	l.r.ParseMultipartForm(maxFileSize)
	file, fileHeader, err := l.r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	logx.Infof("upload file: %+v, file size: %d, MIME header: %+v",
		fileHeader.Filename, fileHeader.Size, fileHeader.Header)
	if err != nil {
		return nil, err
	}
	typeFile := "tempFile"
	if req.MfsType == 2 {
		typeFile = "bizFile"
	}
	dayStr := carbon.Now().Format("20060102")
	dirPath := l.svcCtx.Config.NfsRootPath + "/" + typeFile + "/" + dayStr
	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return nil, err
	}
	u, _ := random.UUIdV4()
	filePath := dirPath + "/" + strings.Replace(fmt.Sprintf("%s", u), "-", "", -1) + path.Ext(fileHeader.Filename)
	f, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buffer := bufio.NewReader(file)
	//b := make([]byte, 1024)
	//for {
	//	n, err := buffer.Read(b)
	//	if err != nil && err != io.EOF {
	//		//有一个特殊问题，当一个文件读读完，遇到文件末尾时，它也会返回一个错误，但是此时我已经读到文件末尾EOF，这个错误应该不算错误，所以应该把读到文件末尾这个错误给去掉。
	//		return nil, err
	//	}
	//	if err == io.EOF {
	//		break
	//	}
	//	_, err = f.Write(b[:n])
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	_, err = io.Copy(f, buffer)
	if err != nil {
		return nil, err
	}
	reply := &types.UploadFileReply{
		Name:        fileHeader.Filename,
		Path:        filePath,
		Size:        fileHeader.Size,
		ContextType: fileHeader.Header.Get("Content-Type"),
		Url:         l.svcCtx.Config.DownloadUrl + filePath,
	}
	meta := types.ImageMeta{}
	if strings.HasPrefix(fileHeader.Header.Get("Content-Type"), "image/") {
		exifMeta, err := imagex.ExtractImageMeta(filePath)
		if err == nil {
			_ = copier.Copy(&meta, &exifMeta)
			reply.Meta = &meta
			if req.IsThumb {
				thumbStart := timex.Now()
				thumbPath := dirPath + "/" + strings.Replace(fmt.Sprintf("%s", u), "-", "", -1) + "_thumb" + path.Ext(fileHeader.Filename)
				err = imagex.FromFileToFile(filePath, thumbPath, 200, 200)
				if err != nil {
					l.Logger.Errorf("thumb error: %v", err)
				}
				duration := timex.Since(thumbStart)
				l.Logger.WithDuration(duration).Infof("thumb finished processing")
				reply.ThumbPath = thumbPath
				reply.ThumbUrl = l.svcCtx.Config.DownloadUrl + thumbPath
			}
		}
	}
	return reply, nil
}
