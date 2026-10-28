package middleware

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"zero-service/app/opengtw/internal/security"
	"zero-service/common/errx"
	"zero-service/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// credentialRefreshInterval 凭据缓存刷新周期，避免逐请求查库。
	credentialRefreshInterval = 60 * time.Second
	// lastUsedTouchInterval 同一应用最近调用时间的落库节流窗口。
	lastUsedTouchInterval = time.Minute
)

// Credential 是鉴权所需的只读应用凭据快照。
type Credential struct {
	AppId        string
	Name         string
	SignRequired bool
	credentials  []credentialSecret
}

type credentialSecret struct {
	Hash          string
	SignKeyCipher string
}

// Match 以常数时间校验密钥是否命中任一有效摘要。
func (c *Credential) Match(secret string) bool {
	for _, credential := range c.credentials {
		if security.VerifySecret(credential.Hash, secret) {
			return true
		}
	}
	return false
}

func (c *Credential) SignKeys(encryptionKey string) []string {
	keys := make([]string, 0, len(c.credentials))
	for _, credential := range c.credentials {
		if credential.SignKeyCipher == "" {
			continue
		}
		key, err := security.DecryptSignKey(encryptionKey, credential.SignKeyCipher)
		if err != nil {
			logx.Errorf("decrypt sign key failed, app_id=%s: %v", c.AppId, err)
			continue
		}
		keys = append(keys, key)
	}
	return keys
}

// CredentialCache 在内存中维护启用应用及其有效密钥，每 60s 全量刷新。
type CredentialCache struct {
	appStore    *gormmodel.OpenAppStore
	secretStore *gormmodel.OpenAppSecretStore
	interval    time.Duration
	now         func() time.Time

	mu       sync.RWMutex
	apps     map[string]*Credential
	lastUsed map[string]time.Time
}

func NewCredentialCache(appStore *gormmodel.OpenAppStore, secretStore *gormmodel.OpenAppSecretStore, interval time.Duration) *CredentialCache {
	if interval <= 0 {
		interval = credentialRefreshInterval
	}
	return &CredentialCache{
		appStore:    appStore,
		secretStore: secretStore,
		interval:    interval,
		now:         time.Now,
		apps:        make(map[string]*Credential),
		lastUsed:    make(map[string]time.Time),
	}
}

// MustRefresh 启动时加载凭据；失败直接终止，避免空缓存对外提供鉴权。
func (c *CredentialCache) MustRefresh(ctx context.Context) {
	if err := c.Refresh(ctx); err != nil {
		logx.Must(err)
	}
}

// Refresh 从 resource 库全量重建凭据快照。
func (c *CredentialCache) Refresh(ctx context.Context) error {
	apps, err := c.appStore.ListActive(ctx)
	if err != nil {
		return err
	}
	secrets, err := c.secretStore.ListUsable(ctx, c.now())
	if err != nil {
		return err
	}

	grouped := make(map[string][]credentialSecret, len(secrets))
	for _, secret := range secrets {
		if _, err := hex.DecodeString(secret.SecretHash); err != nil {
			logx.Errorf("invalid secret hash, app_id=%s id=%d", secret.AppId, secret.Id)
			continue
		}
		grouped[secret.AppId] = append(grouped[secret.AppId], credentialSecret{
			Hash:          secret.SecretHash,
			SignKeyCipher: secret.SignKeyCipher,
		})
	}

	next := make(map[string]*Credential, len(apps))
	for _, app := range apps {
		next[app.AppId] = &Credential{
			AppId:        app.AppId,
			Name:         app.Name,
			SignRequired: app.SignRequired,
			credentials:  grouped[app.AppId],
		}
	}

	c.mu.Lock()
	c.apps = next
	c.mu.Unlock()
	return nil
}

// Run 按固定周期刷新，刷新失败时保留上一份快照。
func (c *CredentialCache) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.Refresh(ctx); err != nil {
				logx.Errorf("refresh open app credentials failed: %v", err)
			}
		}
	}
}

// Get 返回应用凭据快照；不存在或已停用均返回 false。
func (c *CredentialCache) Get(appID string) (*Credential, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cred, ok := c.apps[appID]
	return cred, ok
}

// TouchLastUsed 更新最近调用时间，同一应用每分钟最多落库一次。
func (c *CredentialCache) TouchLastUsed(appID string) {
	now := c.now()

	c.mu.Lock()
	if last, ok := c.lastUsed[appID]; ok && now.Sub(last) < lastUsedTouchInterval {
		c.mu.Unlock()
		return
	}
	c.lastUsed[appID] = now
	c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.appStore.TouchLastUsedAt(ctx, appID, now); err != nil {
		logx.Errorf("touch open app last_used_at failed, app_id=%s: %v", appID, err)
	}
}

// errorMessage 按错误码取对外文案。
func errorMessage(code int) string {
	return errx.ErrorMessage(code)
}
