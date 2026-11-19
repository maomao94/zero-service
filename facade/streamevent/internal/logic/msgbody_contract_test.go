package logic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"zero-service/common/iec104/types"
	"zero-service/facade/streamevent/streamevent"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// bodyContractCases 是 gRPC 通道 bodyRaw 契约的全部信息体用例。
//
// 新增 ASDU 信息体时需在此登记：
//   - TestMsgBodyCaseCoverage 校验 streamevent.proto 中的信息体 message 是否都已登记；
//   - TestMsgBodyDocsInSync 校验 docs/iec104/iec104-message.md 的结构体与字段表是否已登记。
var bodyContractCases = []struct {
	name string
	body any
	pb   proto.Message
}{
	{
		name: "SinglePointInfo",
		body: &types.SinglePointInfo{
			Ioa: 2001, Value: true, Qds: 1, QdsDesc: "QDS(00000001)[OV]",
			Ov: true, Bl: false, Sb: false, Nt: true, Iv: false,
			Time: "2026-06-05 14:30:00.000000",
		},
		pb: &streamevent.SinglePointInfo{},
	},
	{
		name: "DoublePointInfo",
		body: &types.DoublePointInfo{
			Ioa: 2002, Value: 2, Qds: 0, QdsDesc: "QDS(00000000)[]",
			Ov: false, Bl: false, Sb: false, Nt: false, Iv: false,
			Time: "2026-06-05 14:30:00.000000",
		},
		pb: &streamevent.DoublePointInfo{},
	},
	{
		name: "MeasuredValueScaledInfo",
		body: &types.MeasuredValueScaledInfo{
			Ioa: 3001, Value: -1234, Qds: 0, QdsDesc: "QDS(00000000)[]",
			Ov: false, Bl: false, Sb: false, Nt: false, Iv: false,
			Time: "2026-06-05 14:30:00.000000",
		},
		pb: &streamevent.MeasuredValueScaledInfo{},
	},
	{
		name: "MeasuredValueNormalInfo",
		body: &types.MeasuredValueNormalInfo{
			Ioa: 3002, Value: -32768, Nva: 0.5, Qds: 0, QdsDesc: "QDS(00000000)[]",
			Ov: false, Bl: false, Sb: false, Nt: false, Iv: false,
			Time: "2026-06-05 14:30:00.000000",
		},
		pb: &streamevent.MeasuredValueNormalInfo{},
	},
	{
		name: "StepPositionInfo",
		body: &types.StepPositionInfo{
			Ioa: 4001, Value: types.StepPosition{Val: -64, HasTransient: true},
			Qds: 0, QdsDesc: "QDS(00000000)[]",
			Ov: false, Bl: false, Sb: false, Nt: false, Iv: false,
			Time: "2026-06-05 14:30:00.000000",
		},
		pb: &streamevent.StepPositionInfo{},
	},
	{
		name: "BitString32Info",
		body: &types.BitString32Info{
			Ioa: 5001, Value: 4294967295, Qds: 0, QdsDesc: "QDS(00000000)[]",
			Ov: false, Bl: false, Sb: false, Nt: false, Iv: false,
			Time: "2026-06-05 14:30:00.000000",
		},
		pb: &streamevent.BitString32Info{},
	},
	{
		name: "MeasuredValueFloatInfo",
		body: &types.MeasuredValueFloatInfo{
			Ioa: 6001, Value: 220.5, Qds: 0, QdsDesc: "QDS(00000000)[]",
			Ov: false, Bl: false, Sb: false, Nt: false, Iv: false,
			Time: "2026-06-05 14:30:00.000000",
		},
		pb: &streamevent.MeasuredValueFloatInfo{},
	},
	{
		name: "BinaryCounterReadingInfo",
		body: &types.BinaryCounterReadingInfo{
			Ioa: 7001,
			Value: types.BinaryCounterReading{
				CounterReading: -2147483648, SeqNumber: 31,
				HasCarry: true, IsAdjusted: true, IsInvalid: false,
			},
			Time: "2026-06-05 14:30:00.000000",
		},
		pb: &streamevent.BinaryCounterReadingInfo{},
	},
	{
		name: "EventOfProtectionEquipmentInfo",
		body: &types.EventOfProtectionEquipmentInfo{
			Ioa: 8001, Event: 2, Qdp: 1, QdpDesc: "QDP(00000001)[EI]",
			Ei: true, Bl: false, Sb: false, Nt: false, Iv: false,
			Msec: 59999, Time: "2026-06-05 14:30:00.000000",
		},
		pb: &streamevent.EventOfProtectionEquipmentInfo{},
	},
	{
		name: "PackedStartEventsOfProtectionEquipmentInfo",
		body: &types.PackedStartEventsOfProtectionEquipmentInfo{
			Ioa: 9001, Event: 1, Qdp: 0, QdpDesc: "QDP(00000000)[]",
			Ei: false, Bl: true, Sb: false, Nt: false, Iv: false,
			Msec: 1000, Time: "2026-06-05 14:30:00.000000",
		},
		pb: &streamevent.PackedStartEventsOfProtectionEquipmentInfo{},
	},
	{
		name: "PackedOutputCircuitInfoInfo",
		body: &types.PackedOutputCircuitInfoInfo{
			Ioa: 10001, Oci: 255, Gc: true, Cl1: true, Cl2: false, Cl3: true,
			Qdp: 0, QdpDesc: "QDP(00000000)[]",
			Ei: false, Bl: false, Sb: false, Nt: false, Iv: false,
			Msec: 500, Time: "2026-06-05 14:30:00.000000",
		},
		pb: &streamevent.PackedOutputCircuitInfo{},
	},
	{
		name: "PackedSinglePointWithSCDInfo",
		body: &types.PackedSinglePointWithSCDInfo{
			Ioa: 11001, Scd: 4294967295, Stn: "1", Cdn: "0",
			Qds: 0, QdsDesc: "QDS(00000000)[]",
			Ov: false, Bl: false, Sb: false, Nt: false, Iv: false,
		},
		pb: &streamevent.PackedSinglePointWithSCDInfo{},
	},
}

