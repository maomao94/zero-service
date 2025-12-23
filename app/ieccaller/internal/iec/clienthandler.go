package iec

import (
	"context"
	"fmt"
	"zero-service/app/ieccaller/internal/config"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/copierx"
	"zero-service/common/iec104/client"
	"zero-service/common/iec104/types"
	"zero-service/common/iec104/util"
	"zero-service/common/tool"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/jinzhu/copier"
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type ClientCall struct {
	svcCtx     *svc.ServiceContext
	config     config.IecServerConfig
	stationId  string
	taskRunner *threading.TaskRunner
}

var _ client.ASDUCall = (*ClientCall)(nil)

func NewClientCall(svcCtx *svc.ServiceContext, config config.IecServerConfig) *ClientCall {
	// 生成 stationId
	stationId := util.GenerateStationId(config.Host, config.Port)
	if len(config.MetaData) > 0 {
		if sid, ok := config.MetaData["stationId"].(string); ok && sid != "" {
			stationId = sid
		}
	}
	return &ClientCall{
		svcCtx:     svcCtx,
		config:     config,
		taskRunner: threading.NewTaskRunner(config.TaskConcurrency),
		stationId:  stationId,
	}
}

// OnInterrogation 总召唤回复
func (c *ClientCall) OnInterrogation(packet *asdu.ASDU) error {
	addr, value := packet.GetInterrogationCmd()
	logx.Debugf("interrogation reply, addr: %d, value: %d", addr, value)
	return nil
}

// OnCounterInterrogation 总计数器回复
func (c *ClientCall) OnCounterInterrogation(packet *asdu.ASDU) error {
	addr, value := packet.GetCounterInterrogationCmd()
	logx.Debugf("counter interrogation reply, addr: %d, request: 0x%02X, rreeze: 0x%02X",
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
	logx.Debugf("test cmd reply, addr: %d, value: %t", addr, value)
	return nil
}

// OnClockSync 时钟同步回复
func (c *ClientCall) OnClockSync(packet *asdu.ASDU) error {
	addr, value := packet.GetClockSynchronizationCmd()
	logx.Debugf("clock sync reply, addr: %d, value: %d", addr, value.UnixMilli())
	return nil
}

// OnResetProcess 进程重置回复
func (c *ClientCall) OnResetProcess(packet *asdu.ASDU) error {
	addr, value := packet.GetResetProcessCmd()
	logx.Debugf("reset process reply, addr: %d, value: 0x%02X", addr, value)
	return nil
}

// OnDelayAcquisition 延迟获取回复
func (c *ClientCall) OnDelayAcquisition(packet *asdu.ASDU) error {
	addr, value := packet.GetDelayAcquireCommand()
	logx.Debugf("delay acquisition reply, addr: %d, value: %d", addr, value)
	return nil
}

// OnASDU 数据正体
func (c *ClientCall) OnASDU(packet *asdu.ASDU) error {
	ctx := logx.ContextWithFields(context.Background(),
		logx.Field("type", packet.Type),
		logx.Field("coa", packet.Coa.String()),
		logx.Field("commonAddr", packet.CommonAddr),
		logx.Field("asdu", genASDUName(packet.Type)),
		logx.Field("host", c.config.Host),
		logx.Field("port", c.config.Port),
		logx.Field("stationId", c.stationId),
	)
	ctx = context.WithValue(ctx, "stationId", c.stationId)
	logx.WithContext(ctx).Info("received OnASDU")
	c.taskRunner.Schedule(func() {
		dataType := client.GetDataType(packet.Type)
		// 读取设备数据
		switch dataType {
		case client.SinglePoint:
			c.onSinglePoint(ctx, packet)
		case client.DoublePoint:
			c.onDoublePoint(ctx, packet)
		case client.MeasuredValueScaled:
			c.onMeasuredValueScaled(ctx, packet)
		case client.MeasuredValueNormal:
			c.onMeasuredValueNormal(ctx, packet)
		case client.StepPosition:
			c.onStepPosition(ctx, packet)
		case client.BitString32:
			c.onBitString32(ctx, packet)
		case client.MeasuredValueFloat:
			c.onMeasuredValueFloat(ctx, packet)
		case client.IntegratedTotals:
			c.onIntegratedTotals(ctx, packet)
		case client.EventOfProtectionEquipment:
			c.onEventOfProtectionEquipment(ctx, packet)
		case client.PackedStartEventsOfProtectionEquipment:
			c.onPackedStartEventsOfProtectionEquipment(ctx, packet)
		case client.PackedOutputCircuitInfo:
			c.onPackedOutputCircuitInfo(ctx, packet)
		case client.PackedSinglePointWithSCD:
			c.onPackedSinglePointWithSCD(ctx, packet)
		default:
			return
		}
	})
	return nil
}

func (c *ClientCall) onSinglePoint(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	asduDataList := packet.GetSinglePoint()
	logx.WithContext(ctx).Debugf("single point, size: %d", len(asduDataList))
	// [M_SP_NA_1], [M_SP_TA_1] or [M_SP_TB_1] 获取单点信息信息体集合
	for _, p := range asduDataList {
		msgId, _ := tool.SimpleUUID()
		logx.WithContext(ctx).Debugf("single point, msgId: %s, ioa: %d, value: %v", msgId, p.Ioa, p.Value)
		var obj types.SinglePointInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, copierx.Option)
		obj.QdsDesc = util.QdsString(p.Qds)
		obj.Ov = util.QdsIsOverflow(p.Qds)
		obj.Bl = util.QdsIsBlocked(p.Qds)
		obj.Sb = util.QdsIsSubstituted(p.Qds)
		obj.Nt = util.QdsIsNotTopical(p.Qds)
		obj.Iv = util.QdsIsInvalid(p.Qds)
		_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
			MsgId:    msgId,
			Host:     c.config.Host,
			Port:     c.config.Port,
			Asdu:     genASDUName(packet.Type),
			TypeId:   int(packet.Type),
			DataType: int(client.GetDataType(packet.Type)),
			Coa:      uint(coa),
			Body:     &obj,
			MetaData: c.config.MetaData,
		}, obj.Ioa)
	}
}

