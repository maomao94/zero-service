package errx

import (
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"zero-service/third_party/extproto"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// 平台码表只允许已登记的功能模块。新增 BC 必须先在这里登记，
// 避免随手占用保留段（BC=90-99）或与已有功能域冲突。
var allowedCodeModules = map[int]string{
	0:  "通用",
	30: "开放平台",
}

// 语义 HTTP 只允许这些值。新增状态必须同时更新 httpCodeToGRPCCode，
// 并确认客户端分类、重试策略和文档同步调整。
var allowedSemanticHTTPCodes = map[int]string{
	http.StatusOK:                    "成功",
	http.StatusBadRequest:            "参数或业务请求错误",
	http.StatusUnauthorized:          "未认证",
	http.StatusForbidden:             "无权限",
	http.StatusNotFound:              "资源或路由不存在",
	http.StatusMethodNotAllowed:      "方法不允许",
	http.StatusRequestTimeout:        "请求取消或超时",
	http.StatusConflict:              "状态或数据冲突",
	http.StatusRequestEntityTooLarge: "请求体或文件过大",
	http.StatusUnsupportedMediaType:  "媒体类型或文件类型不支持",
	http.StatusTooManyRequests:       "限流",
	http.StatusInternalServerError:   "服务端内部错误",
	http.StatusNotImplemented:        "功能未实现",
	http.StatusBadGateway:            "网关或第三方接口错误",
	http.StatusServiceUnavailable:    "依赖不可用",
	http.StatusGatewayTimeout:        "下游超时",
}

// 枚举名格式：_<A>_<BC>_<SEMANTIC>。0 的别名单独放行。
var codeIdentifierPattern = regexp.MustCompile(`^_([0-9])_([0-9]{2})_([A-Z0-9_]+)$`)

// TestCodeTableRules 把 extproto 码表的结构规则固化成 CI 约束。
//
// 规则与 docs/error-codes.md 第 3 节一致：
//  1. 非 0 码的标识符前缀必须与数字的 A/BC 一致；
//  2. A 只允许 0（成功）和 1（平台服务端）；
//  3. BC 必须是已登记功能域；
//  4. DEF 必须落在 001-999，000 保留；
//  5. 每个码必须带非空 name 和 http_code 扩展；
//  6. 只有 0 可以使用 200，错误码只能是 4xx/5xx；
//  7. http_code 必须来自允许集合。
func TestCodeTableRules(t *testing.T) {
	values := extproto.File_extproto_proto.Enums().Get(0).Values()
	for i := 0; i < values.Len(); i++ {
		value := values.Get(i)
		t.Run(string(value.Name()), func(t *testing.T) {
			assertCodeTableValue(t, value)
		})
	}
}

// TestCodeTableNumbersAreUnique 数字一旦被占用就不能再分配给新语义。
func TestCodeTableNumbersAreUnique(t *testing.T) {
	values := extproto.File_extproto_proto.Enums().Get(0).Values()
	seen := make(map[int32]string, values.Len())
	for i := 0; i < values.Len(); i++ {
		value := values.Get(i)
		number := int32(value.Number())
		if previous, exists := seen[number]; exists {
			t.Fatalf("错误码 %d 同时被 %s 和 %s 占用", number, previous, value.Name())
		}
		seen[number] = string(value.Name())
	}
}

// TestCodeTableDisplayNamesAreUnique 保证 ErrorMessage 不会出现一个文案对应多个码。
func TestCodeTableDisplayNamesAreUnique(t *testing.T) {
	values := extproto.File_extproto_proto.Enums().Get(0).Values()
	seen := make(map[string]int32, values.Len())
	for i := 0; i < values.Len(); i++ {
		value := values.Get(i)
		display, _ := codeTableExtensions(t, value)
		if previous, exists := seen[display]; exists && previous != int32(value.Number()) {
			t.Fatalf("展示文案 %q 同时用于错误码 %d 和 %d", display, previous, value.Number())
		}
		seen[display] = int32(value.Number())
	}
}

func assertCodeTableValue(t *testing.T, value protoreflect.EnumValueDescriptor) {
	t.Helper()

	number := int(value.Number())
	identifier := string(value.Name())
	_, httpCode := codeTableExtensions(t, value)

	if _, ok := allowedSemanticHTTPCodes[httpCode]; !ok {
		t.Fatalf("%s 的 http_code=%d 不在允许集合 %v 内",
			identifier, httpCode, slices.Sorted(maps.Keys(allowedSemanticHTTPCodes)))
	}

	if number == 0 {
		if identifier != "_0_OK" {
			t.Fatalf("成功码只允许 _0_OK，got %s", identifier)
		}
		if httpCode != http.StatusOK {
			t.Fatalf("成功码 %s 的 http_code=%d，必须是 200", identifier, httpCode)
		}
		return
	}

	if httpCode == http.StatusOK {
		t.Fatalf("%s 是错误码，http_code 不能是 200", identifier)
	}
	if httpCode < http.StatusBadRequest || httpCode >= 600 {
		t.Fatalf("%s 的 http_code=%d，错误码只能是 4xx/5xx", identifier, httpCode)
	}

	matches := codeIdentifierPattern.FindStringSubmatch(identifier)
	if matches == nil {
		t.Fatalf("%s 不符合 _<A>_<BC>_<SEMANTIC> 命名规范", identifier)
	}
	source, _ := strconv.Atoi(matches[1])
	module, _ := strconv.Atoi(matches[2])
	actualSource := number / 100000
	actualModule := (number / 1000) % 100
	if source != actualSource {
		t.Fatalf("%s 的来源位 A=%d，与数字 %d 的 A=%d 不一致", identifier, source, number, actualSource)
	}
	if module != actualModule {
		t.Fatalf("%s 的功能模块 BC=%02d，与数字 %d 的 BC=%02d 不一致", identifier, module, number, actualModule)
	}
	if source != 1 {
		t.Fatalf("%s 的来源位 A=%d，非 0 码只允许 A=1（平台服务端）", identifier, source)
	}
	if _, ok := allowedCodeModules[module]; !ok {
		t.Fatalf("%s 使用了未登记的功能模块 BC=%02d，已登记模块：%v",
			identifier, module, allowedCodeModules)
	}
	if def := number % 1000; def == 0 {
		t.Fatalf("%s 的 DEF=%03d，模块内错误必须落在 001-999", identifier, def)
	}
}

// codeTableExtensions 读取并校验枚举值上的 name 与 http_code 扩展。
func codeTableExtensions(t *testing.T, value protoreflect.EnumValueDescriptor) (string, int) {
	t.Helper()

	options := value.Options()
	if options == nil {
		t.Fatalf("%s 缺少枚举值选项", value.Name())
	}
	if !proto.HasExtension(options, extproto.E_Name) {
		t.Fatalf("%s 缺少 name 扩展", value.Name())
	}
	if !proto.HasExtension(options, extproto.E_HttpCode) {
		t.Fatalf("%s 缺少 http_code 扩展", value.Name())
	}

	display, _ := proto.GetExtension(options, extproto.E_Name).(string)
	if display == "" || strings.TrimSpace(display) != display {
		t.Fatalf("%s 的 name 必须是非空且首尾无空白的展示文案，got %q", value.Name(), display)
	}
	httpCode, _ := proto.GetExtension(options, extproto.E_HttpCode).(int32)

	return display, int(httpCode)
}

// 保证测试本身覆盖了成功码和所有已登记模块，避免码表被清空后规则测试空跑通过。
func TestCodeTableCoversKnownEntries(t *testing.T) {
	values := extproto.File_extproto_proto.Enums().Get(0).Values()
	if values.Len() == 0 {
		t.Fatal("extproto Code 枚举为空")
	}

	seen := make(map[int]bool)
	for i := 0; i < values.Len(); i++ {
		seen[int(values.Get(i).Number())] = true
	}
	for _, number := range []int{0, int(extproto.Code__1_00_PARAM), int(extproto.Code__1_30_APP_NOT_FOUND)} {
		if !seen[number] {
			t.Fatalf("码表缺少基础码 %d，规则测试未覆盖真实码表", number)
		}
	}
}