// TestMsgBodyRawContract 锁定 bodyRaw 的跨端契约：同一条 body JSON
// 既能被消费方 json.Unmarshal 转成 map，也能 protojson.Unmarshal 转成
// streamevent 包中对应的强类型 message。
//
// 断言内容：
//  1. JSON 字段名与 proto 的 json_name 完全一致（严格模式禁止未知字段，
//     字段少声明或拼写不一致都会直接失败）；
//  2. 字段值在 JSON -> PB -> JSON 往返后不丢失、不变形。
//
// 因此 types.MsgBody.Body 的 JSON 标签与 streamevent.proto 的 message 定义
// 必须同频演进；只改其中一处，本测试会失败。文档是否同步由
// TestMsgBodyDocsInSync 校验，proto 侧是否漏登记由 TestMsgBodyCaseCoverage 校验。
func TestMsgBodyRawContract(t *testing.T) {
	for _, c := range bodyContractCases {
		t.Run(c.name, func(t *testing.T) {
			raw, err := json.Marshal(c.body)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}

			// 路径一：转 map（消费方不使用强类型时的解析方式）。
			var wantMap map[string]any
			if err := json.Unmarshal(raw, &wantMap); err != nil {
				t.Fatalf("unmarshal body to map: %v", err)
			}

			// 路径二：转 PB（严格模式，未知字段直接报错）。
			if err := protojson.Unmarshal(raw, c.pb); err != nil {
				t.Fatalf("protojson 解析失败，proto 字段与 body JSON 不同步: %v", err)
			}

			pbJSON, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(c.pb)
			if err != nil {
				t.Fatalf("marshal pb to json: %v", err)
			}
			var gotMap map[string]any
			if err := json.Unmarshal(pbJSON, &gotMap); err != nil {
				t.Fatalf("unmarshal pb json to map: %v", err)
			}

			if !reflect.DeepEqual(wantMap, gotMap) {
				t.Fatalf("JSON -> PB 往返后字段不一致\nbody JSON : %s\npb JSON   : %s", raw, pbJSON)
			}
		})
	}
}

// TestMsgBodyRawRejectsUnknownField 记录强类型解析的硬约束：
// body 新增字段后必须同步 streamevent.proto，否则默认严格模式会拒绝。
func TestMsgBodyRawRejectsUnknownField(t *testing.T) {
	raw := []byte(`{"ioa":2001,"value":true,"qds":0,"futureField":1}`)
	err := protojson.Unmarshal(raw, &streamevent.SinglePointInfo{})
	if err == nil {
		t.Fatal("默认严格模式应拒绝未知字段 futureField，实际未报错")
	}

	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(raw, &streamevent.SinglePointInfo{}); err != nil {
		t.Fatalf("DiscardUnknown 应容忍未知字段，实际报错: %v", err)
	}
}

// TestMsgBodyCaseCoverage 保证 bodyContractCases 与 streamevent.proto 同步：
// proto 新增信息体 message 却忘记登记用例时直接失败，避免新 ASDU 类型绕过契约校验。
func TestMsgBodyCaseCoverage(t *testing.T) {
	covered := make(map[string]struct{}, len(bodyContractCases))
	for _, c := range bodyContractCases {
		covered[string(c.pb.ProtoReflect().Descriptor().FullName())] = struct{}{}
	}

	// 信息体 message 的判定依据是含 ioa 字段；请求/响应、MsgBody、PointMapping
	// 以及嵌套结构 StepPosition、BinaryCounterReading 都不含 ioa。
	msgs := streamevent.File_streamevent_proto.Messages()
	for i := 0; i < msgs.Len(); i++ {
		m := msgs.Get(i)
		if m.Fields().ByName("ioa") == nil {
			continue
		}
		if _, ok := covered[string(m.FullName())]; !ok {
			t.Errorf("streamevent.%s 是信息体 message，但 bodyContractCases 未登记用例", m.Name())
		}
	}
}