func (c *ClientCall) onDoublePoint(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	asduDataList := packet.GetDoublePoint()
	logx.WithContext(ctx).Debugf("double point, size: %d", len(asduDataList))
	// [M_DP_NA_1], [M_DP_TA_1] or [M_DP_TB_1] 获得双点信息体集合
	for _, p := range asduDataList {
		msgId, _ := tool.SimpleUUID()
		logx.WithContext(ctx).Debugf("double point, msgId: %s, ioa: %d, value: %v, bl: %v, sb: %v, nt: %v, iv:%v", msgId, p.Ioa, p.Value,
			util.QdsIsBlocked(p.Qds), util.QdsIsSubstituted(p.Qds), util.QdsIsNotTopical(p.Qds), util.QdsIsInvalid(p.Qds))
		logx.WithContext(ctx).Debugf("qds: %s", util.QdsString(p.Qds))
		var obj types.DoublePointInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, copierx.Option)
		obj.QdsDesc = util.QdsString(p.Qds)
		obj.Ov = util.QdsIsOverflow(p.Qds)
		obj.Bl = util.QdsIsBlocked(p.Qds)
		obj.Sb = util.QdsIsSubstituted(p.Qds)
		obj.Nt = util.QdsIsNotTopical(p.Qds)
		obj.Iv = util.QdsIsInvalid(p.Qds)
		_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
			MsgId:    msgId,
			Host:     c.config.Host,
			Port:     c.config.Port,
			Asdu:     genASDUName(packet.Type),
			TypeId:   int(packet.Type),
			DataType: int(client.GetDataType(packet.Type)),
			Coa:      uint(coa),
			Body:     &obj,
			MetaData: c.config.MetaData,
		}, obj.Ioa)
	}
}

