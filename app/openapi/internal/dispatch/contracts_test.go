package dispatch

import (
	"context"
	"errors"
	"strings"
	"testing"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/gis/gis"
	"zero-service/common/errx"
	"zero-service/third_party/extproto"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type recordingConn struct {
	method string
	input  proto.Message
}

func (c *recordingConn) Invoke(_ context.Context, method string, args, _ any, _ ...grpc.CallOption) error {
	c.method, c.input = method, args.(proto.Message)
	return nil
}

func (c *recordingConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("unexpected streaming RPC")
}

func TestRegistryCoversRPCContracts(t *testing.T) {
	conn := &recordingConn{}
	modbus := bridgemodbus.NewBridgeModbusClient(conn)
	registry := NewRegistry(nil, modbus, gis.NewGisClient(conn))
	for _, service := range []struct {
		prefix string
		desc   protoreflect.ServiceDescriptor
	}{
		{"modbus", bridgemodbus.File_bridgemodbus_proto.Services().ByName("BridgeModbus")},
		{"gis", gis.File_gis_proto.Services().ByName("Gis")},
	} {
		for i := 0; i < service.desc.Methods().Len(); i++ {
			rpc := service.desc.Methods().Get(i)
			name := string(rpc.Name())
			key := service.prefix + "." + strings.ToLower(name[:1]) + name[1:]
			t.Run(key, func(t *testing.T) {
				conn.method = ""
				res, err := registry.Invoke(context.Background(), key, `{}`)
				if err != nil {
					t.Fatal(err)
				}
				want := "/" + string(service.desc.FullName()) + "/" + name
				if conn.method != want || conn.input.ProtoReflect().Descriptor() != rpc.Input() || res.ProtoReflect().Descriptor() != rpc.Output() {
					t.Fatalf("incorrect RPC or message types for %s, invoked %s", want, conn.method)
				}
			})
		}
	}
	_, err := registry.Invoke(context.Background(), MethodModbusBatchConvertValue32ToRegisters, `{"values":[220.5],"dataType":"VALUE32_FLOAT32","wordOrder":"WORD_ORDER_CDAB"}`)
	want := &bridgemodbus.BatchConvertValue32ToRegistersReq{Values: []float64{220.5}, DataType: bridgemodbus.Value32Type_VALUE32_FLOAT32, WordOrder: bridgemodbus.WordOrder_WORD_ORDER_CDAB}
	if err != nil || !proto.Equal(conn.input, want) {
		t.Fatalf("request did not reach downstream intact: %v, %v", conn.input, err)
	}

	withoutGIS := NewRegistry(nil, modbus, nil)
	if _, err := withoutGIS.Invoke(context.Background(), MethodGISDistance, `{}`); !errx.IsErrorByPbCode(err, extproto.Code__1_30_METHOD_NOT_OPEN) {
		t.Fatalf("%s: expected method not open, got %v", MethodGISDistance, err)
	}
}
