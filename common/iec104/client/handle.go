package client

import (
	"context"
	"zero-service/common/iec104"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/timex"
)

const (
	SinglePoint                            DataType = iota // 单点信息
	DoublePoint                                            // 双点信息
	MeasuredValueScaled                                    // 测量值，标度化值信息
	MeasuredValueNormal                                    // 测量值,规一化值信息
	StepPosition                                           // 步位置信息
	BitString32                                            // 比特位串信息
	MeasuredValueFloat                                     // 测量值,短浮点数信息
	IntegratedTotals                                       // 累计量信息
	EventOfProtectionEquipment                             // 继电器保护设备事件信息
	PackedStartEventsOfProtectionEquipment                 // 继电器保护设备事件信息
	PackedOutputCircuitInfo                                // 继电器保护设备成组输出电路信息
	PackedSinglePointWithSCD                               // 带变位检出的成组单点信息
	SetSingleCommand
	SetDoubleCommand
	SetStepCommand
	SetSetpointNormalized
	SetSetpointScaled
	SetSetpointFloat
	SetBitstringCommand
	EndOfInitialization // 初始化结束 (M_EI_NA_1)
	UNKNOWN             // 未知的
)

type DataType int

type ClientHandler struct {
	call      ASDUCall
	metrics   *stat.Metrics
	traceOpts iec104.FrameTraceOptions
}

// InterrogationHandler 总召唤回复
func (h *ClientHandler) InterrogationHandler(_ asdu.Connect, rxAsdu *asdu.ASDU) error {
	startTime := timex.Now()
	defer h.metrics.Add(stat.Task{
		Duration: timex.Since(startTime),
	})
	ctx, span := iec104.StartRecvSpan(context.Background(), rxAsdu, h.traceOpts)
	defer span.End()
	ctx = IecLogContext(ctx, rxAsdu, h.traceOpts)
	return h.call.OnInterrogation(ctx, rxAsdu)
}

// CounterInterrogationHandler 总计数器回复
func (h *ClientHandler) CounterInterrogationHandler(_ asdu.Connect, rxAsdu *asdu.ASDU) error {
	startTime := timex.Now()
	defer h.metrics.Add(stat.Task{
		Duration: timex.Since(startTime),
	})
	ctx, span := iec104.StartRecvSpan(context.Background(), rxAsdu, h.traceOpts)
	defer span.End()
	ctx = IecLogContext(ctx, rxAsdu, h.traceOpts)
	return h.call.OnCounterInterrogation(ctx, rxAsdu)
}

// ReadHandler 读定值回复
func (h *ClientHandler) ReadHandler(_ asdu.Connect, rxAsdu *asdu.ASDU) error {
	startTime := timex.Now()
	defer h.metrics.Add(stat.Task{
		Duration: timex.Since(startTime),
	})
	ctx, span := iec104.StartRecvSpan(context.Background(), rxAsdu, h.traceOpts)
	defer span.End()
	ctx = IecLogContext(ctx, rxAsdu, h.traceOpts)
	return h.call.OnRead(ctx, rxAsdu)
}

// TestCommandHandler 测试下发回复
func (h *ClientHandler) TestCommandHandler(_ asdu.Connect, rxAsdu *asdu.ASDU) error {
	startTime := timex.Now()
	defer h.metrics.Add(stat.Task{
		Duration: timex.Since(startTime),
	})
	ctx, span := iec104.StartRecvSpan(context.Background(), rxAsdu, h.traceOpts)
	defer span.End()
	ctx = IecLogContext(ctx, rxAsdu, h.traceOpts)
	return h.call.OnTestCommand(ctx, rxAsdu)
}

// ClockSyncHandler 时钟同步回复
func (h *ClientHandler) ClockSyncHandler(_ asdu.Connect, rxAsdu *asdu.ASDU) error {
	startTime := timex.Now()
	defer h.metrics.Add(stat.Task{
		Duration: timex.Since(startTime),
	})
	ctx, span := iec104.StartRecvSpan(context.Background(), rxAsdu, h.traceOpts)
	defer span.End()
	ctx = IecLogContext(ctx, rxAsdu, h.traceOpts)
	return h.call.OnClockSync(ctx, rxAsdu)
}

// ResetProcessHandler 进程重置回复
func (h *ClientHandler) ResetProcessHandler(_ asdu.Connect, rxAsdu *asdu.ASDU) error {
	startTime := timex.Now()
	defer h.metrics.Add(stat.Task{
		Duration: timex.Since(startTime),
	})
	ctx, span := iec104.StartRecvSpan(context.Background(), rxAsdu, h.traceOpts)
	defer span.End()
	ctx = IecLogContext(ctx, rxAsdu, h.traceOpts)
	return h.call.OnResetProcess(ctx, rxAsdu)
}