// TestMsgBodyDocsInSync 把文档纳入契约校验，保证
// docs/iec104/iec104-message.md 第 2 节的结构体名单、第 3 节的字段表
// 与 types 的 JSON 标签一致；新增信息体只改代码或只改文档都会失败。
func TestMsgBodyDocsInSync(t *testing.T) {
	docPath := filepath.Join(repoRoot(t), "docs", "iec104", "iec104-message.md")
	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("读取文档失败: %v", err)
	}
	docStructs, docFields := parseDocBodyTables(string(content))

	covered := make(map[string]struct{}, len(bodyContractCases))
	for _, c := range bodyContractCases {
		typ := reflect.TypeOf(c.body)
		if typ.Kind() != reflect.Ptr {
			t.Fatalf("%s 的 body 不是指针类型", c.name)
		}
		typ = typ.Elem()
		structName := typ.Name()
		covered[structName] = struct{}{}

		if _, ok := docStructs[structName]; !ok {
			t.Errorf("文档未登记结构体 %s", structName)
			continue
		}
		if diff := diffFieldSets(jsonFieldNames(typ), docFields[structName]); diff != "" {
			t.Errorf("%s 的文档字段表与 types JSON 标签不一致：%s", structName, diff)
		}
	}

	for name := range docStructs {
		if _, ok := covered[name]; !ok {
			t.Errorf("文档登记了 %s，但 bodyContractCases 没有对应用例", name)
		}
	}
}

// parseDocBodyTables 解析文档第 2 节表格的「Body 结构体」列（结构体名单），
// 以及第 3 节每个 `### 3.x` 小节的字段表（字段名单）。
// 字段表中的嵌套字段（如 `value.val`）按顶层字段名归并。
func parseDocBodyTables(content string) (map[string]struct{}, map[string]map[string]struct{}) {
	structs := make(map[string]struct{})
	fields := make(map[string]map[string]struct{})

	section := ""
	current := ""
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "## "):
			section = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			current = ""
			continue
		case strings.HasPrefix(trimmed, "### "):
			current = ""
			if !strings.HasPrefix(section, "3.") {
				continue
			}
			tokens := backtickTokens(trimmed)
			if len(tokens) == 0 {
				continue
			}
			current = tokens[len(tokens)-1]
			structs[current] = struct{}{}
			fields[current] = make(map[string]struct{})
			continue
		case !strings.HasPrefix(trimmed, "|"):
			continue
		}

		cells := tableCells(trimmed)
		switch {
		case strings.HasPrefix(section, "2.") && len(cells) >= 4:
			// 第 2 节：第 4 列是「Body 结构体」，不推送的类型没有反引号。
			for _, token := range backtickTokens(cells[3]) {
				structs[token] = struct{}{}
			}
		case strings.HasPrefix(section, "3.") && current != "" && len(cells) > 0:
			// 第 3 节：第 1 列是字段名，可能用「、」并列多个，也可能嵌套。
			for _, token := range backtickTokens(cells[0]) {
				fields[current][strings.SplitN(token, ".", 2)[0]] = struct{}{}
			}
		}
	}
	return structs, fields
}

// tableCells 把 Markdown 表格行拆成单元格并去掉首尾空白。
func tableCells(line string) []string {
	line = strings.Trim(strings.TrimSpace(line), "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// backtickTokens 取出字符串中所有反引号包裹的内容。
func backtickTokens(s string) []string {
	var tokens []string
	for {
		start := strings.Index(s, "`")
		if start < 0 {
			return tokens
		}
		rest := s[start+1:]
		end := strings.Index(rest, "`")
		if end < 0 {
			return tokens
		}
		tokens = append(tokens, rest[:end])
		s = rest[end+1:]
	}
}

// jsonFieldNames 返回结构体 JSON 标签中的字段名集合。
func jsonFieldNames(typ reflect.Type) map[string]struct{} {
	names := make(map[string]struct{}, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		tag := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		names[tag] = struct{}{}
	}
	return names
}

// diffFieldSets 以可读形式描述两个字段集合的差异，无差异时返回空串。
func diffFieldSets(want, got map[string]struct{}) string {
	var missing, extra []string
	for name := range want {
		if _, ok := got[name]; !ok {
			missing = append(missing, name)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			extra = append(extra, name)
		}
	}
	if len(missing) == 0 && len(extra) == 0 {
		return ""
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return fmt.Sprintf("文档缺少 %v，文档多出 %v", missing, extra)
}

// repoRoot 从测试工作目录向上查找 go.mod，避免写死相对层级。
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取工作目录失败: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("从 %s 向上未找到 go.mod", dir)
		}
		dir = parent
	}
}
