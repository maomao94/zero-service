package modbusx

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grid-x/modbus"
)

func TestNewModbusClientValidatesSlave(t *testing.T) {
	// 超出范围必须报错，不能静默截断成错误从站号。
	if _, err := NewModbusClient(&ModbusClientConf{Address: "127.0.0.1:1", Slave: 248}); err == nil {
		t.Fatal("slave=248 error = nil, want error")
	}

	// 0 表示未配置，按缺省从站 1 处理。
	cli, err := NewModbusClient(&ModbusClientConf{Address: "127.0.0.1:1", Slave: 0})
	if err != nil {
		t.Fatalf("slave=0 error = %v", err)
	}
	if err := cli.Close(); err != nil {
		t.Fatalf("close default client: %v", err)
	}
}

func TestClientConcurrentReuseAndSDKDisconnectReconnect(t *testing.T) {
	var connections atomic.Int32
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			connections.Add(1)
			go func() {
				defer conn.Close()
				for {
					req, err := readModbusFrame(conn)
					if err != nil {
						return
					}
					// 将请求地址回显为寄存器值，检测并发响应错配。
					if _, err := conn.Write(holdingRegistersResponse(req, binary.BigEndian.Uint16(req[8:10]))); err != nil {
						return
					}
				}
			}()
		}
	}()
	// 单连接断言不依赖空闲断链，留足间隔避免慢 CI 上 SDK 提前回收连接。
	cli, err := NewModbusClient(&ModbusClientConf{Address: ln.Addr().String(), IdleTimeout: 5000, Timeout: 1000})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Go(func() {
			result, err := cli.ReadHoldingRegisters(context.Background(), uint16(i), 1)
			if err != nil || len(result) != 2 || binary.BigEndian.Uint16(result) != uint16(i) {
				t.Errorf("address=%d result=%x err=%v", i, result, err)
			}
		})
	}
	wg.Wait()
	if got := connections.Load(); got != 1 {
		t.Fatalf("connections=%d, want 1", got)
	}
	// SDK 空闲回收也走此关闭路径；直接关闭底层连接验证封装对象可复用。
	if err := cli.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := cli.ReadHoldingRegisters(context.Background(), 42, 1); err != nil {
		t.Fatal(err)
	}
	if got := connections.Load(); got != 2 {
		t.Fatalf("reconnect count=%d, want 2", got)
	}
	if err := cli.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := cli.ReadHoldingRegisters(context.Background(), 0, 1); err != nil {
		t.Fatal(err)
	}
	if got := connections.Load(); got != 3 {
		t.Fatalf("Close should allow SDK reconnect: %d", got)
	}
}

func TestClientIdleTimeoutClosesSocketWithoutClosingClient(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	disconnected := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		req, err := readModbusFrame(conn)
		if err != nil {
			return
		}
		if _, err := conn.Write(holdingRegistersResponse(req, 1)); err != nil {
			return
		}
		var b [1]byte
		if _, err := conn.Read(b[:]); err == io.EOF {
			close(disconnected)
		}
	}()
	cli, err := NewModbusClient(&ModbusClientConf{Address: ln.Addr().String(), IdleTimeout: 20, Timeout: 1000})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	if _, err := cli.ReadHoldingRegisters(context.Background(), 0, 1); err != nil {
		t.Fatal(err)
	}
	select {
	case <-disconnected:
	case <-time.After(time.Second):
		t.Fatal("SDK did not close idle TCP connection")
	}
}

// startFakeModbusServer 启动一个最小 Modbus TCP 从站，按 handler 返回响应帧。
func startFakeModbusServer(t *testing.T, handler func(req []byte) []byte) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				for {
					req, err := readModbusFrame(conn)
					if err != nil {
						return
					}
					resp := handler(req)
					if resp == nil {
						return
					}
					if _, err := conn.Write(resp); err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	return ln.Addr().String()
}

func readModbusFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 7)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	length := int(binary.BigEndian.Uint16(header[4:6]))
	body := make([]byte, length-1)
	if _, err := io.ReadFull(conn, body); err != nil {
		return nil, err
	}
	return append(header, body...), nil
}

func exceptionResponse(req []byte, exceptionCode byte) []byte {
	resp := make([]byte, 9)
	copy(resp[0:2], req[0:2]) // 复用事务 ID
	binary.BigEndian.PutUint16(resp[2:4], 0)
	binary.BigEndian.PutUint16(resp[4:6], 3)
	resp[6] = req[6]
	resp[7] = req[7] | 0x80
	resp[8] = exceptionCode
	return resp
}

func holdingRegistersResponse(req []byte, value uint16) []byte {
	resp := make([]byte, 11)
	copy(resp[0:2], req[0:2])
	binary.BigEndian.PutUint16(resp[2:4], 0)
	binary.BigEndian.PutUint16(resp[4:6], 5) // unit + fc + byteCount + data
	resp[6] = req[6]
	resp[7] = req[7]
	resp[8] = 2
	binary.BigEndian.PutUint16(resp[9:11], value)
	return resp
}

func TestClientReadHoldingRegisters(t *testing.T) {
	address := startFakeModbusServer(t, func(req []byte) []byte {
		return holdingRegistersResponse(req, 42)
	})

	cli, err := NewModbusClient(&ModbusClientConf{Address: address, Timeout: 1000})
	if err != nil {
		t.Fatalf("NewModbusClient() error = %v", err)
	}
	defer cli.Close()

	results, err := cli.ReadHoldingRegisters(context.Background(), 0, 1)
	if err != nil {
		t.Fatalf("ReadHoldingRegisters() error = %v", err)
	}
	if len(results) != 2 || binary.BigEndian.Uint16(results) != 42 {
		t.Fatalf("results = % x, want 00 2a", results)
	}
}

func TestClientReadHoldingRegistersException(t *testing.T) {
	address := startFakeModbusServer(t, func(req []byte) []byte {
		return exceptionResponse(req, modbus.ExceptionCodeIllegalDataAddress)
	})

	cli, err := NewModbusClient(&ModbusClientConf{Address: address, Timeout: 1000})
	if err != nil {
		t.Fatalf("NewModbusClient() error = %v", err)
	}
	defer cli.Close()

	_, err = cli.ReadHoldingRegisters(context.Background(), 0, 1)
	if err == nil {
		t.Fatalf("ReadHoldingRegisters() error = nil, want modbus exception")
	}

	functionCode, exceptionCode, ok := Exception(err)
	if !ok {
		t.Fatalf("Exception() ok = false for %v", err)
	}
	if functionCode&0x7F != modbus.FuncCodeReadHoldingRegisters {
		t.Fatalf("functionCode = %#x, want %#x", functionCode, modbus.FuncCodeReadHoldingRegisters)
	}
	if exceptionCode != modbus.ExceptionCodeIllegalDataAddress {
		t.Fatalf("exceptionCode = %#x, want %#x", exceptionCode, modbus.ExceptionCodeIllegalDataAddress)
	}
}

func TestClientTimeout(t *testing.T) {
	// 服务端接受连接但从不响应，验证配置的 Timeout 生效。
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	cli, err := NewModbusClient(&ModbusClientConf{Address: ln.Addr().String(), Timeout: 100})
	if err != nil {
		t.Fatalf("NewModbusClient() error = %v", err)
	}
	defer cli.Close()

	start := time.Now()
	_, err = cli.ReadHoldingRegisters(context.Background(), 0, 1)
	if err == nil {
		t.Fatalf("ReadHoldingRegisters() error = nil, want timeout")
	}
	if !IsTimeout(err) {
		t.Fatalf("IsTimeout(%v) = false, want true", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("read waited %s, want about 100ms", elapsed)
	}
}
