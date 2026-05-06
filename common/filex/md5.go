package filex

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io"
)

// MD5Digest 保存流式计算过程中的 MD5 状态。
type MD5Digest struct {
	sum hash.Hash
}

// NewMD5TeeReader 返回一个带 MD5 装饰的 Reader 以及摘要对象。
// 调用方将返回的 reader 传给上传逻辑，上传结束后通过 digest.Hex() 读取摘要值。
func NewMD5TeeReader(reader io.Reader) (io.Reader, *MD5Digest) {
	digest := &MD5Digest{sum: md5.New()}
	return io.TeeReader(reader, digest.sum), digest
}

// Hex 返回当前累积数据的十六进制 MD5 字符串。
func (d *MD5Digest) Hex() string {
	if d == nil || d.sum == nil {
		return ""
	}
	return fmt.Sprintf("%x", d.sum.Sum(nil))
}
