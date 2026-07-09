package ftps

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/jlaffaye/ftp"
)

// TLSMode selects how the FTP connection is upgraded to TLS.
type TLSMode string

const (
	// TLSModeImplicit connects with TLS from the first byte, typically on port 990.
	TLSModeImplicit TLSMode = "implicit"
	// TLSModeExplicit connects in plain FTP first, then upgrades with AUTH TLS.
	TLSModeExplicit TLSMode = "explicit"
)

// Config contains the reusable FTP/FTPS upload settings.
type Config struct {
	Address            string
	Username           string
	Password           string
	RemoteDir          string
	TLSMode            TLSMode
	InsecureSkipVerify bool
	Timeout            time.Duration
	DisableEPSV        bool
	UseTemporaryFile   bool
}

// Uploader uploads local or streamed content to an FTP/FTPS server.
type Uploader struct {
	cfg Config
}

// NewUploader creates an uploader with conservative defaults for timeout and TLS mode.
func NewUploader(cfg Config) *Uploader {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.TLSMode == "" {
		cfg.TLSMode = TLSModeImplicit
	}
	return &Uploader{cfg: cfg}
}

// Config returns the effective uploader configuration after defaults are applied.
func (u *Uploader) Config() Config {
	return u.cfg
}

// UploadResult describes the uploaded remote file.
type UploadResult struct {
	RemotePath string
	Size       int64
}

// UploadFile opens a local file and uploads it as remoteName, defaulting to the local base name.
func (u *Uploader) UploadFile(ctx context.Context, localPath, remoteName string) (UploadResult, error) {
	if u.cfg.Address == "" {
		return UploadResult{}, fmt.Errorf("ftps address is required")
	}
	if remoteName == "" {
		remoteName = path.Base(localPath)
	}
	file, err := os.Open(localPath)
	if err != nil {
		return UploadResult{}, fmt.Errorf("open local file: %w", err)
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return UploadResult{}, fmt.Errorf("stat local file: %w", err)
	}
	return u.Upload(ctx, remoteName, file, stat.Size())
}

// Upload streams content to remoteName under Config.RemoteDir.
func (u *Uploader) Upload(ctx context.Context, remoteName string, r io.Reader, size int64) (UploadResult, error) {
	if remoteName == "" {
		return UploadResult{}, fmt.Errorf("remote name is required")
	}
	client, err := u.dial(ctx)
	if err != nil {
		return UploadResult{}, err
	}
	defer client.Quit()
	if err := client.Login(u.cfg.Username, u.cfg.Password); err != nil {
		return UploadResult{}, fmt.Errorf("ftps login failed: %w", err)
	}

	remotePath := path.Join(u.cfg.RemoteDir, remoteName)
	storePath := remotePath
	if u.cfg.UseTemporaryFile {
		storePath = remotePath + ".uploading"
		_ = client.Delete(storePath)
	}
	if err := client.Stor(storePath, r); err != nil {
		return UploadResult{}, fmt.Errorf("ftps upload failed: %w", err)
	}
	if size >= 0 {
		remoteSize, err := client.FileSize(storePath)
		if err != nil {
			return UploadResult{}, fmt.Errorf("ftps stat uploaded file failed: %w", err)
		}
		if remoteSize != size {
			return UploadResult{}, fmt.Errorf("ftps uploaded size mismatch: local=%d remote=%d", size, remoteSize)
		}
	}
	if u.cfg.UseTemporaryFile {
		_ = client.Delete(remotePath)
		if err := client.Rename(storePath, remotePath); err != nil {
			return UploadResult{}, fmt.Errorf("ftps rename uploaded file failed: %w", err)
		}
	}
	return UploadResult{RemotePath: remotePath, Size: size}, nil
}

func (u *Uploader) dial(ctx context.Context) (*ftp.ServerConn, error) {
	tlsConfig := &tls.Config{InsecureSkipVerify: u.cfg.InsecureSkipVerify}
	opts := []ftp.DialOption{
		ftp.DialWithContext(ctx),
		ftp.DialWithTimeout(u.cfg.Timeout),
	}
	if u.cfg.DisableEPSV {
		opts = append(opts, ftp.DialWithDisabledEPSV(true))
	}
	switch u.cfg.TLSMode {
	case TLSModeImplicit:
		opts = append(opts, ftp.DialWithTLS(tlsConfig))
	case TLSModeExplicit:
		opts = append(opts, ftp.DialWithExplicitTLS(tlsConfig))
	default:
		return nil, fmt.Errorf("unsupported ftps tls mode %q", u.cfg.TLSMode)
	}
	client, err := ftp.Dial(u.cfg.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("ftps dial failed: %w", err)
	}
	return client, nil
}
