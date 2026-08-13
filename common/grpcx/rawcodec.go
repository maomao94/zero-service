// Package grpcx provides shared gRPC helpers.
package grpcx

import (
	"fmt"

	"zero-service/common/tool"

	"google.golang.org/grpc/encoding"
)

// RawCodec adapts protobuf or raw-byte requests to gRPC and appends response
// bytes to a caller-provided *[]byte.
type RawCodec struct {
	name string
}

const defaultRawCodecName = "raw"

var _ encoding.Codec = RawCodec{}

// NewRawCodec creates a raw codec with the content subtype name sent to gRPC.
// An empty name uses "raw", which is also the zero-value codec's name.
func NewRawCodec(name string) RawCodec {
	if name == "" {
		name = defaultRawCodecName
	}
	return RawCodec{name: name}
}

// Marshal encodes v using the project's protobuf-or-bytes conversion rules.
func (c RawCodec) Marshal(v any) ([]byte, error) {
	return tool.ToProtoBytes(v)
}

// Unmarshal appends data to v, which must be a *[]byte.
func (c RawCodec) Unmarshal(data []byte, v any) error {
	target, ok := v.(*[]byte)
	if !ok || target == nil {
		return fmt.Errorf("please pass in *[]byte")
	}
	*target = append(*target, data...)
	return nil
}

// Name returns the configured gRPC codec name.
func (c RawCodec) Name() string {
	if c.name == "" {
		return defaultRawCodecName
	}
	return c.name
}
