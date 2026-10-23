package dispatch

import (
	"context"
	"errors"
	"testing"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/podengine/podengine"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"google.golang.org/grpc"
)

type fakePodEngine struct {
	podengine.PodEngineClient
	listRes *podengine.ListPodsRes
	listErr error
}

func (f *fakePodEngine) ListPods(ctx context.Context, in *podengine.ListPodsReq, opts ...grpc.CallOption) (*podengine.ListPodsRes, error) {
	return f.listRes, f.listErr
}

type fakeBridgeModbus struct {
	bridgemodbus.BridgeModbusClient
	readRes  *bridgemodbus.ReadHoldingRegistersRes
	writeRes *bridgemodbus.WriteSingleCoilRes
}

func (f *fakeBridgeModbus) ReadHoldingRegisters(ctx context.Context, in *bridgemodbus.ReadHoldingRegistersReq, opts ...grpc.CallOption) (*bridgemodbus.ReadHoldingRegistersRes, error) {
	return f.readRes, nil
}

func (f *fakeBridgeModbus) WriteSingleCoil(ctx context.Context, in *bridgemodbus.WriteSingleCoilReq, opts ...grpc.CallOption) (*bridgemodbus.WriteSingleCoilRes, error) {
	return f.writeRes, nil
}

func TestRegistryUnknownMethod(t *testing.T) {
	registry := NewRegistry(&fakePodEngine{}, nil)
	_, err := registry.Invoke(context.Background(), "pod.unknown", []byte(`{}`))
	if !tool.IsErrorByPbCode(err, extproto.Code__1_30_METHOD_NOT_OPEN) {
		t.Fatalf("error = %v, want 130103", err)
	}
}

func TestRegistryInvalidData(t *testing.T) {
	registry := NewRegistry(&fakePodEngine{}, nil)
	for _, data := range []string{`{"limit":"abc"}`, `{"unknownField":1}`, `{`} {
		_, err := registry.Invoke(context.Background(), MethodPodList, []byte(data))
		if !tool.IsErrorByPbCode(err, extproto.Code__1_00_PARAM) {
			t.Fatalf("data %s error = %v, want 100101", data, err)
		}
	}
}

func TestRegistryEmptyData(t *testing.T) {
	registry := NewRegistry(&fakePodEngine{listRes: &podengine.ListPodsRes{Total: 0}}, nil)
	msg, err := registry.Invoke(context.Background(), MethodPodList, nil)
	if err != nil {
		t.Fatalf("invoke error = %v", err)
	}
	if msg.(*podengine.ListPodsRes).Total != 0 {
		t.Fatalf("unexpected response: %v", msg)
	}
}

func TestRegistryRelaysPodList(t *testing.T) {
	registry := NewRegistry(&fakePodEngine{listRes: &podengine.ListPodsRes{Total: 1}}, nil)
	msg, err := registry.Invoke(context.Background(), MethodPodList, []byte(`{"limit":100}`))
	if err != nil {
		t.Fatalf("invoke error = %v", err)
	}
	if msg.(*podengine.ListPodsRes).Total != 1 {
		t.Fatalf("unexpected response: %v", msg)
	}
}

func TestRegistryPropagatesDownstreamError(t *testing.T) {
	registry := NewRegistry(&fakePodEngine{listErr: errors.New("boom")}, nil)
	if _, err := registry.Invoke(context.Background(), MethodPodList, []byte(`{}`)); err == nil {
		t.Fatal("expected downstream error")
	}
}

func TestMockRegistryAllMethods(t *testing.T) {
	registry := NewMockRegistry()
	if len(registry.routes) == 0 {
		t.Fatal("mock registry has no routes")
	}
	for method := range registry.routes {
		t.Run(method, func(t *testing.T) {
			msg, err := registry.Invoke(context.Background(), method, []byte(`{}`))
			if err != nil {
				t.Fatalf("invoke error = %v", err)
			}
			data, err := Encode(msg)
			if err != nil {
				t.Fatalf("encode error = %v", err)
			}
			if len(data) == 0 || string(data) == "{}" {
				t.Fatalf("mock data is empty: %s", data)
			}
		})
	}
}

func TestRegistryRelaysModbusRead(t *testing.T) {
	modbusClient := &fakeBridgeModbus{readRes: &bridgemodbus.ReadHoldingRegistersRes{
		UintValues: []uint32{100},
		IntValues:  []int32{100},
	}}
	registry := NewRegistry(&fakePodEngine{}, modbusClient)

	msg, err := registry.Invoke(context.Background(), MethodModbusReadHoldingRegisters, []byte(`{"address":0,"quantity":1}`))
	if err != nil {
		t.Fatalf("invoke error = %v", err)
	}
	res := msg.(*bridgemodbus.ReadHoldingRegistersRes)
	if len(res.UintValues) != 1 || res.UintValues[0] != 100 {
		t.Fatalf("unexpected response: %v", res)
	}
}

func TestRegistryRelaysModbusWrite(t *testing.T) {
	modbusClient := &fakeBridgeModbus{writeRes: &bridgemodbus.WriteSingleCoilRes{Results: []byte{0xFF, 0x00}}}
	registry := NewRegistry(&fakePodEngine{}, modbusClient)

	msg, err := registry.Invoke(context.Background(), MethodModbusWriteSingleCoil, []byte(`{"address":0,"value":true}`))
	if err != nil {
		t.Fatalf("invoke error = %v", err)
	}
	res := msg.(*bridgemodbus.WriteSingleCoilRes)
	if string(res.Results) != string([]byte{0xFF, 0x00}) {
		t.Fatalf("unexpected response: %v", res)
	}
}
