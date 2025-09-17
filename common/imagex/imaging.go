package imagex

import (
	"bytes"
	"image"
	"io"
	"os"

	"github.com/disintegration/imaging"
)

// coreResize 通用方法：输入 image.Image，输出到 io.Writer
func coreResize(img image.Image, width, height int, format imaging.Format, w io.Writer) error {
	thumb := imaging.Fit(img, width, height, imaging.Lanczos)
	return imaging.Encode(w, thumb, format)
}

// FromFileToFile 文件路径输入 → 文件路径输出
func FromFileToFile(inputPath, outputPath string, width, height int) error {
	src, err := imaging.Open(inputPath)
	if err != nil {
		return err
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return coreResize(src, width, height, imaging.JPEG, outFile)
}

// FromFileToBytes 文件路径输入 → 返回字节流
func FromFileToBytes(inputPath string, width, height int, format imaging.Format) ([]byte, error) {
	src, err := imaging.Open(inputPath)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = coreResize(src, width, height, format, &buf)
	return buf.Bytes(), err
}

// FromBytesToBytes 字节流输入 → 返回字节流
func FromBytesToBytes(data []byte, width, height int, format imaging.Format) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = coreResize(img, width, height, format, &buf)
	return buf.Bytes(), err
}

// FromReaderToReader io.Reader 输入 → 返回 io.Reader
func FromReaderToReader(r io.Reader, width, height int, format imaging.Format) (io.Reader, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = coreResize(img, width, height, format, &buf)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(buf.Bytes()), nil
}
