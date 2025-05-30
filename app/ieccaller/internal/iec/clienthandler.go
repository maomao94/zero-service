package iec

import (
	"context"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/golang-module/carbon/v2"
	"github.com/jinzhu/copier"
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/ieccaller/internal/svc"
	iec104client "zero-service/iec104/iec104client"
	"zero-service/iec104/types"
)

type ClientCall struct {
	svcCtx *svc.ServiceContext
	host   string
	port   int
	logger logx.Logger
}

func NewClientCall(svcCtx *svc.ServiceContext, host string, port int) *ClientCall {
	ctx := logx.ContextWithFields(context.Background(),
		logx.Field("host", host),
		logx.Field("port", port),
	)
	return &ClientCall{
		svcCtx: svcCtx,
		host:   host,
		port:   port,
		logger: logx.WithContext(ctx),
	}
}

// OnInterrogation 总召唤回复
func (c *ClientCall) OnInterrogation(packet *asdu.ASDU) error {
	addr, value := packet.GetInterrogationCmd()
	c.logger.Infof("interrogation reply, addr: %d, value: %d", addr, value)
	return nil
}

// OnCounterInterrogation 总计数器回复
func (c *ClientCall) OnCounterInterrogation(packet *asdu.ASDU) error {
	addr, value := packet.GetCounterInterrogationCmd()
	c.logger.Infof("counter interrogation reply, addr: %d, request: 0x%02X, rreeze: 0x%02X",
		addr, value.Request, value.Freeze)
	return nil
}

// OnRead 读定值回复
func (c *ClientCall) OnRead(packet *asdu.ASDU) error {
	return c.OnASDU(packet)
}

// OnTestCommand 测试下发回复
func (c *ClientCall) OnTestCommand(packet *asdu.ASDU) error {
	addr, value := packet.GetTestCommand()
	c.logger.Infof("test cmd reply, addr: %d, value: %t", addr, value)
	return nil
}

// OnClockSync 时钟同步回复
func (c *ClientCall) OnClockSync(packet *asdu.ASDU) error {
	addr, value := packet.GetClockSynchronizationCmd()
	c.logger.Infof("clock sync reply, addr: %d, value: %d", addr, value.UnixMilli())
	return nil
}

// OnResetProcess 进程重置回复
func (c *ClientCall) OnResetProcess(packet *asdu.ASDU) error {
	addr, value := packet.GetResetProcessCmd()
	c.logger.Infof("reset process reply, addr: %d, value: 0x%02X", addr, value)
	return nil
}

// OnDelayAcquisition 延迟获取回复
func (c *ClientCall) OnDelayAcquisition(packet *asdu.ASDU) error {
	addr, value := packet.GetDelayAcquireCommand()
	c.logger.Infof("delay acquisition reply, addr: %d, value: %d", addr, value)
	return nil
}

// OnASDU 数据正体
func (c *ClientCall) OnASDU(packet *asdu.ASDU) error {
	// 读取设备数据
	switch iec104client.GetDataType(packet.Type) {
	case iec104client.SinglePoint:
		c.onSinglePoint(packet)
	case iec104client.DoublePoint:
		c.onDoublePoint(packet)
	case iec104client.MeasuredValueScaled:
		c.onMeasuredValueScaled(packet)
	case iec104client.MeasuredValueNormal:
		c.onMeasuredValueNormal(packet)
	case iec104client.StepPosition:
		c.onStepPosition(packet)
	case iec104client.BitString32:
		c.onBitString32(packet)
	case iec104client.MeasuredValueFloat:
		c.onMeasuredValueFloat(packet)
	case iec104client.IntegratedTotals:
		c.onIntegratedTotals(packet)
	case iec104client.EventOfProtectionEquipment:
		c.onEventOfProtectionEquipment(packet)
	case iec104client.PackedStartEventsOfProtectionEquipment:
		c.onPackedStartEventsOfProtectionEquipment(packet)
	case iec104client.PackedOutputCircuitInfo:
		c.onPackedOutputCircuitInfo(packet)
	case iec104client.PackedSinglePointWithSCD:
		c.onPackedSinglePointWithSCD(packet)
	default:
		return nil
	}

	return nil
}

func (c *ClientCall) onSinglePoint(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_SP_NA_1], [M_SP_TA_1] or [M_SP_TB_1] 获取单点信息信息体集合
	for _, p := range packet.GetSinglePoint() {
		c.logger.Infof("single point, ioa: %d, value: %v", p.Ioa, p.Value)
		var obj types.SinglePointInfo
		obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, types.Option)
		_ = c.svcCtx.PushASDU(&types.MsgBody{
			Host:   c.host,
			Port:   c.port,
			Asdu:   genASDUName(packet.Type),
			TypeId: int(packet.Type),
			Coa:    uint(coa),
			Body:   &obj,
		})
	}
}

