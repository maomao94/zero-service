// Package antsx 响应式编程工具包
package antsx

import (
	"io"
)

// TeeWriter 管道扇出写入器，将写入的数据同时分发到内部管道和所有附加写入器。
//
// 典型用法：
//
//	// 单目标上传：数据写入 TeeWriter，同时计算 hash，goroutine 从 Reader 读取上传 OSS
//	hash := md5.New()
//	tee := antsx.NewTeeWriter(hash)
//	go uploadOSS(ctx, tee.Reader()) // 从 pipe reader 读取数据
//	io.Copy(tee, sourceReader)       // 写入数据
//	tee.Close()
//
//	// 多目标转推：组合多个 TeeWriter
//	tee1 := antsx.NewTeeWriter()  // 目标 OSS-1
//	tee2 := antsx.NewTeeWriter()  // 目标 OSS-2
//	tee3 := antsx.NewTeeWriter()  // 目标 OSS-3
//	mw := io.MultiWriter(tee1, tee2, tee3)
//	go uploadOSS1(ctx, tee1.Reader())
//	go uploadOSS2(ctx, tee2.Reader())
//	go uploadOSS3(ctx, tee3.Reader())
//	io.Copy(mw, sourceReader)
//	tee1.Close(); tee2.Close(); tee3.Close()
type TeeWriter struct {
	pr  *io.PipeReader
	pw  *io.PipeWriter
	tee io.Writer
}

// NewTeeWriter 创建管道扇出写入器。
// additionalWriters 中的写入器（如 hash.Hash、临时文件等）也会同时收到所有写入的数据。
//
// 写入顺序固定为：内部 pipe writer -> additionalWriters。
// 底层使用 io.MultiWriter，因此任一 writer 返回错误后，Write 会立即返回该错误，后续 writer 不会再收到本次数据。
// 这个语义适合上传链路：只要 OSS reader 或本地副本任一环节失败，本次写入就应该整体失败。
func NewTeeWriter(additionalWriters ...io.Writer) *TeeWriter {
	pr, pw := io.Pipe()
	writers := make([]io.Writer, 1, 1+len(additionalWriters))
	writers[0] = pw
	writers = append(writers, additionalWriters...)
	return &TeeWriter{
		pr:  pr,
		pw:  pw,
		tee: io.MultiWriter(writers...),
	}
}

// Reader 返回管道读端，供 goroutine 读取所有写入的数据。
// Reader 在调用 Close() 后返回 io.EOF。
func (w *TeeWriter) Reader() *io.PipeReader {
	return w.pr
}

// Write 写入数据，数据同时进入管道和所有附加写入器。
// 当任一写入器返回错误时，Write 立即返回该错误（不再向后续写入器写入）。
func (w *TeeWriter) Write(p []byte) (int, error) {
	return w.tee.Write(p)
}

// Close 只关闭内部 pipe 写端，通知 Reader 数据已结束。
// NewTeeWriter 传入的 additionalWriters 不会被自动关闭，调用方需要自行管理这些资源。
func (w *TeeWriter) Close() error {
	return w.pw.Close()
}

// CloseWithError 以指定错误关闭内部 pipe 写端，Reader 将收到该错误。
// NewTeeWriter 传入的 additionalWriters 不会被自动关闭，调用方需要自行管理这些资源。
func (w *TeeWriter) CloseWithError(err error) error {
	return w.pw.CloseWithError(err)
}
