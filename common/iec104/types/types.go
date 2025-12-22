package types

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/zeromicro/go-zero/core/mapping"
)

type BroadcastBody struct {
	BroadcastGroupId string `json:"broadcastGroupId"`
	Method           string `json:"method"`
	Body             string `json:"body"`
}

type MsgBody struct {
	MsgId    string         `json:"msgId"`
	Host     string         `json:"host"`
	Port     int            `json:"port"`
	Asdu     string         `json:"asdu"`
	TypeId   int            `json:"typeId"`
	DataType int            `json:"dataType"`
	Coa      uint           `json:"coa"` // 公共地址
	Body     IoaGetter      `json:"body"`
	Time     string         `json:"time"`
	MetaData map[string]any `json:"metaData"`
	Pm       *PointMapping  `json:"pm,omitempty"`
}

type PointMapping struct {
	DeviceId    string `json:"deviceId"`    // 设备编号/ID
	DeviceName  string `json:"deviceName"`  // 设备名称
	TdTableType string `json:"tdTableType"` // TDengine 表类型（遥信表/遥测表等，逗号分隔）
	Ext1        string `json:"ext1"`        // 扩展字段1，如：alarm, normal, control等，用于主题拆分
	Ext2        string `json:"ext2"`        // 扩展字段2
	Ext3        string `json:"ext3"`        // 扩展字段3
	Ext4        string `json:"ext4"`        // 扩展字段4
	Ext5        string `json:"ext5"`        // 扩展字段5
}

func (m *MsgBody) GetKey() (string, error) {
	if m.Body == nil {
		return "", errors.New("body is nil")
	}
	rv := reflect.ValueOf(m.Body)
	if err := mapping.ValidatePtr(rv); err != nil {
		return "", err
	}
	//coaHex := fmt.Sprintf("0x%x", m.Coa)
	coa := fmt.Sprintf("%d", m.Coa)
	ioaHex := fmt.Sprintf("0x%06x", m.Body.GetIoa())
	return fmt.Sprintf("%s_%s_%s", m.Host, coa, ioaHex), nil
}

type IoaGetter interface {
	GetIoa() uint
}

// asdu.M_SP_NA_1, asdu.M_SP_TA_1, asdu.M_SP_TB_1
// 单点信息体
type SinglePointInfo struct {
	Ioa     uint   `json:"ioa"`   // 信息对象地址
	Value   bool   `json:"value"` // 状态值
	Qds     byte   `json:"qds"`
	QdsDesc string `json:"qdsDesc"`
	Ov      bool   `json:"ov"`
	Bl      bool   `json:"bl"`
	Sb      bool   `json:"sb"`
	Nt      bool   `json:"nt"`
	Iv      bool   `json:"iv"`
	Time    string `json:"time"`
}

func (s *SinglePointInfo) GetIoa() uint {
	return s.Ioa
}

// asdu.M_DP_NA_1, asdu.M_DP_TA_1, asdu.M_DP_TB_1
// 双点信息体
type DoublePointInfo struct {
	Ioa     uint   `json:"ioa"`   // 信息对象地址
	Value   byte   `json:"value"` // 状态值
	Qds     byte   `json:"qds"`
	QdsDesc string `json:"qdsDesc"`
	Ov      bool   `json:"ov"`
	Bl      bool   `json:"bl"`
	Sb      bool   `json:"sb"`
	Nt      bool   `json:"nt"`
	Iv      bool   `json:"iv"`
	Time    string `json:"time"`
}

func (s *DoublePointInfo) GetIoa() uint {
	return s.Ioa
}

// asdu.M_ME_NB_1, asdu.M_ME_TB_1, asdu.M_ME_TE_1
// 测量值,标度化值信息
type MeasuredValueScaledInfo struct {
	Ioa     uint   `json:"ioa"` // 信息对象地址
	Value   int16  `json:"value"`
	Qds     byte   `json:"qds"`
	QdsDesc string `json:"qdsDesc"`
	Ov      bool   `json:"ov"`
	Bl      bool   `json:"bl"`
	Sb      bool   `json:"sb"`
	Nt      bool   `json:"nt"`
	Iv      bool   `json:"iv"`
	Time    string `json:"time"`
}

