package logic

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/antsx"
	"zero-service/common/filex"
	"zero-service/common/ossx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"
)

const (
	relayMaxSourceBytes  = 200 * 1024 * 1024
	relayDownloadPattern = "relay-source-*"
)

type RelayFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRelayFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RelayFileLogic {
	return &RelayFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RelayFileLogic) RelayFile(in *file.RelayFileReq) (*file.RelayFileRes, error) {
	if in.SourceUrl == "" && in.SourcePath == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "sourceUrl or sourcePath must be provided")
	}
	if len(in.Targets) == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "at least one target must be specified")
	}
	source, err := l.prepareRelaySource(in)
	if err != nil {
		return nil, err
	}
	defer source.cleanup()

	results, uploadErr := l.putRelayTargets(in.Targets, source)
	res := l.buildRelayResponse(results)
	if uploadErr != nil {
		return res, uploadErr
	}
	return res, nil
}

// prepareRelaySource 准备转推源，统一为可重复打开的 Reader。
func (l *RelayFileLogic) prepareRelaySource(in *file.RelayFileReq) (*relaySource, error) {
	if in.SourceUrl != "" {
		return l.fetchURLSource(in.SourceUrl, in.ContentType)
	}
	return l.openLocalSource(in.SourcePath, in.ContentType)
}

func (l *RelayFileLogic) openLocalSource(sourcePath, reqContentType string) (*relaySource, error) {
	head, size, err := filex.ReadFileHead(sourcePath, ossx.MaxContentTypeDetectBytes)
	if err != nil {
		return nil, err
	}
	contentType := resolveContentType(l.ctx, reqContentType, head)
	return &relaySource{
		filename:    filex.ExtractFilenameFromURL(sourcePath),
		size:        size,
		contentType: contentType,
		openReader: func() (io.ReadCloser, error) {
			return os.Open(sourcePath)
		},
		cleanup: func() {},
	}, nil
}

// fetchURLSource 将远端文件落到本地临时文件，避免转推时整文件驻留内存。
// 下载阶段不另设更短超时，与 file RPC 共用 l.ctx（deadline 由根配置 Timeout 与客户端决定），
// 避免大文件或慢源站时被固定 30s 误杀；整条 RelayFile 仍受该 deadline 约束（下载 + 多目标上传）。
func (l *RelayFileLogic) fetchURLSource(sourceURL, reqContentType string) (*relaySource, error) {
	body, err := l.svcCtx.NetClient.Download(l.ctx, sourceURL)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "download from source URL failed")
	}
	defer body.Close()

	if err := os.MkdirAll(l.svcCtx.Config.Upload.TempDir, os.ModePerm); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "create relay temp dir failed")
	}
	f, err := os.CreateTemp(l.svcCtx.Config.Upload.TempDir, relayDownloadPattern)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "create relay temp file failed")
	}
	tempPath := f.Name()

	headWriter := filex.NewHeadCaptureWriter(ossx.MaxContentTypeDetectBytes)
	written, copyErr := io.Copy(f, io.TeeReader(io.LimitReader(body, relayMaxSourceBytes+1), headWriter))
	if copyErr != nil {
		_ = f.Close()
		_ = os.Remove(tempPath)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, copyErr, "download source URL body failed")
	}
	if written > relayMaxSourceBytes {
		_ = f.Close()
		_ = os.Remove(tempPath)
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, fmt.Sprintf("source URL body exceeds limit: %d", relayMaxSourceBytes))
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(tempPath)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "close relay temp file failed")
	}

	contentType := resolveContentType(l.ctx, reqContentType, headWriter.Bytes())

	return &relaySource{
		filename:    filex.ExtractFilenameFromURL(sourceURL),
		size:        written,
		contentType: contentType,
		openReader: func() (io.ReadCloser, error) {
			return os.Open(tempPath)
		},
		cleanup: l.makePathCleanup(tempPath),
	}, nil
}

func (l *RelayFileLogic) makePathCleanup(tempPath string) func() {
	return func() {
		if l.svcCtx.Config.Upload.KeepTempFiles {
			l.Logger.Infof("Relay source temp file kept: %s", tempPath)
			return
		}
		_ = os.Remove(tempPath)
	}
}

// putRelayTargets 逐个目标执行转推，单目标失败不影响后续目标。
func (l *RelayFileLogic) putRelayTargets(targets []*file.RelayTarget, source *relaySource) ([]relayResult, error) {
	concurrency := l.svcCtx.Config.Upload.RelayUploadConcurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	reactor, err := antsx.NewReactor(concurrency)
	if err != nil {
		return nil, err
	}
	defer reactor.Release()

	tasks := make([]antsx.Task[relayResult], 0, len(targets))
	for i, target := range targets {
		index, relayTarget := i, target
		tasks = append(tasks, antsx.Task[relayResult]{
			Name: fmt.Sprintf("relay-target-%d", index),
			Fn: func(ctx context.Context) (relayResult, error) {
				ossTemplate, err := l.svcCtx.GetOssTemplate(ctx, relayTarget.TenantId, relayTarget.Code)
				if err != nil {
					return relayResult{Index: index, Err: err}, nil
				}
				return l.uploadToRelayTarget(ctx, index, relayTarget, source, ossTemplate), nil
			},
		})
	}

	settledResults := antsx.InvokeAllSettledWithReactor(l.ctx, reactor, tasks...)
	results := make([]relayResult, len(settledResults))
	var firstErr error
	for i, settledResult := range settledResults {
		if settledResult.Err != nil {
			results[i] = relayResult{Index: i, Err: settledResult.Err}
		} else {
			results[i] = settledResult.Val
		}
		if firstErr == nil && results[i].Err != nil {
			firstErr = results[i].Err
		}
	}
	return results, firstErr
}

// uploadToRelayTarget 执行单目标上传，统一管理 reader 生命周期。
func (l *RelayFileLogic) uploadToRelayTarget(ctx context.Context, index int, target *file.RelayTarget, source *relaySource, ossTemplate ossx.OssTemplate) relayResult {
	reader, err := source.openReader()
	if err != nil {
		return relayResult{Index: index, Err: err}
	}
	defer reader.Close()

	filename := target.Filename
	if filename == "" {
		filename = source.filename
	}
	uploadedFile, err := ossTemplate.PutObject(ctx, target.TenantId, target.BucketName, filename, source.contentType, reader, source.size, target.PathPrefix)
	if err != nil {
		return relayResult{Index: index, Err: err}
	}
	return relayResult{Index: index, File: uploadedFile}
}

func (l *RelayFileLogic) buildRelayResponse(results []relayResult) *file.RelayFileRes {
	res := &file.RelayFileRes{}
	for _, result := range results {
		if result.Err != nil {
			l.Logger.Errorf("Relay target %d failed: %v", result.Index, result.Err)
			continue
		}
		if result.File == nil {
			// 防御性检查：正常流程下 putRelayTargets 不会返回(Err==nil, File==nil) 的组合
			continue
		}
		var pbFile file.File
		copier.Copy(&pbFile, result.File) // nolint:errcheck
		l.Logger.Infof("Relay to target %d success: %s", result.Index, pbFile.Link)
		res.Files = append(res.Files, &pbFile)
	}
	return res
}

type relaySource struct {
	filename    string
	size        int64
	contentType string
	openReader  func() (io.ReadCloser, error)
	cleanup     func()
}

type relayResult struct {
	Index int
	File  *ossx.File
	Err   error
}
