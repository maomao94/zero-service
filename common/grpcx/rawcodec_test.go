package grpcx

import (
	"bytes"
	"testing"

	"zero-service/common/tool"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestRawCodecName(t *testing.T) {
	tests := []struct {
		name  string
		codec RawCodec
		want  string
	}{
		{name: "configured", codec: NewRawCodec("proto_raw"), want: "proto_raw"},
		{name: "empty name", codec: NewRawCodec(""), want: "raw"},
		{name: "zero value", codec: RawCodec{}, want: "raw"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.codec.Name(); got != tt.want {
				t.Fatalf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRawCodecMarshalMatchesToProtoBytes(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "protobuf message", value: wrapperspb.String("payload")},
		{name: "raw bytes", value: []byte("payload")},
	}

	codec := NewRawCodec("test_raw")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, wantErr := tool.ToProtoBytes(tt.value)
			got, gotErr := codec.Marshal(tt.value)
			if (gotErr != nil) != (wantErr != nil) {
				t.Fatalf("Marshal() error = %v, ToProtoBytes() error = %v", gotErr, wantErr)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("Marshal() = %v, want %v", got, want)
			}
		})
	}
}

func TestRawCodecUnmarshalAppends(t *testing.T) {
	codec := NewRawCodec("test_raw")
	got := []byte("existing-")
	if err := codec.Unmarshal([]byte("first-"), &got); err != nil {
		t.Fatalf("first Unmarshal() error = %v", err)
	}
	if err := codec.Unmarshal([]byte("second"), &got); err != nil {
		t.Fatalf("second Unmarshal() error = %v", err)
	}
	if want := []byte("existing-first-second"); !bytes.Equal(got, want) {
		t.Fatalf("Unmarshal() result = %q, want %q", got, want)
	}
}

func TestRawCodecUnmarshalRejectsOtherTargets(t *testing.T) {
	codec := NewRawCodec("test_raw")
	tests := []struct {
		name   string
		target any
	}{
		{name: "non-pointer slice", target: []byte{}},
		{name: "nil byte slice pointer", target: (*[]byte)(nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := codec.Unmarshal([]byte("payload"), tt.target); err == nil {
				t.Fatal("Unmarshal() error = nil, want an invalid target error")
			}
		})
	}
}
