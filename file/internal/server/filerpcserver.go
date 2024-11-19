// Code generated by goctl. DO NOT EDIT.
// goctl 1.7.3
// Source: file.proto

package server

import (
	"context"

	"zero-service/file/file"
	"zero-service/file/internal/logic"
	"zero-service/file/internal/svc"
)

type FileRpcServer struct {
	svcCtx *svc.ServiceContext
	file.UnimplementedFileRpcServer
}

func NewFileRpcServer(svcCtx *svc.ServiceContext) *FileRpcServer {
	return &FileRpcServer{
		svcCtx: svcCtx,
	}
}

func (s *FileRpcServer) Ping(ctx context.Context, in *file.Req) (*file.Res, error) {
	l := logic.NewPingLogic(ctx, s.svcCtx)
	return l.Ping(in)
}

func (s *FileRpcServer) OssDetail(ctx context.Context, in *file.OssDetailReq) (*file.OssDetailRes, error) {
	l := logic.NewOssDetailLogic(ctx, s.svcCtx)
	return l.OssDetail(in)
}

func (s *FileRpcServer) OssList(ctx context.Context, in *file.OssListReq) (*file.OssListRes, error) {
	l := logic.NewOssListLogic(ctx, s.svcCtx)
	return l.OssList(in)
}

func (s *FileRpcServer) CreateOss(ctx context.Context, in *file.CreateOssReq) (*file.CreateOssRes, error) {
	l := logic.NewCreateOssLogic(ctx, s.svcCtx)
	return l.CreateOss(in)
}

func (s *FileRpcServer) UpdateOss(ctx context.Context, in *file.UpdateOssReq) (*file.UpdateOssRes, error) {
	l := logic.NewUpdateOssLogic(ctx, s.svcCtx)
	return l.UpdateOss(in)
}

func (s *FileRpcServer) DeleteOss(ctx context.Context, in *file.DeleteOssReq) (*file.DeleteOssRes, error) {
	l := logic.NewDeleteOssLogic(ctx, s.svcCtx)
	return l.DeleteOss(in)
}

func (s *FileRpcServer) MakeBucket(ctx context.Context, in *file.MakeBucketReq) (*file.MakeBucketRes, error) {
	l := logic.NewMakeBucketLogic(ctx, s.svcCtx)
	return l.MakeBucket(in)
}

func (s *FileRpcServer) RemoveBucket(ctx context.Context, in *file.RemoveBucketReq) (*file.RemoveBucketRes, error) {
	l := logic.NewRemoveBucketLogic(ctx, s.svcCtx)
	return l.RemoveBucket(in)
}

func (s *FileRpcServer) StatFile(ctx context.Context, in *file.StatFileReq) (*file.StatFileRes, error) {
	l := logic.NewStatFileLogic(ctx, s.svcCtx)
	return l.StatFile(in)
}

func (s *FileRpcServer) PutFile(ctx context.Context, in *file.PutFileReq) (*file.PutFileRes, error) {
	l := logic.NewPutFileLogic(ctx, s.svcCtx)
	return l.PutFile(in)
}

func (s *FileRpcServer) GetFile(ctx context.Context, in *file.GetFileReq) (*file.GetFileRes, error) {
	l := logic.NewGetFileLogic(ctx, s.svcCtx)
	return l.GetFile(in)
}

func (s *FileRpcServer) RemoveFile(ctx context.Context, in *file.RemoveFileReq) (*file.RemoveFileRes, error) {
	l := logic.NewRemoveFileLogic(ctx, s.svcCtx)
	return l.RemoveFile(in)
}

func (s *FileRpcServer) RemoveFiles(ctx context.Context, in *file.RemoveFilesReq) (*file.RemoveFileRes, error) {
	l := logic.NewRemoveFilesLogic(ctx, s.svcCtx)
	return l.RemoveFiles(in)
}
