// Package security 集中实现开放平台的凭据生成、摘要校验和签名密钥加解密。
//
// 这里有两种不同性质的密码学操作，不能混用：
//  1. secret 只需要“单向校验”：数据库只保存 sha256(secret)，每次请求重新计算摘要并比较，
//     服务端永远不需要知道 secret 明文；
//  2. signKey 服务端必须拿到明文才能重新计算 HMAC，所以用主密钥加密后保存，读取时解密。
//
// 前者用哈希，后者用“可逆加密 + 防篡改”，原因见各函数注释。
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/google/uuid"
)

const (
	// 前缀只用于人工识别和日志排查，服务端不会解析或去掉它；
	// appId、secret、signKey 在校验和签名时都按完整字符串处理。
	appIDPrefix   = "app_"
	secretPrefix  = "sec_"
	signKeyPrefix = "sk_"
)

// NewAppID 生成 app_ + 32 位十六进制（UUID v4 去掉横杠）的应用 ID，共 36 个字符。
//
// UUID v4 有 122 位随机数，实际使用中不需要考虑碰撞；即使极端情况撞上，
// open_app.app_id 的主键约束也会拒绝重复插入，不会产生两个相同应用。
func NewAppID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return appIDPrefix + hex.EncodeToString(id[:]), nil
}

// NewSecret 生成 sec_ + 32 位十六进制的 Basic 密钥（16 字节随机数）。
//
// 16 字节 = 128 位熵，足够抵抗在线/离线暴力枚举；随机源是 crypto/rand，
// 不能用 math/rand，否则可被预测。
func NewSecret() (string, error) {
	return prefixedHex(secretPrefix, 16)
}

// NewSignKey 生成 sk_ + 64 位十六进制的 HMAC 签名密钥（32 字节随机数）。
//
// HMAC-SHA256 的推荐密钥长度就是 32 字节，与摘要长度一致。
func NewSignKey() (string, error) {
	return prefixedHex(signKeyPrefix, 32)
}

// HashSecret 返回 secret 的 sha256 十六进制摘要。
//
// 数据库中只保存这个摘要，不保存 secret 明文。secret 本身是高熵随机串，
// 不是用户口令，所以不需要加盐或慢哈希（bcrypt/argon2）；sha256 的抗预映像性足够。
func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// VerifySecret 以常数时间比较“数据库摘要”和“请求携带的 secret 计算出的摘要”。
//
// 不能直接用 == 比较字符串：== 会在第一个不同字节处提前返回，
// 攻击者可能通过响应时间差逐字节猜测摘要。subtle.ConstantTimeCompare 始终比较完整长度。
func VerifySecret(hash, secret string) bool {
	expected, err := hex.DecodeString(hash)
	if err != nil || len(expected) != sha256.Size {
		// 数据库里的摘要格式非法，直接视为不匹配，避免把脏数据当成有效凭据。
		return false
	}
	actual := sha256.Sum256([]byte(secret))
	return subtle.ConstantTimeCompare(expected, actual[:]) == 1
}

// prefixedHex 生成 size 字节的密码学随机数并转成十六进制，再拼上前缀。
func prefixedHex(prefix string, size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf), nil
}

