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

// CountingReader 包装 io.Reader 并统计实际读取的字节数。
type CountingReader struct {
	reader io.Reader
	n      int64
}

// NewCountingReader 构造一个计数 Reader。
func NewCountingReader(reader io.Reader) *CountingReader {
	return &CountingReader{reader: reader}
}

func (c *CountingReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.n += int64(n)
	return n, err
}

// N 返回截至目前已读取的字节数。
func (c *CountingReader) N() int64 {
	return c.n
}

// UploadWithMD5 在上传读流时同步计算内容 MD5。
// upload 回调会收到带 MD5 装饰的 reader，上传完成后返回十六进制 MD5。
func UploadWithMD5(reader io.Reader, upload func(reader io.Reader) error) (string, error) {
	md5Reader, digest := NewMD5TeeReader(reader)
	if err := upload(md5Reader); err != nil {
		return "", err
	}
	return digest.Hex(), nil
}

// UploadWithMD5Size 在 UploadWithMD5 基础上统计实际读取字节数，
// 供无法提前获知大小的上传（如流式上传）在上传后回填 size。
func UploadWithMD5Size(reader io.Reader, upload func(reader io.Reader) error) (string, int64, error) {
	counter := NewCountingReader(reader)
	md5Hex, err := UploadWithMD5(counter, upload)
	if err != nil {
		return "", 0, err
	}
	return md5Hex, counter.N(), nil
}