func (s *MeasuredValueScaledInfo) GetIoa() uint {
	return s.Ioa
}

// asdu.M_ME_NA_1, asdu.M_ME_TA_1, asdu.M_ME_TD_1, asdu.M_ME_ND_1
// 测量值,规一化值信息
type MeasuredValueNormalInfo struct {
	Ioa uint `json:"ioa"` // 信息对象地址
	// Normalize is a 16-bit normalized value in[-1, 1 − 2⁻¹⁵]..
	// 规一化值 f归一= 32768 * f真实 / 满码值
	// See companion standard 101, subclass 7.2.6.6.
	Value   int16   `json:"value"`
	Nva     float32 `json:"nva"`
	Qds     byte    `json:"qds"`
	QdsDesc string  `json:"qdsDesc"`
	Ov      bool    `json:"ov"`
	Bl      bool    `json:"bl"`
	Sb      bool    `json:"sb"`
	Nt      bool    `json:"nt"`
	Iv      bool    `json:"iv"`
	Time    string  `json:"time"`
}

func (s *MeasuredValueNormalInfo) GetIoa() uint {
	return s.Ioa
}

// asdu.M_ST_NA_1, asdu.M_ST_TA_1, asdu.M_ST_TB_1
// 步位置信息
type StepPositionInfo struct {
	Ioa     uint         `json:"ioa"` // 信息对象地址
	Value   StepPosition `json:"value"`
	Qds     byte         `json:"qds"`
	QdsDesc string       `json:"qdsDesc"`
	Ov      bool         `json:"ov"`
	Bl      bool         `json:"bl"`
	Sb      bool         `json:"sb"`
	Nt      bool         `json:"nt"`
	Iv      bool         `json:"iv"`
	Time    string       `json:"time"`
}

// StepPosition is a measured value with transient state indication.
// 带瞬变状态指示的测量值，用于变压器步位置或其它步位置的值
// See companion standard 101, subclass 7.2.6.5.
// Val range <-64..63>
// bit[0-5]: <-64..63>
// NOTE: bit6 为符号位
// bit7: 0: 设备未在瞬变状态 1： 设备处于瞬变状态
type StepPosition struct {
	Val          int  `json:"val"`
	HasTransient bool `json:"hasTransient"`
}

func (s *StepPositionInfo) GetIoa() uint {
	return s.Ioa
}

// asdu.M_BO_NA_1, asdu.M_BO_TA_1, asdu.M_BO_TB_1
type BitString32Info struct {
	Ioa     uint   `json:"ioa"` // 信息对象地址
	Value   uint32 `json:"value"`
	Qds     byte   `json:"qds"`
	QdsDesc string `json:"qdsDesc"`
	Ov      bool   `json:"ov"`
	Bl      bool   `json:"bl"`
	Sb      bool   `json:"sb"`
	Nt      bool   `json:"nt"`
	Iv      bool   `json:"iv"`
	Time    string `json:"time"`
}

func (s *BitString32Info) GetIoa() uint {
	return s.Ioa
}

// asdu.M_ME_NC_1, asdu.M_ME_TC_1, asdu.M_ME_TF_1
// 测量值,短浮点数信息
type MeasuredValueFloatInfo struct {
	Ioa     uint    `json:"ioa"` // 信息对象地址
	Value   float32 `json:"value"`
	Qds     byte    `json:"qds"`
	QdsDesc string  `json:"qdsDesc"`
	Ov      bool    `json:"ov"`
	Bl      bool    `json:"bl"`
	Sb      bool    `json:"sb"`
	Nt      bool    `json:"nt"`
	Iv      bool    `json:"iv"`
	Time    string  `json:"time"`
}

func (s *MeasuredValueFloatInfo) GetIoa() uint {
	return s.Ioa
}

// asdu.M_IT_NA_1, asdu.M_IT_TA_1, asdu.M_IT_TB_1
// 累计量信息体
type BinaryCounterReadingInfo struct {
	Ioa   uint                 `json:"ioa"` // 信息对象地址
	Value BinaryCounterReading `json:"value"`
	Time  string               `json:"time"`
}