// EncryptSignKey 使用“AES-256-CBC 加密 + HMAC-SHA256 防篡改”保护签名密钥。
//
// 为什么可逆：验签时服务端必须拿到 signKey 明文，才能重新计算
// HMAC-SHA256(canonical, signKey) 与客户端签名比较。只存哈希无法验签。
//
// 参数 key 是配置中的主密钥（SignKeyEncryptionKey），要求至少 32 字节。
// 主密钥本身不直接用于加密，而是先派生出两把用途隔离的子密钥：
//   - encryptionKey：只用于 AES 加密；
//   - macKey：只用于 HMAC 防篡改。
//
// 存储格式为：v1.<base64(ciphertext)>.<hmac>
//   - v1 是格式版本，未来换算法时可以平滑升级；
//   - ciphertext 是 AES-CBC 的输出（lancet 会在前面拼上随机 IV）；
//   - hmac 是对 "v1.<base64>" 的 HMAC-SHA256，使用 Encrypt-then-MAC 顺序。
func EncryptSignKey(key, plaintext string) (string, error) {
	if len(key) < 32 {
		return "", fmt.Errorf("sign key encryption key must be at least 32 bytes")
	}

	// 从主密钥派生加密子密钥和 MAC 子密钥。
	// deriveCipherKey 内部是 HMAC-SHA256(key=主密钥, msg=purpose)，输出 32 字节。
	// 两个 purpose 不同，保证两把子密钥互不相关（domain separation）。
	encryptionKey, err := deriveCipherKey(key, "opengtw/sign-key/encryption/v1")
	if err != nil {
		return "", err
	}
	macKey, err := deriveCipherKey(key, "opengtw/sign-key/authentication/v1")
	if err != nil {
		return "", err
	}

	// lancet 的 AesCbcEncrypt 内部会：
	//   1. 按 PKCS#7 补位到 16 字节整数倍；
	//   2. 用 crypto/rand 生成 16 字节随机 IV；
	//   3. 返回 IV || AES-CBC 密文。
	// 随机 IV 保证相同明文每次加密结果不同，避免泄露“两个应用用了同一个 signKey”。
	ciphertext := cryptor.AesCbcEncrypt([]byte(plaintext), encryptionKey)

	// RawStdEncoding 不写末尾的 '=' 填充，便于放进 varchar/text 字段。
	encoded := base64.RawStdEncoding.EncodeToString(ciphertext)

	// Encrypt-then-MAC：先有密文，再对“版本号 + 密文”计算 MAC。
	// 解密时必须先验证 MAC 再解密，防止攻击者构造畸形密文触发填充错误（padding oracle）。
	mac := cryptor.HmacSha256("v1."+encoded, string(macKey))
	return "v1." + encoded + "." + mac, nil
}

// DecryptSignKey 验证并解密 EncryptSignKey 生成的签名密钥。
//
// 顺序很重要：先验 MAC，再 base64 解码，最后 AES 解密。
// 如果先解密再验 MAC，错误密文可能触发 CBC 填充错误，攻击者能通过错误差异推断明文。
func DecryptSignKey(key, ciphertext string) (string, error) {
	if len(key) < 32 {
		return "", fmt.Errorf("sign key encryption key must be at least 32 bytes")
	}

	// 格式必须是 v1.<base64>.<hmac>。
	parts := strings.Split(ciphertext, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return "", fmt.Errorf("invalid encrypted sign key format")
	}

	// 用同一把 macKey 重新计算 MAC，并做常数时间比较。
	macKey, err := deriveCipherKey(key, "opengtw/sign-key/authentication/v1")
	if err != nil {
		return "", err
	}
	expectedMAC := cryptor.HmacSha256(parts[0]+"."+parts[1], string(macKey))
	if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expectedMAC)) != 1 {
		return "", fmt.Errorf("encrypted sign key authentication failed")
	}

	// MAC 通过后才解码。长度检查：至少包含 1 个 IV 块 + 1 个密文块，且必须是 16 的整数倍。
	ciphertextBytes, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil || len(ciphertextBytes) < 2*16 || len(ciphertextBytes)%16 != 0 {
		return "", fmt.Errorf("invalid encrypted sign key payload")
	}

	encryptionKey, err := deriveCipherKey(key, "opengtw/sign-key/encryption/v1")
	if err != nil {
		return "", err
	}
	plaintext, err := decryptCBC(ciphertextBytes, encryptionKey)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// deriveCipherKey 用 HMAC-SHA256 从主密钥派生一把 32 字节子密钥。
//
// 这是简化版 KDF（类似 HKDF-Expand 的单次调用）：
//   - key 是主密钥；
//   - purpose 是用途标签，作为 HMAC 的消息；
//   - 输出十六进制字符串后再解码成 32 字节，直接作为 AES-256 密钥或 HMAC 密钥。
//
// 注意 HMAC 的参数顺序是 HmacSha256(message, key)，这里 message=purpose，key=masterKey。
func deriveCipherKey(masterKey, purpose string) ([]byte, error) {
	derived, err := hex.DecodeString(cryptor.HmacSha256(purpose, masterKey))
	if err != nil {
		return nil, err
	}
	return derived, nil
}

// decryptCBC 调用第三方 AES-CBC 解密，并把 panic 转成 error。
//
// lancet 在密钥长度非法或密文长度非法时会 panic；虽然调用前已经做了长度检查，
// 这里仍兜底 recover，避免脏数据导致整个服务进程崩溃。
func decryptCBC(ciphertext, key []byte) (plaintext []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			plaintext = nil
			err = fmt.Errorf("decrypt sign key failed")
		}
	}()
	return cryptor.AesCbcDecrypt(ciphertext, key), nil
}