func (c *ClientCall) onMeasuredValueScaled(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	asduDataList := packet.GetMeasuredValueScaled()
	logx.WithContext(ctx).Debugf("measured value scaled, size: %d", len(asduDataList))
	// [M_ME_NB_1], [M_ME_TB_1] or [M_ME_TE_1] 获得测量值,标度化值信息体集合
	for _, p := range asduDataList {
		msgId, _ := tool.SimpleUUID()
		logx.WithContext(ctx).Debugf("measured value scaled, msgId: %s, ioa: %d, value: %v", msgId, p.Ioa, p.Value)
		var obj types.MeasuredValueScaledInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, copierx.Option)
		obj.QdsDesc = util.QdsString(p.Qds)
		obj.Ov = util.QdsIsOverflow(p.Qds)
		obj.Bl = util.QdsIsBlocked(p.Qds)
		obj.Sb = util.QdsIsSubstituted(p.Qds)
		obj.Nt = util.QdsIsNotTopical(p.Qds)
		obj.Iv = util.QdsIsInvalid(p.Qds)
		_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
			MsgId:    msgId,
			Host:     c.config.Host,
			Port:     c.config.Port,
			Asdu:     genASDUName(packet.Type),
			TypeId:   int(packet.Type),
			DataType: int(client.GetDataType(packet.Type)),
			Coa:      uint(coa),
			Body:     &obj,
			MetaData: c.config.MetaData,
		}, obj.Ioa)
	}
}

func (c *ClientCall) onMeasuredValueNormal(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	asduDataList := packet.GetMeasuredValueNormal()
	logx.WithContext(ctx).Debugf("measured value normal, size: %d", len(asduDataList))
	// [M_ME_NA_1], [M_ME_TA_1],[ M_ME_TD_1] or [M_ME_ND_1] 获得测量值,规一化值信息体集合
	for _, p := range asduDataList {
		msgId, _ := tool.SimpleUUID()
		nva := util.NormalizeToFloat(p.Value)
		logx.WithContext(ctx).Debugf("measured value normal, msgId: %s, ioa: %d, value: %v, nva: %.5f", msgId, p.Ioa, p.Value, nva)
		var obj types.MeasuredValueNormalInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, copierx.Option)
		obj.Nva = nva
		obj.QdsDesc = util.QdsString(p.Qds)
		obj.Ov = util.QdsIsOverflow(p.Qds)
		obj.Bl = util.QdsIsBlocked(p.Qds)
		obj.Sb = util.QdsIsSubstituted(p.Qds)
		obj.Nt = util.QdsIsNotTopical(p.Qds)
		obj.Iv = util.QdsIsInvalid(p.Qds)
		_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
			MsgId:    msgId,
			Host:     c.config.Host,
			Port:     c.config.Port,
			Asdu:     genASDUName(packet.Type),
			TypeId:   int(packet.Type),
			DataType: int(client.GetDataType(packet.Type)),
			Coa:      uint(coa),
			Body:     &obj,
			MetaData: c.config.MetaData,
		}, obj.Ioa)
	}
}

func (c *ClientCall) onStepPosition(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	asduDataList := packet.GetStepPosition()
	logx.WithContext(ctx).Debugf("step position, size: %d", len(asduDataList))
	// [M_ST_NA_1], [M_ST_TA_1] or [M_ST_TB_1] 获得步位置信息体集合
	for _, p := range asduDataList {
		msgId, _ := tool.SimpleUUID()
		// state：false: 设备未在瞬变状态 true： 设备处于瞬变状态
		logx.WithContext(ctx).Debugf("step position, msgId: %s, ioa: %d, state: %t, value: %d", msgId, p.Ioa, p.Value.HasTransient, p.Value.Val)
		var obj types.StepPositionInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, copierx.Option)
		obj.QdsDesc = util.QdsString(p.Qds)
		obj.Ov = util.QdsIsOverflow(p.Qds)
		obj.Bl = util.QdsIsBlocked(p.Qds)
		obj.Sb = util.QdsIsSubstituted(p.Qds)
		obj.Nt = util.QdsIsNotTopical(p.Qds)
		obj.Iv = util.QdsIsInvalid(p.Qds)
		_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
			MsgId:    msgId,
			Host:     c.config.Host,
			Port:     c.config.Port,
			Asdu:     genASDUName(packet.Type),
			TypeId:   int(packet.Type),
			DataType: int(client.GetDataType(packet.Type)),
			Coa:      uint(coa),
			Body:     &obj,
			MetaData: c.config.MetaData,
		}, obj.Ioa)
	}
}

