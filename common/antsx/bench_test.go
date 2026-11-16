package antsx_test

import (
	"testing"

	"zero-service/common/antsx"
)

// benchPayload 是刻意不带指针的结构体，用于暴露泛型 any 装箱产生的堆分配。
type benchPayload struct {
	A, B, C int64
}

const benchElements = 1000

// BenchmarkStreamReader_Convert 测量 Convert 链路每 1000 个元素的成本。
// allocs/op 应保持在个位数（每个算子链若干个闭包），不应随元素数量增长。
func BenchmarkStreamReader_Convert(b *testing.B) {
	payloads := make([]benchPayload, benchElements)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converted := antsx.StreamReaderFromArray(payloads).Convert(
			func(p benchPayload) (benchPayload, error) { return p, nil })
		for {
			if _, err := converted.Recv(); err != nil {
				break
			}
		}
	}
}

// BenchmarkStreamReader_Direct 是同一份数据的直接读取基线。
func BenchmarkStreamReader_Direct(b *testing.B) {
	payloads := make([]benchPayload, benchElements)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sr := antsx.StreamReaderFromArray(payloads)
		for {
			if _, err := sr.Recv(); err != nil {
				break
			}
		}
	}
}

// BenchmarkStreamReader_OperatorChain 测量 Convert+Filter+Map 三段链路的成本。
func BenchmarkStreamReader_OperatorChain(b *testing.B) {
	payloads := make([]benchPayload, benchElements)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := antsx.StreamReaderFromArray(payloads).
			Convert(func(p benchPayload) (int64, error) { return p.A, nil }).
			Filter(func(v int64) bool { return v%2 == 0 }).
			Map(func(v int64) int64 { return v * 2 })
		for {
			if _, err := out.Recv(); err != nil {
				break
			}
		}
	}
}

// BenchmarkMergeStreamReaders_Arrays 测量纯数组合并后的读取成本（走游标快路径）。
func BenchmarkMergeStreamReaders_Arrays(b *testing.B) {
	a := make([]benchPayload, benchElements/2)
	c := make([]benchPayload, benchElements/2)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		merged := antsx.MergeStreamReaders([]*antsx.StreamReader[benchPayload]{
			antsx.StreamReaderFromArray(a),
			antsx.StreamReaderFromArray(c),
		})
		for {
			if _, err := merged.Recv(); err != nil {
				break
			}
		}
	}
}
