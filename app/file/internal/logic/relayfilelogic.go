package logic

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/filex"
	"zero-service/common/ossx"
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
		return nil, fmt.Errorf("sourceUrl or sourcePath must be provided")
	}
	if len(in.Targets) == 0 {
		return nil, fmt.Errorf("at least one target must be specified")
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
	head, size, err := filex.ReadFileHead(sourcePath, uploadContentTypeProbeBytes)
	if err != nil {
		return nil, err
	}
	contentType := resolveContentType(l.ctx, reqContentType, head)
	return &relaySource{
		filename:    relaySourceFilename(sourcePath),
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
		return nil, fmt.Errorf("download from source URL failed: %w", err)
	}
	defer body.Close()

	if err := os.MkdirAll(l.svcCtx.Config.Upload.TempDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create relay temp dir failed: %w", err)
	}
	f, err := os.CreateTemp(l.svcCtx.Config.Upload.TempDir, relayDownloadPattern)
	if err != nil {
		return nil, fmt.Errorf("create relay temp file failed: %w", err)
	}
	tempPath := f.Name()

	headWriter := filex.NewHeadCaptureWriter(uploadContentTypeProbeBytes)
	written, copyErr := io.Copy(f, io.TeeReader(io.LimitReader(body, relayMaxSourceBytes+1), headWriter))
	if copyErr != nil {
		_ = f.Close()
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("download source URL body failed: %w", copyErr)
	}
	if written > relayMaxSourceBytes {
		_ = f.Close()
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("source URL body exceeds limit: %d", relayMaxSourceBytes)
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("close relay temp file failed: %w", err)
	}

	contentType := resolveContentType(l.ctx, reqContentType, headWriter.Bytes())

	return &relaySource{
		filename:    relaySourceFilename(sourceURL),
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
	results := make([]relayResult, len(targets))
	var firstErr error
	for i, target := range targets {
		ossTemplate, err := l.svcCtx.GetOssTemplate(l.ctx, target.TenantId, target.Code)
		if err != nil {
			results[i] = relayResult{Index: i, Err: err}
		} else {
			results[i] = l.uploadToRelayTarget(i, target, source, ossTemplate)
		}
		if firstErr == nil && results[i].Err != nil {
			firstErr = results[i].Err
		}
	}
	return results, firstErr
}

// uploadToRelayTarget 执行单目标上传，统一管理 reader 生命周期。
func (l *RelayFileLogic) uploadToRelayTarget(index int, target *file.RelayTarget, source *relaySource, ossTemplate ossx.OssTemplate) relayResult {
	reader, err := source.openReader()
	if err != nil {
		return relayResult{Index: index, Err: err}
	}
	defer reader.Close()

	filename := target.Filename
	if filename == "" {
		filename = source.filename
	}
	uploadedFile, err := ossTemplate.PutObject(l.ctx, target.TenantId, target.BucketName, filename, source.contentType, reader, source.size, target.PathPrefix)
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

func relaySourceFilename(source string) string {
	if source == "" {
		return ""
	}
	if u, err := url.Parse(source); err == nil && u.Path != "" {
		if name := path.Base(u.Path); name != "." && name != "/" {
			return name
		}
	}
	return filepath.Base(strings.TrimRight(source, string(os.PathSeparator)))
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