func (c *ClientCall) onBitString32(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	asduDataList := packet.GetBitString32()
	logx.WithContext(ctx).Debugf("bitstring32, size: %d", len(asduDataList))
	// [M_BO_NA_1], [M_BO_TA_1] or [M_BO_TB_1] 获得比特位串信息体集合
	for _, p := range asduDataList {
		msgId, _ := tool.SimpleUUID()
		logx.WithContext(ctx).Debugf("bigtstring32, msgId: %s, ioa: %d, value: %v, bsi: %032b", msgId, p.Ioa, p.Value, p.Value)
		var obj types.BitString32Info
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, copierx.Option)
		obj.QdsDesc = util.QdsString(p.Qds)
		obj.Ov = util.QdsIsOverflow(p.Qds)
		obj.Bl = util.QdsIsBlocked(p.Qds)
		obj.Sb = util.QdsIsSubstituted(p.Qds)
		obj.Nt = util.QdsIsNotTopical(p.Qds)
		obj.Iv = util.QdsIsInvalid(p.Qds)
		_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
			MsgId:    msgId,
			Host:     c.config.Host,
			Port:     c.config.Port,
			Asdu:     genASDUName(packet.Type),
			TypeId:   int(packet.Type),
			DataType: int(client.GetDataType(packet.Type)),
			Coa:      uint(coa),
			Body:     &obj,
			MetaData: c.config.MetaData,
		}, obj.Ioa)
	}
}

func (c *ClientCall) onMeasuredValueFloat(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	asduDataList := packet.GetMeasuredValueFloat()
	logx.WithContext(ctx).Debugf("measured value float, size: %d", len(asduDataList))
	// [M_ME_NC_1], [M_ME_TC_1] or [M_ME_TF_1].获得测量值,短浮点数信息体集合
	for _, p := range asduDataList {
		msgId, _ := tool.SimpleUUID()
		logx.WithContext(ctx).Debugf("measured value float, msgId: %s, ioa: %d, value: %v", msgId, p.Ioa, p.Value)
		var obj types.MeasuredValueFloatInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, copierx.Option)
		obj.QdsDesc = util.QdsString(p.Qds)
		obj.Ov = util.QdsIsOverflow(p.Qds)
		obj.Bl = util.QdsIsBlocked(p.Qds)
		obj.Sb = util.QdsIsSubstituted(p.Qds)
		obj.Nt = util.QdsIsNotTopical(p.Qds)
		obj.Iv = util.QdsIsInvalid(p.Qds)
		_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
			MsgId:    msgId,
			Host:     c.config.Host,
			Port:     c.config.Port,
			Asdu:     genASDUName(packet.Type),
			TypeId:   int(packet.Type),
			DataType: int(client.GetDataType(packet.Type)),
			Coa:      uint(coa),
			Body:     &obj,
			MetaData: c.config.MetaData,
		}, obj.Ioa)
	}
}

func (c *ClientCall) onIntegratedTotals(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	asduDataList := packet.GetIntegratedTotals()
	logx.WithContext(ctx).Debugf("integrated totals, size: %d", len(asduDataList))
	// [M_IT_NA_1], [M_IT_TA_1] or [M_IT_TB_1]. 获得累计量信息体集合
	for _, p := range asduDataList {
		msgId, _ := tool.SimpleUUID()
		logx.WithContext(ctx).Debugf("integrated totals, msgId: %s, ioa: %d, counter: %d, sq: %d, cy: %t, ca: %t, iv: %t",
			msgId, p.Ioa, p.Value.CounterReading, p.Value.SeqNumber, p.Value.HasCarry, p.Value.IsAdjusted, p.Value.IsInvalid)
		var obj types.BinaryCounterReadingInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, copierx.Option)
		_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
			MsgId:    msgId,
			Host:     c.config.Host,
			Port:     c.config.Port,
			Asdu:     genASDUName(packet.Type),
			TypeId:   int(packet.Type),
			DataType: int(client.GetDataType(packet.Type)),
			Coa:      uint(coa),
			Body:     &obj,
			MetaData: c.config.MetaData,
		}, obj.Ioa)
	}
}