func (c *ClientCall) onDoublePoint(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_DP_NA_1], [M_DP_TA_1] or [M_DP_TB_1] 获得双点信息体集合
	for _, p := range packet.GetDoublePoint() {
		c.logger.Infof("double point, ioa: %d, value: %v", p.Ioa, p.Value)
		var obj types.DoublePointInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, types.Option)
		_ = c.svcCtx.PushASDU(&types.MsgBody{
			Host:   c.host,
			Port:   c.port,
			Asdu:   genASDUName(packet.Type),
			TypeId: int(packet.Type),
			Coa:    uint(coa),
			Body:   &obj,
		})
	}
}

func (c *ClientCall) onMeasuredValueScaled(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_ME_NB_1], [M_ME_TB_1] or [M_ME_TE_1] 获得测量值,标度化值信息体集合
	for _, p := range packet.GetMeasuredValueScaled() {
		c.logger.Infof("measured value scaled, ioa: %d, value: %v", p.Ioa, p.Value)
		var obj types.MeasuredValueScaledInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, types.Option)
		_ = c.svcCtx.PushASDU(&types.MsgBody{
			Host:   c.host,
			Port:   c.port,
			Asdu:   genASDUName(packet.Type),
			TypeId: int(packet.Type),
			Coa:    uint(coa),
			Body:   &obj,
		})
	}
}

func (c *ClientCall) onMeasuredValueNormal(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_ME_NA_1], [M_ME_TA_1],[ M_ME_TD_1] or [M_ME_ND_1] 获得测量值,规一化值信息体集合
	for _, p := range packet.GetMeasuredValueNormal() {
		c.logger.Infof("measured value normal, ioa: %d, value: %v", p.Ioa, p.Value)
		var obj types.MeasuredValueNormalInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, types.Option)
		_ = c.svcCtx.PushASDU(&types.MsgBody{
			Host:   c.host,
			Port:   c.port,
			Asdu:   genASDUName(packet.Type),
			TypeId: int(packet.Type),
			Coa:    uint(coa),
			Body:   &obj,
		})
	}
}

func (c *ClientCall) onStepPosition(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_ST_NA_1], [M_ST_TA_1] or [M_ST_TB_1] 获得步位置信息体集合
	for _, p := range packet.GetStepPosition() {
		// state：false: 设备未在瞬变状态 true： 设备处于瞬变状态
		c.logger.Infof("step position, ioa: %d, state: %t, value: %d", p.Ioa, p.Value.HasTransient, p.Value.Val)
		var obj types.StepPositionInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, types.Option)
		_ = c.svcCtx.PushASDU(&types.MsgBody{
			Host:   c.host,
			Port:   c.port,
			Asdu:   genASDUName(packet.Type),
			TypeId: int(packet.Type),
			Coa:    uint(coa),
			Body:   &obj,
		})
	}
}

func (c *ClientCall) onBitString32(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_BO_NA_1], [M_BO_TA_1] or [M_BO_TB_1] 获得比特位串信息体集合
	for _, p := range packet.GetBitString32() {
		c.logger.Infof("bigtstring32, ioa: %d, value: %v, bsi: %032b", p.Ioa, p.Value, p.Value)
		var obj types.BitString32Info
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, types.Option)
		_ = c.svcCtx.PushASDU(&types.MsgBody{
			Host:   c.host,
			Port:   c.port,
			Asdu:   genASDUName(packet.Type),
			TypeId: int(packet.Type),
			Coa:    uint(coa),
			Body:   &obj,
		})
	}
}

func (c *ClientCall) onMeasuredValueFloat(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_ME_NC_1], [M_ME_TC_1] or [M_ME_TF_1].获得测量值,短浮点数信息体集合
	for _, p := range packet.GetMeasuredValueFloat() {
		c.logger.Infof("measured value float, ioa: %d, value: %v", p.Ioa, p.Value)
		var obj types.MeasuredValueFloatInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, types.Option)
		_ = c.svcCtx.PushASDU(&types.MsgBody{
			Host:   c.host,
			Port:   c.port,
			Asdu:   genASDUName(packet.Type),
			TypeId: int(packet.Type),
			Coa:    uint(coa),
			Body:   &obj,
		})
	}
}

func (c *ClientCall) onIntegratedTotals(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_IT_NA_1], [M_IT_TA_1] or [M_IT_TB_1]. 获得累计量信息体集合
	for _, p := range packet.GetIntegratedTotals() {
		c.logger.Infof("integrated totals, ioa: %d, count: %d, SQ: 0x%02X, CY: %t, CA: %t, IV: %t",
			p.Ioa, p.Value.CounterReading, p.Value.SeqNumber, p.Value.HasCarry, p.Value.IsAdjusted, p.Value.IsInvalid)
		var obj types.BinaryCounterReadingInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, types.Option)
		_ = c.svcCtx.PushASDU(&types.MsgBody{
			Host:   c.host,
			Port:   c.port,
			Asdu:   genASDUName(packet.Type),
			TypeId: int(packet.Type),
			Coa:    uint(coa),
			Body:   &obj,
		})
	}
}

