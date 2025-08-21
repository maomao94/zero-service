package api

import (
	"context"

	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"

	"github.com/Masterminds/squirrel"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListTsFilesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询 TS 文件列表（按时间区间）
func NewListTsFilesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTsFilesLogic {
	return &ListTsFilesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTsFilesLogic) ListTsFiles(req *types.ApiListTsRequest) (resp *types.ApiListTsReply, err error) {
	selectBuilder := l.svcCtx.HlsTsFilesModel.SelectBuilder()
	selectBuilder = selectBuilder.Where(squirrel.Eq{"stream_name": req.StreamName})
	if req.StartTime > 0 {
		selectBuilder = selectBuilder.Where(squirrel.GtOrEq{"ts_timestamp": req.StartTime})
	}
	if req.EndTime > 0 {
		selectBuilder = selectBuilder.Where(squirrel.LtOrEq{"ts_timestamp": req.EndTime})
	}
	if req.Event != "" {
		selectBuilder = selectBuilder.Where(squirrel.Eq{"event": req.Event})
	}
	list, err := l.svcCtx.HlsTsFilesModel.FindAll(l.ctx, selectBuilder, "ts_timestamp desc")
	if err != nil {
		return nil, err
	}
	var serverID string
	var files []types.ApiTsFile
	if len(list) > 0 {
		serverID = list[0].ServerId
		for _, v := range list {
			file := types.ApiTsFile{
				Event:       v.Event,
				TsFile:      v.TsFile,
				TsId:        v.TsId,
				Duration:    v.Duration.Float64,
				TsTimestamp: v.TsTimestamp,
			}
			files = append(files, file)
		}
	}

	return &types.ApiListTsReply{
		Files:      files,
		ServerID:   serverID,
		StreamName: req.StreamName,
	}, nil
}
