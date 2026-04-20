package fsrestrict

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/adk/filesystem"
)

type policyBackend struct {
	inner filesystem.Backend
	cfg   Config
}

const (
	zoneNone = iota
	zoneUser
	zoneSession
)

func (b *policyBackend) sessionRoot(ctx context.Context) (string, bool) {
	if b.cfg.SessionBaseDir == "" {
		return "", false
	}
	sid := SessionIDFrom(ctx)
	if sid == "" {
		return "", false
	}
	if strings.ContainsAny(sid, `/\:`) {
		return "", false
	}
	return filepath.Join(b.cfg.SessionBaseDir, sid), true
}

func (b *policyBackend) offloadBase(ctx context.Context) string {
	if sr, ok := b.sessionRoot(ctx); ok {
		return sr
	}
	if len(b.cfg.UserRoots) > 0 {
		return b.cfg.UserRoots[0]
	}
	return ""
}

// resolveWithCtx 解析路径并判定区域（依赖 ctx 中的 session）。
func (b *policyBackend) resolveWithCtx(ctx context.Context, raw string) (abs string, zone int, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "."
	}
	ps := filepath.ToSlash(filepath.Clean(raw))
	if strings.HasPrefix(ps, "/large_tool_result") {
		base := b.offloadBase(ctx)
		if base == "" {
			return "", zoneNone, fmt.Errorf("fsrestrict: /large_tool_result requires user roots or session workspace")
		}
		suffix := strings.TrimPrefix(strings.TrimPrefix(ps, "/large_tool_result"), "/")
		abs = filepath.Clean(filepath.Join(base, "large_tool_result", suffix))
		return b.classifyWithCtx(ctx, abs)
	}

	var absPath string
	if filepath.IsAbs(raw) {
		absPath = filepath.Clean(raw)
	} else {
		if sr, ok := b.sessionRoot(ctx); ok {
			absPath = filepath.Join(sr, raw)
		} else if len(b.cfg.UserRoots) > 0 {
			absPath = filepath.Join(b.cfg.UserRoots[0], raw)
		} else {
			absPath, err = filepath.Abs(filepath.Clean(raw))
			if err != nil {
				return "", zoneNone, err
			}
			return b.classifyWithCtx(ctx, absPath)
		}
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", zoneNone, err
	}
	return b.classifyWithCtx(ctx, absPath)
}

func (b *policyBackend) classifyWithCtx(ctx context.Context, abs string) (string, int, error) {
	abs = filepath.Clean(abs)
	if sr, ok := b.sessionRoot(ctx); ok {
		if underRoot(sr, abs) {
			return abs, zoneSession, nil
		}
	}
	for _, u := range b.cfg.UserRoots {
		if underRoot(u, abs) {
			return abs, zoneUser, nil
		}
	}
	return "", zoneNone, fmt.Errorf("fsrestrict: path outside allowed roots: %s", abs)
}

func (b *policyBackend) allowRead(zone int) bool {
	switch zone {
	case zoneSession:
		return b.cfg.Policy.ReadSession
	case zoneUser:
		return b.cfg.Policy.ReadUser
	default:
		return false
	}
}

func (b *policyBackend) allowWrite(zone int) bool {
	switch zone {
	case zoneSession:
		return b.cfg.Policy.WriteSession
	case zoneUser:
		return b.cfg.Policy.WriteUser
	default:
		return false
	}
}

func (b *policyBackend) allowEdit(zone int) bool {
	switch zone {
	case zoneSession:
		return b.cfg.Policy.EditSession
	case zoneUser:
		return b.cfg.Policy.EditUser
	default:
		return false
	}
}

func (b *policyBackend) LsInfo(ctx context.Context, req *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	p := req.Path
	if p == "" {
		p = "."
	}
	np, zone, err := b.resolveWithCtx(ctx, p)
	if err != nil {
		return nil, err
	}
	if !b.allowRead(zone) {
		return nil, fmt.Errorf("fsrestrict: ls denied by policy for this path")
	}
	cp := *req
	cp.Path = np
	return b.inner.LsInfo(ctx, &cp)
}

func (b *policyBackend) Read(ctx context.Context, req *filesystem.ReadRequest) (*filesystem.FileContent, error) {
	np, zone, err := b.resolveWithCtx(ctx, req.FilePath)
	if err != nil {
		return nil, err
	}
	if !b.allowRead(zone) {
		return nil, fmt.Errorf("fsrestrict: read denied by policy")
	}
	cp := *req
	cp.FilePath = np
	return b.inner.Read(ctx, &cp)
}

func (b *policyBackend) GrepRaw(ctx context.Context, req *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	p := req.Path
	if p == "" {
		p = "."
	}
	np, zone, err := b.resolveWithCtx(ctx, p)
	if err != nil {
		return nil, err
	}
	if !b.allowRead(zone) {
		return nil, fmt.Errorf("fsrestrict: grep denied by policy")
	}
	cp := *req
	cp.Path = np
	return b.inner.GrepRaw(ctx, &cp)
}

func (b *policyBackend) GlobInfo(ctx context.Context, req *filesystem.GlobInfoRequest) ([]filesystem.FileInfo, error) {
	p := req.Path
	if p == "" {
		p = "."
	}
	np, zone, err := b.resolveWithCtx(ctx, p)
	if err != nil {
		return nil, err
	}
	if !b.allowRead(zone) {
		return nil, fmt.Errorf("fsrestrict: glob denied by policy")
	}
	cp := *req
	cp.Path = np
	return b.inner.GlobInfo(ctx, &cp)
}

func (b *policyBackend) Write(ctx context.Context, req *filesystem.WriteRequest) error {
	np, zone, err := b.resolveWithCtx(ctx, req.FilePath)
	if err != nil {
		return err
	}
	if !b.allowWrite(zone) {
		return fmt.Errorf("fsrestrict: write denied by policy")
	}
	cp := *req
	cp.FilePath = np
	return b.inner.Write(ctx, &cp)
}

func (b *policyBackend) Edit(ctx context.Context, req *filesystem.EditRequest) error {
	np, zone, err := b.resolveWithCtx(ctx, req.FilePath)
	if err != nil {
		return err
	}
	if !b.allowEdit(zone) {
		return fmt.Errorf("fsrestrict: edit denied by policy")
	}
	cp := *req
	cp.FilePath = np
	return b.inner.Edit(ctx, &cp)
}
