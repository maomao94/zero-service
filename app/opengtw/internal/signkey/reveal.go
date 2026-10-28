// Package signkey 提供应用当前签名密钥的读取能力。
//
// HTTP 管理面和本地 keytool 共用这里，避免两边对“当前生效密钥”的
// 判定（应用状态、signRequired、status=1 记录、主密钥解密）出现分叉。
package signkey

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"zero-service/app/opengtw/internal/security"
	"zero-service/model/gormmodel"

	"gorm.io/gorm"
)

var (
	// ErrAppIDRequired 表示没有传 appId。
	ErrAppIDRequired = errors.New("appId 不能为空")
	// ErrAppNotFound 表示应用不存在。
	ErrAppNotFound = errors.New("应用不存在")
	// ErrNotEnabled 表示应用没有启用请求签名，不存在可查看的 signKey。
	ErrNotEnabled = errors.New("应用未启用请求签名")
	// ErrEncryptionKeyMissing 表示配置里没有可用的主密钥。
	ErrEncryptionKeyMissing = errors.New("签名密钥未配置")
	// ErrKeyMissing 表示应用没有当前生效的密钥记录或密文为空。
	ErrKeyMissing = errors.New("应用没有可用的签名密钥")
	// ErrDecrypt 表示密文无法用当前主密钥解开（主密钥变更或数据损坏）。
	ErrDecrypt = errors.New("解密签名密钥失败")
)

// Reveal 返回应用当前生效的 signKey 明文。
//
// 只读取 open_app_secret 中 status=1 的记录：status=2 是重置后等待失效的旧密钥，
// 仍能验签，但不应该作为“当前密钥”交付给调用方。
func Reveal(ctx context.Context, appStore *gormmodel.OpenAppStore, secretStore *gormmodel.OpenAppSecretStore, encryptionKey, appID string) (string, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return "", ErrAppIDRequired
	}

	app, err := appStore.FindByAppId(ctx, appID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrAppNotFound
		}
		return "", err
	}
	if !app.SignRequired {
		return "", ErrNotEnabled
	}
	if len(encryptionKey) < 32 {
		return "", ErrEncryptionKeyMissing
	}

	record, err := secretStore.FindActiveByAppId(ctx, appID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrKeyMissing
		}
		return "", err
	}
	if strings.TrimSpace(record.SignKeyCipher) == "" {
		return "", ErrKeyMissing
	}

	plaintext, err := security.DecryptSignKey(encryptionKey, record.SignKeyCipher)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecrypt, err)
	}
	return plaintext, nil
}