func (c *ClientCall) onEventOfProtectionEquipment(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_EP_TA_1] [M_EP_TD_1] 获取继电器保护设备事件信息体
	for _, p := range packet.GetEventOfProtectionEquipment() {
		c.logger.Infof("event of protection equipment, ioa: %d, event: %d, qdp: %d, mesc: %d, time: %d",
			p.Ioa, p.Event, p.Qdp, p.Msec, p.Time.UnixMilli())
		var obj types.EventOfProtectionEquipmentInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, types.Option)
		_ = c.svcCtx.PushASDU(&types.MsgBody{
			Host:   c.host,
			Port:   c.port,
			Asdu:   genASDUName(packet.Type),
			TypeId: int(packet.Type),
			Coa:    uint(coa),
			Body:   &obj,
		})
	}
}

func (c *ClientCall) onPackedStartEventsOfProtectionEquipment(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_EP_TB_1] [M_EP_TE_1] 获取继电器保护设备事件信息体
	p := packet.GetPackedStartEventsOfProtectionEquipment()
	c.logger.Infof("packed start events of protection equipment, ioa: %d, event: %d, qdp: %d, mesc: %d, time: %d",
		p.Ioa, p.Event, p.Qdp, p.Msec, p.Time.UnixMilli())
	var obj types.PackedStartEventsOfProtectionEquipmentInfo
	//obj.Time = carbon.Now().ToDateTimeString()
	copier.CopyWithOption(&obj, &p, types.Option)
	_ = c.svcCtx.PushASDU(&types.MsgBody{
		Host:   c.host,
		Port:   c.port,
		Asdu:   genASDUName(packet.Type),
		TypeId: int(packet.Type),
		Coa:    uint(coa),
		Body:   &obj,
	})
}

func (c *ClientCall) onPackedOutputCircuitInfo(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_EP_TC_1] [M_EP_TF_1] 获取继电器保护设备成组输出电路信息信息体
	p := packet.GetPackedOutputCircuitInfo()
	gc := (p.Oci & asdu.OCIGeneralCommand) != 0
	cl1 := (p.Oci & asdu.OCICommandL1) != 0
	cl2 := (p.Oci & asdu.OCICommandL2) != 0
	cl3 := (p.Oci & asdu.OCICommandL3) != 0
	c.logger.Infof("packed Output circuit, ioa: %d, qci: %d, gc: %v, cl1: %v, cl2: %v, cl3: %v, qdp: %d, mesc: %d, time: %d",
		p.Ioa, p.Oci, gc, cl1, cl2, cl3, p.Qdp, p.Msec, p.Time.UnixMilli())
	var obj types.PackedOutputCircuitInfoInfo
	//obj.Time = carbon.Now().ToDateTimeString()
	copier.CopyWithOption(&obj, &p, types.Option)
	_ = c.svcCtx.PushASDU(&types.MsgBody{
		Host:   c.host,
		Port:   c.port,
		Asdu:   genASDUName(packet.Type),
		TypeId: int(packet.Type),
		Coa:    uint(coa),
		Body:   &obj,
	})
}

func (c *ClientCall) onPackedSinglePointWithSCD(packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_PS_NA_1]. 获得带变位检出的成组单点信息
	for _, p := range packet.GetPackedSinglePointWithSCD() {
		c.logger.Infof("packed single point with SCD, ioa: %d, scd: %d, qds: %d", p.Ioa, p.Scd, p.Qds)
		var obj types.PackedSinglePointWithSCDInfo
		currentStatus := p.Scd & 0xFFFF        // 低16位（当前状态）
		statusChange := (p.Scd >> 16) & 0xFFFF // 高16位（状态变化）
		var activePoints []int
		var changedPoints []int
		c.logger.Infof("stn: %d, %016b, cdn: %d, %016b", currentStatus, currentStatus, statusChange, statusChange)
		for i := 0; i < 16; i++ {
			if currentStatus&(1<<i) != 0 {
				activePoints = append(activePoints, i)
			}
			if statusChange&(1<<i) != 0 {
				changedPoints = append(changedPoints, i)
			}
		}

		c.logger.Infof("当前闭合的位: %v", activePoints)
		c.logger.Infof("状态变化的位: %v", changedPoints)
		copier.CopyWithOption(&obj, &p, types.Option)
		_ = c.svcCtx.PushASDU(&types.MsgBody{
			Host:   c.host,
			Port:   c.port,
			Asdu:   genASDUName(packet.Type),
			TypeId: int(packet.Type),
			Coa:    uint(coa),
			Body:   &obj,
		})
	}
}

func genASDUName(typeId asdu.TypeID) string {
	return strutil.SubInBetween(typeId.String(), "<", ">")
}