// DelayAcquisitionHandler 延迟获取回复
func (h *ClientHandler) DelayAcquisitionHandler(_ asdu.Connect, rxAsdu *asdu.ASDU) error {
	startTime := timex.Now()
	defer h.metrics.Add(stat.Task{
		Duration: timex.Since(startTime),
	})
	ctx, span := iec104.StartRecvSpan(context.Background(), rxAsdu, h.traceOpts)
	defer span.End()
	ctx = IecLogContext(ctx, rxAsdu, h.traceOpts)
	return h.call.OnDelayAcquisition(ctx, rxAsdu)
}

// ASDUHandler ASDU上报，ASDU数据
func (h *ClientHandler) ASDUHandler(_ asdu.Connect, rxAsdu *asdu.ASDU) error {
	startTime := timex.Now()
	defer h.metrics.Add(stat.Task{
		Duration: timex.Since(startTime),
	})
	ctx, span := iec104.StartRecvSpan(context.Background(), rxAsdu, h.traceOpts)
	defer span.End()
	ctx = IecLogContext(ctx, rxAsdu, h.traceOpts)
	return h.call.OnASDU(ctx, rxAsdu)
}

func GetDataType(typeId asdu.TypeID) DataType {
	switch typeId {
	case asdu.M_SP_NA_1, asdu.M_SP_TA_1, asdu.M_SP_TB_1:
		return SinglePoint
	case asdu.M_DP_NA_1, asdu.M_DP_TA_1, asdu.M_DP_TB_1:
		return DoublePoint
	case asdu.M_ST_NA_1, asdu.M_ST_TA_1, asdu.M_ST_TB_1:
		return StepPosition
	case asdu.M_BO_NA_1, asdu.M_BO_TA_1, asdu.M_BO_TB_1:
		return BitString32
	case asdu.M_ME_NB_1, asdu.M_ME_TB_1, asdu.M_ME_TE_1:
		return MeasuredValueScaled
	case asdu.M_ME_NA_1, asdu.M_ME_TA_1, asdu.M_ME_TD_1, asdu.M_ME_ND_1:
		return MeasuredValueNormal
	case asdu.M_ME_NC_1, asdu.M_ME_TC_1, asdu.M_ME_TF_1:
		return MeasuredValueFloat
	case asdu.M_IT_NA_1, asdu.M_IT_TA_1, asdu.M_IT_TB_1:
		return IntegratedTotals
	case asdu.M_EP_TA_1, asdu.M_EP_TD_1:
		return EventOfProtectionEquipment
	case asdu.M_EP_TB_1, asdu.M_EP_TE_1:
		return PackedStartEventsOfProtectionEquipment
	case asdu.M_EP_TC_1, asdu.M_EP_TF_1:
		return PackedOutputCircuitInfo
	case asdu.M_PS_NA_1:
		return PackedSinglePointWithSCD
	case asdu.M_EI_NA_1:
		return EndOfInitialization
	case asdu.C_SC_NA_1, asdu.C_SC_TA_1:
		return SetSingleCommand
	case asdu.C_DC_NA_1, asdu.C_DC_TA_1:
		return SetDoubleCommand
	case asdu.C_RC_NA_1, asdu.C_RC_TA_1:
		return SetStepCommand
	case asdu.C_SE_NA_1, asdu.C_SE_TA_1:
		return SetSetpointNormalized
	case asdu.C_SE_NB_1, asdu.C_SE_TB_1:
		return SetSetpointScaled
	case asdu.C_SE_NC_1, asdu.C_SE_TC_1:
		return SetSetpointFloat
	case asdu.C_BO_NA_1, asdu.C_BO_TA_1:
		return SetBitstringCommand
	default:
		return UNKNOWN
	}
}

func IecLogContext(ctx context.Context, packet *asdu.ASDU, traceOpts iec104.FrameTraceOptions) context.Context {
	ctx = context.WithValue(ctx, "stationId", traceOpts.StationId)
	return logx.ContextWithFields(ctx,
		logx.Field("host", traceOpts.Host),
		logx.Field("port", traceOpts.Port),
		logx.Field("stationId", traceOpts.StationId),
		logx.Field("iecType", GenTypeName(packet.Type)),
		logx.Field("typeId", int(packet.Type)),
		logx.Field("coa", uint(packet.CommonAddr)),
		logx.Field("cot", GenCOTName(packet.Coa.Cause)),
		logx.Field("cotCause", int(packet.Coa.Cause)),
		logx.Field("isNegative", packet.Coa.IsNegative),
	)
}

func GenTypeName(typeId asdu.TypeID) string {
	return strutil.SubInBetween(typeId.String(), "<", ">")
}

func GenCOTName(cause asdu.Cause) string {
	switch cause {
	case asdu.ActivationCon:
		return "ActivationCon"
	case asdu.DeactivationCon:
		return "DeactivationCon"
	case asdu.ActivationTerm:
		return "ActivationTerm"
	case asdu.Activation:
		return "Activation(echo)"
	case asdu.Request:
		return "Request"
	case asdu.UnknownTypeID:
		return "UnknownTypeID"
	case asdu.UnknownCOT:
		return "UnknownCOT"
	case asdu.UnknownCA:
		return "UnknownCA"
	case asdu.UnknownIOA:
		return "UnknownIOA"
	default:
		return "Unknown"
	}
}