func (c *ClientCall) onEventOfProtectionEquipment(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	asduDataList := packet.GetEventOfProtectionEquipment()
	logx.WithContext(ctx).Debugf("event of protection equipment, size: %d", len(asduDataList))
	// [M_EP_TA_1] [M_EP_TD_1] 获取继电器保护设备事件信息体
	for _, p := range asduDataList {
		msgId, _ := tool.SimpleUUID()
		logx.WithContext(ctx).Debugf("event of protection equipment, msgId: %s, ioa: %d, event: %d, qdp: %d, mesc: %d, time: %d",
			msgId, p.Ioa, p.Event, p.Qdp, p.Msec, p.Time.UnixMilli())
		var obj types.EventOfProtectionEquipmentInfo
		//obj.Time = carbon.Now().ToDateTimeString()
		copier.CopyWithOption(&obj, &p, copierx.Option)
		obj.QdpDesc = util.QdpString(p.Qdp)
		obj.Ei = util.QdpIsElapsedTimeInvalid(p.Qdp)
		obj.Bl = util.QdpIsBlocked(p.Qdp)
		obj.Sb = util.QdpIsSubstituted(p.Qdp)
		obj.Nt = util.QdpIsNotTopical(p.Qdp)
		obj.Iv = util.QdpIsInvalid(p.Qdp)
		_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
			MsgId:    msgId,
			Host:     c.config.Host,
			Port:     c.config.Port,
			Asdu:     genASDUName(packet.Type),
			TypeId:   int(packet.Type),
			DataType: int(client.GetDataType(packet.Type)),
			Coa:      uint(coa),
			Body:     &obj,
			MetaData: c.config.MetaData,
		}, obj.Ioa)
	}
}

func (c *ClientCall) onPackedStartEventsOfProtectionEquipment(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_EP_TB_1] [M_EP_TE_1] 获取继电器保护设备事件信息体
	p := packet.GetPackedStartEventsOfProtectionEquipment()
	msgId, _ := tool.SimpleUUID()
	logx.WithContext(ctx).Debugf("packed start events of protection equipment, msgId: %s, ioa: %d, event: %d, qdp: %d, mesc: %d, time: %d",
		msgId, p.Ioa, p.Event, p.Qdp, p.Msec, p.Time.UnixMilli())
	var obj types.PackedStartEventsOfProtectionEquipmentInfo
	//obj.Time = carbon.Now().ToDateTimeString()
	copier.CopyWithOption(&obj, &p, copierx.Option)
	obj.QdpDesc = util.QdpString(p.Qdp)
	obj.Ei = util.QdpIsElapsedTimeInvalid(p.Qdp)
	obj.Bl = util.QdpIsBlocked(p.Qdp)
	obj.Sb = util.QdpIsSubstituted(p.Qdp)
	obj.Nt = util.QdpIsNotTopical(p.Qdp)
	obj.Iv = util.QdpIsInvalid(p.Qdp)
	_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
		MsgId:    msgId,
		Host:     c.config.Host,
		Port:     c.config.Port,
		Asdu:     genASDUName(packet.Type),
		TypeId:   int(packet.Type),
		DataType: int(client.GetDataType(packet.Type)),
		Coa:      uint(coa),
		Body:     &obj,
		MetaData: c.config.MetaData,
	}, obj.Ioa)
}

