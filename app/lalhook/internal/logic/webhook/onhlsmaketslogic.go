package webhook

import (
	"context"
	"database/sql"
	"zero-service/model"

	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"

	"github.com/Masterminds/squirrel"
	"github.com/zeromicro/go-zero/core/logx"
)

type OnHlsMakeTsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// HLS 生成每个 ts 分片文件时
func NewOnHlsMakeTsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OnHlsMakeTsLogic {
	return &OnHlsMakeTsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OnHlsMakeTsLogic) OnHlsMakeTs(req *types.OnHlsMakeTsRequest) (resp *types.EmptyReply, err error) {
	hlsTsFiles := model.HlsTsFiles{
		Event:        req.Event,
		StreamName:   req.StreamName,
		Cwd:          req.Cwd,
		TsFile:       req.TsFile,
		LiveM3u8File: req.LiveM3u8File,
		RecordM3u8File: sql.NullString{
			String: req.RecordM3u8File,
			Valid:  req.RecordM3u8File != "",
		},
		TsId: req.ID,
		Duration: sql.NullFloat64{
			Float64: req.Duration,
			Valid:   req.Duration > 0,
		},
		ServerId: req.ServerID,
	}
	list, err := l.svcCtx.HlsTsFilesModel.FindAll(l.ctx, l.svcCtx.HlsTsFilesModel.SelectBuilder().Where(squirrel.Eq{"ts_file": hlsTsFiles.TsFile}), "")
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		_, err = l.svcCtx.HlsTsFilesModel.Insert(l.ctx, nil, &hlsTsFiles)
		if err != nil {
			return nil, err
		}
	} else {
		hlsTsFiles.Id = list[0].Id
		_, err = l.svcCtx.HlsTsFilesModel.Update(l.ctx, nil, &hlsTsFiles)
		if err != nil {
			return nil, err
		}
	}
	return
}
