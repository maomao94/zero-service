package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"zero-service/aiapp/mcpserver/internal/svc"
	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/common/mcpx"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReadHoldingRegistersArgs 读保持寄存器参数
type ReadHoldingRegistersArgs struct {
	ModbusCode string `json:"modbusCode,omitempty" jsonschema:"Modbus配置编码,空则使用默认配置"`
	Address    uint32 `json:"address" jsonschema:"起始寄存器地址"`
	Quantity   uint32 `json:"quantity" jsonschema:"读取数量(1-125)"`
}

// ReadCoilsArgs 读线圈参数
type ReadCoilsArgs struct {
	ModbusCode string `json:"modbusCode,omitempty" jsonschema:"Modbus配置编码,空则使用默认配置"`
	Address    uint32 `json:"address" jsonschema:"起始线圈地址"`
	Quantity   uint32 `json:"quantity" jsonschema:"读取数量(1-2000)"`
}

// RegisterModbus 注册 Modbus 读操作工具
func RegisterModbus(server *sdkmcp.Server, svcCtx *svc.ServiceContext) {
	// 读保持寄存器
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "read_holding_registers",
		Description: "读取 Modbus 保持寄存器 (Function Code 0x03)，返回寄存器值的多种表示形式（无符号整数、有符号整数、十六进制）",
	}, mcpx.CallToolWrapper(func(ctx context.Context, req *sdkmcp.CallToolRequest, args ReadHoldingRegistersArgs) (*sdkmcp.CallToolResult, any, error) {
		// 注意：同步工具不需要手动发送进度
		// wrapper 会自动发送开始(0%)和结束(100%)进度

		resp, err := svcCtx.BridgeModbusCli.ReadHoldingRegisters(ctx, &bridgemodbus.ReadHoldingRegistersReq{
			ModbusCode: args.ModbusCode,
			Address:    args.Address,
			Quantity:   args.Quantity,
		})
		if err != nil {
			return nil, nil, err
		}

		text := formatRegistersResult(args.Address, resp.UintValues, resp.IntValues, resp.HexValues)
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: text}},
		}, nil, nil
	}, mcpx.WithExtractUserCtx()))

	// 读线圈
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "read_coils",
		Description: "读取 Modbus 线圈状态 (Function Code 0x01)，返回每个线圈的开关状态（true=ON, false=OFF）",
	}, mcpx.CallToolWrapper(func(ctx context.Context, req *sdkmcp.CallToolRequest, args ReadCoilsArgs) (*sdkmcp.CallToolResult, any, error) {
		// 注意：同步工具不需要手动发送进度
		// wrapper 会自动发送开始(0%)和结束(100%)进度

		resp, err := svcCtx.BridgeModbusCli.ReadCoils(ctx, &bridgemodbus.ReadCoilsReq{
			ModbusCode: args.ModbusCode,
			Address:    args.Address,
			Quantity:   args.Quantity,
		})
		if err != nil {
			return nil, nil, err
		}

		text := formatCoilsResult(args.Address, resp.Values)
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: text}},
		}, nil, nil
	}, mcpx.WithExtractUserCtx()))
}

type registerEntry struct {
	Address   uint32 `json:"address"`
	UintValue uint32 `json:"uint_value"`
	IntValue  int32  `json:"int_value"`
	HexValue  string `json:"hex_value"`
}

// formatRegistersResult 格式化寄存器读取结果为 JSON 文本
func formatRegistersResult(startAddr uint32, uintValues []uint32, intValues []int32, hexValues []string) string {
	entries := make([]registerEntry, len(uintValues))
	for i := range uintValues {
		entries[i] = registerEntry{
			Address:   startAddr + uint32(i),
			UintValue: uintValues[i],
		}
		if i < len(intValues) {
			entries[i].IntValue = intValues[i]
		}
		if i < len(hexValues) {
			entries[i].HexValue = hexValues[i]
		}
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Sprintf("格式化失败: %v", err)
	}
	return string(data)
}

type coilEntry struct {
	Address uint32 `json:"address"`
	Value   bool   `json:"value"`
	Status  string `json:"status"`
}

// formatCoilsResult 格式化线圈读取结果
func formatCoilsResult(startAddr uint32, values []bool) string {
	entries := make([]coilEntry, len(values))
	for i, v := range values {
		status := "OFF"
		if v {
			status = "ON"
		}
		entries[i] = coilEntry{
			Address: startAddr + uint32(i),
			Value:   v,
			Status:  status,
		}
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Sprintf("格式化失败: %v", err)
	}
	return string(data)
}