// BinaryCounterReading is binary counter reading
// See companion standard 101, subclass 7.2.6.9.
// CounterReading: 计数器读数 [bit0...bit31]
// SeqNumber: 顺序记法 [bit32...bit40]
// SQ: 顺序号 [bit32...bit36]
// CY: 进位 [bit37]
// CA: 计数量被调整
// IV: 无效
type BinaryCounterReading struct {
	CounterReading int32 `json:"counterReading"`
	SeqNumber      byte  `json:"seqNumber"`
	HasCarry       bool  `json:"hasCarry"`
	IsAdjusted     bool  `json:"isAdjusted"`
	IsInvalid      bool  `json:"isInvalid"`
}

func (s *BinaryCounterReadingInfo) GetIoa() uint {
	return s.Ioa
}

// asdu.M_EP_TA_1, asdu.M_EP_TD_1
// asdu.M_EP_TD_1 EOF
// 继电器保护设备事件信息
type EventOfProtectionEquipmentInfo struct {
	Ioa     uint   `json:"ioa"` // 信息对象地址
	Event   byte   `json:"event"`
	Qdp     byte   `json:"qdp"`
	QdpDesc string `json:"qdpDesc"`
	Ei      bool   `json:"ei"`
	Bl      bool   `json:"bl"`
	Sb      bool   `json:"sb"`
	Nt      bool   `json:"nt"`
	Iv      bool   `json:"iv"`
	Msec    uint16 `json:"msec"`
	Time    string `json:"time"`
}

func (s *EventOfProtectionEquipmentInfo) GetIoa() uint {
	return s.Ioa
}

// asdu.M_EP_TB_1, asdu.M_EP_TE_1
// 继电器保护设备成组启动事件
type PackedStartEventsOfProtectionEquipmentInfo struct {
	Ioa     uint   `json:"ioa"` // 信息对象地址
	Event   byte   `json:"event"`
	Qdp     byte   `json:"qdp"`
	QdpDesc string `json:"qdpDesc"`
	Ei      bool   `json:"ei"`
	Bl      bool   `json:"bl"`
	Sb      bool   `json:"sb"`
	Nt      bool   `json:"nt"`
	Iv      bool   `json:"iv"`
	Msec    uint16 `json:"msec"`
	Time    string `json:"time"`
}

func (s *PackedStartEventsOfProtectionEquipmentInfo) GetIoa() uint {
	return s.Ioa
}

// asdu.M_EP_TC_1, asdu.M_EP_TF_1
// 继电器保护设备成组输出电路信息
type PackedOutputCircuitInfoInfo struct {
	Ioa     uint   `json:"ioa"` // 信息对象地址
	Oci     byte   `json:"oci"`
	Gc      bool   `json:"gc"`
	Cl1     bool   `json:"cl1"`
	Cl2     bool   `json:"cl2"`
	Cl3     bool   `json:"cl3"`
	Qdp     byte   `json:"qdp"`
	QdpDesc string `json:"qdpDesc"`
	Ei      bool   `json:"ei"`
	Bl      bool   `json:"bl"`
	Sb      bool   `json:"sb"`
	Nt      bool   `json:"nt"`
	Iv      bool   `json:"iv"`
	Msec    uint16 `json:"msec"`
	// the type does not include timing will ignore
	Time string `json:"time"`
}

func (s *PackedOutputCircuitInfoInfo) GetIoa() uint {
	return s.Ioa
}

// asdu.M_PS_NA_1
// 带变位检出的成组单点信息
type PackedSinglePointWithSCDInfo struct {
	Ioa uint `json:"ioa"` // 信息对象地址
	// StatusAndStatusChangeDetection 状态和状态变位检出
	// See companion standard 101, subclass 7.2.6.40.
	Scd     uint32 `json:"scd"`
	Stn     string `json:"stn"`
	Cdn     string `json:"cdn"`
	Qds     byte   `json:"qds"`
	QdsDesc string `json:"qdsDesc"`
	Ov      bool   `json:"ov"`
	Bl      bool   `json:"bl"`
	Sb      bool   `json:"sb"`
	Nt      bool   `json:"nt"`
	Iv      bool   `json:"iv"`
}

func (s *PackedSinglePointWithSCDInfo) GetIoa() uint {
	return s.Ioa
}
