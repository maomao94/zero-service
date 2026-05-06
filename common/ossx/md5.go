package ossx

import (
	"io"

	"zero-service/common/filex"
)

// UploadWithMD5 用统一方式为上传读流增加 MD5 装饰。
// 不同 OSS 实现只需提供 upload 回调，即可复用同一套 MD5 计算逻辑。
func UploadWithMD5(reader io.Reader, upload func(reader io.Reader) error) (string, error) {
	md5Reader, digest := filex.NewMD5TeeReader(reader)
	if err := upload(md5Reader); err != nil {
		return "", err
	}
	return digest.Hex(), nil
}