func (c *ClientCall) onPackedOutputCircuitInfo(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	// [M_EP_TC_1] [M_EP_TF_1] 获取继电器保护设备成组输出电路信息信息体
	p := packet.GetPackedOutputCircuitInfo()
	msgId, _ := tool.SimpleUUID()
	gc := (p.Oci & asdu.OCIGeneralCommand) != 0
	cl1 := (p.Oci & asdu.OCICommandL1) != 0
	cl2 := (p.Oci & asdu.OCICommandL2) != 0
	cl3 := (p.Oci & asdu.OCICommandL3) != 0
	logx.WithContext(ctx).Debugf("packed Output circuit, msgId: %s, ioa: %d, qci: %d, gc: %v, cl1: %v, cl2: %v, cl3: %v, qdp: %d, mesc: %d, time: %d",
		msgId, p.Ioa, p.Oci, gc, cl1, cl2, cl3, p.Qdp, p.Msec, p.Time.UnixMilli())
	var obj types.PackedOutputCircuitInfoInfo
	//obj.Time = carbon.Now().ToDateTimeString()
	copier.CopyWithOption(&obj, &p, copierx.Option)
	obj.Gc = gc
	obj.Cl1 = cl1
	obj.Cl2 = cl2
	obj.Cl3 = cl3
	obj.QdpDesc = util.QdpString(p.Qdp)
	obj.Ei = util.QdpIsElapsedTimeInvalid(p.Qdp)
	obj.Bl = util.QdpIsBlocked(p.Qdp)
	obj.Sb = util.QdpIsSubstituted(p.Qdp)
	obj.Nt = util.QdpIsNotTopical(p.Qdp)
	obj.Iv = util.QdpIsInvalid(p.Qdp)
	_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
		MsgId:    msgId,
		Host:     c.config.Host,
		Port:     c.config.Port,
		Asdu:     genASDUName(packet.Type),
		TypeId:   int(packet.Type),
		DataType: int(client.GetDataType(packet.Type)),
		Coa:      uint(coa),
		Body:     &obj,
		MetaData: c.config.MetaData,
	}, obj.Ioa)
}

func (c *ClientCall) onPackedSinglePointWithSCD(ctx context.Context, packet *asdu.ASDU) {
	coa := packet.CommonAddr
	asduDataList := packet.GetPackedSinglePointWithSCD()
	logx.WithContext(ctx).Debugf("packed single point with SCD, size: %d", len(asduDataList))
	// [M_PS_NA_1]. 获得带变位检出的成组单点信息
	for _, p := range asduDataList {
		msgId, _ := tool.SimpleUUID()
		logx.WithContext(ctx).Debugf("packed single point with SCD, msgId: %s, ioa: %d, scd: %d, qds: %d", msgId, p.Ioa, p.Scd, p.Qds)
		var obj types.PackedSinglePointWithSCDInfo
		currentStatus := p.Scd & 0xFFFF // 低16位（当前状态）
		stn := fmt.Sprintf("%016b", currentStatus)
		statusChange := (p.Scd >> 16) & 0xFFFF // 高16位（状态变化）
		cdn := fmt.Sprintf("%016b", statusChange)
		var activePoints []int
		var changedPoints []int
		logx.WithContext(ctx).Debugf("stn: %d, %s, cdn: %d, %s", currentStatus, stn, statusChange, cdn)
		for i := 0; i < 16; i++ {
			if currentStatus&(1<<i) != 0 {
				activePoints = append(activePoints, i)
			}
			if statusChange&(1<<i) != 0 {
				changedPoints = append(changedPoints, i)
			}
		}

		logx.WithContext(ctx).Debugf("当前闭合的位: %v", activePoints)
		logx.WithContext(ctx).Debugf("状态变化的位: %v", changedPoints)
		copier.CopyWithOption(&obj, &p, copierx.Option)
		obj.Stn = stn
		obj.Cdn = cdn
		obj.QdsDesc = util.QdsString(p.Qds)
		obj.Ov = util.QdsIsOverflow(p.Qds)
		obj.Bl = util.QdsIsBlocked(p.Qds)
		obj.Sb = util.QdsIsSubstituted(p.Qds)
		obj.Nt = util.QdsIsNotTopical(p.Qds)
		obj.Iv = util.QdsIsInvalid(p.Qds)
		_ = c.svcCtx.PushASDU(ctx, &types.MsgBody{
			MsgId:    msgId,
			Host:     c.config.Host,
			Port:     c.config.Port,
			Asdu:     genASDUName(packet.Type),
			TypeId:   int(packet.Type),
			DataType: int(client.GetDataType(packet.Type)),
			Coa:      uint(coa),
			Body:     &obj,
			MetaData: c.config.MetaData,
		}, obj.Ioa)
	}
}

func genASDUName(typeId asdu.TypeID) string {
	return strutil.SubInBetween(typeId.String(), "<", ">")
}
