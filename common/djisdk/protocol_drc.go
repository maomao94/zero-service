package djisdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// DRC 协议模型对应官方 DRC 上云文档的 drc/down 与 drc/up 分节。
// drc/down 是云平台发布到 thing/product/{gateway_sn}/drc/down 的实时控制通道；drc/up 是设备发布到 thing/product/{gateway_sn}/drc/up 的状态与回执通道。
// 进入/退出 DRC 模式、飞行控制权抢占等仍走 services/services_reply，不属于 drc/down 或 drc/up 子路径。

// DrcDownMessage 云平台经 drc/down 下发的通用报文。
// stick_control 与 heart_beat 的 seq 位于 data 同级，调用方应按 method 选择对应 payload。
type DrcDownMessage struct {
	// Tid 事务 ID。建议填写 UUID；若设备不校验可仍填。
	Tid string `json:"tid,omitempty"`
	// Bid 业务 ID。
	Bid string `json:"bid,omitempty"`
	// Timestamp 报文时间戳，毫秒；未填时由 NewDrcDownMessage 填当前时间。
	Timestamp int64 `json:"timestamp"`
	// Method 见 [MethodDrcHeartBeat]、[MethodDroneEmergencyStop] 等。
	Method string `json:"method"`
	// Data 载荷，随 method 变化。
	Data any `json:"data"`
	// Seq 与 data 同级时递增（如 DRC-心跳），可选。
	Seq *int `json:"seq,omitempty"`
}

// DrcHeartBeatDownData 云→设备 **drc/down** `heart_beat` 的 data 体。
type DrcHeartBeatDownData struct {
	// Timestamp 心跳时间戳，毫秒；协议说明用于 DRC 链路保活判断。
	Timestamp int64 `json:"timestamp"`
}

// NewDrcDownMessage 创建 drc/down 报文，自动填 Timestamp 为当前毫秒时间。
func NewDrcDownMessage(tid, bid, method string, data any, seq *int) *DrcDownMessage {
	return &DrcDownMessage{
		Tid:       tid,
		Bid:       bid,
		Timestamp: time.Now().UnixMilli(),
		Method:    method,
		Data:      data,
		Seq:       seq,
	}
}

// DrcUpMessage 是设备经 drc/up 上行的通用报文壳。
// Data 保留原始 data 字段；已知 method 可通过 DrcUnmarshalUpData 转为强类型，未知 method 会以 DrcUnknownUpData 继续分发，避免新增协议字段阻断业务 hook。
type DrcUpMessage struct {
	// Tid 事务 ID，设备侧按协议携带；部分上行消息可能省略。
	Tid string `json:"tid,omitempty"`
	// Bid 业务 ID，设备侧按协议携带；部分上行消息可能省略。
	Bid string `json:"bid,omitempty"`
	// Timestamp 设备上报时间戳，单位毫秒。
	Timestamp int64 `json:"timestamp"`
	// Method DRC 上行方法名，如 drc_initial_state_subscribe、heart_beat、hsi_info_push。
	Method string `json:"method"`
	// Gateway 网关设备 SN；部分固件仅在 topic 中体现。
	Gateway string `json:"gateway,omitempty"`
	// Data DRC 上行业务载荷，结构随 Method 变化。
	Data json.RawMessage `json:"data"`
	// Seq 部分 method（如 heart_beat、drc_initial_state_subscribe）在顶层与 data 同级出现。
	Seq int `json:"seq,omitempty"`
}

// DrcUnknownUpData 保存 SDK 尚未建模的 drc/up data 原文，便于 hook 继续分发和业务侧按需扩展。
type DrcUnknownUpData struct {
	// Method 原始 DRC 上行方法名。
	Method string `json:"method"`
	// Raw 原始 data 字段内容，不做结构化解释。
	Raw json.RawMessage `json:"raw"`
}

// DrcUpMessageFromJSON 解析 drc/up 报文，兼容 data:null 与缺省 data。
func DrcUpMessageFromJSON(payload []byte) (*DrcUpMessage, error) {
	var w struct {
		Tid       string           `json:"tid,omitempty"`
		Bid       string           `json:"bid,omitempty"`
		Timestamp int64            `json:"timestamp"`
		Method    string           `json:"method"`
		Gateway   string           `json:"gateway,omitempty"`
		Seq       int              `json:"seq,omitempty"`
		Data      *json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &w); err != nil {
		return nil, err
	}
	m := &DrcUpMessage{
		Tid: w.Tid, Bid: w.Bid, Timestamp: w.Timestamp, Method: w.Method, Gateway: w.Gateway, Seq: w.Seq,
	}
	if w.Data != nil {
		m.Data = *w.Data
	} else {
		m.Data = nil
	}
	return m, nil
}

// DrcStickControlAckData 是设备在 drc/up 对 `stick_control` 的执行回执。
type DrcStickControlAckData struct {
	// Result 返回码，0 表示成功，非 0 对应 DJI 设备错误码。
	Result int `json:"result"`
	// Output 携带设备确认的 seq，用于与 drc/down 杆量序号对应。
	Output *struct {
		// Seq 设备确认的杆量序号。
		Seq int `json:"seq"`
	} `json:"output,omitempty"`
}

// DrcDroneEmergencyStopUpData 是设备在 drc/up 对 `drone_emergency_stop` 的执行回执。
type DrcDroneEmergencyStopUpData struct {
	// Result 返回码，0 表示成功，非 0 对应 DJI 设备错误码。
	Result int `json:"result"`
}

// DrcHeartBeatUpData 是设备在 drc/up `heart_beat` 中回传的 DRC 链路心跳。
type DrcHeartBeatUpData struct {
	// Timestamp 设备侧心跳时间戳，单位毫秒。
	Timestamp int64 `json:"timestamp"`
	// Seq 文档标为 deprecated，部分固件仍带。
	Seq int `json:"seq,omitempty"`
}

// DrcInitialStateSubscribeUpData 设备在 drc/up 对 `drc_initial_state_subscribe` 的回执数据。
//
// 官方 up 方向 data 仅包含 result 字段，非 0 代表设备侧执行错误。
type DrcInitialStateSubscribeUpData struct {
	// Result 返回码，0 表示成功，非 0 对应 DJI 设备错误码。
	Result int `json:"result"`
}

// DrcHsiInfoPushData 是设备在 drc/up `hsi_info_push` 中上报的避障与水平态势信息。
// 设备示例中数组字段名为 `around_distance`，与表头 `around_distances` 可能并存，UnmarshalJSON 会同时兼容。
type DrcHsiInfoPushData struct {
	// UpDistance 上方障碍物距离，单位以官方文档为准。
	UpDistance int `json:"up_distance"`
	// DownDistance 下方障碍物距离，单位以官方文档为准。
	DownDistance int `json:"down_distance"`
	// UpEnable 上方避障能力是否启用。
	UpEnable bool `json:"up_enable"`
	// UpWork 上方避障是否处于工作状态。
	UpWork bool `json:"up_work"`
	// DownEnable 下方避障能力是否启用。
	DownEnable bool `json:"down_enable"`
	// DownWork 下方避障是否处于工作状态。
	DownWork bool `json:"down_work"`
	// LeftEnable 左侧避障能力是否启用。
	LeftEnable bool `json:"left_enable"`
	// LeftWork 左侧避障是否处于工作状态。
	LeftWork bool `json:"left_work"`
	// RightEnable 右侧避障能力是否启用。
	RightEnable bool `json:"right_enable"`
	// RightWork 右侧避障是否处于工作状态。
	RightWork bool `json:"right_work"`
	// FrontEnable 前方避障能力是否启用。
	FrontEnable bool `json:"front_enable"`
	// FrontWork 前方避障是否处于工作状态。
	FrontWork bool `json:"front_work"`
	// BackEnable 后方避障能力是否启用。
	BackEnable bool `json:"back_enable"`
	// BackWork 后方避障是否处于工作状态。
	BackWork bool `json:"back_work"`
	// VerticalEnable 垂直方向避障能力是否启用。
	VerticalEnable bool `json:"vertical_enable"`
	// VerticalWork 垂直方向避障是否处于工作状态。
	VerticalWork bool `json:"vertical_work"`
	// HorizontalEnable 水平方向避障能力是否启用。
	HorizontalEnable bool `json:"horizontal_enable"`
	// HorizontalWork 水平方向避障是否处于工作状态。
	HorizontalWork bool `json:"horizontal_work"`
	// AroundDistances 周向距离数组；兼容 JSON 中 around_distance 与 around_distances 两种键名。
	AroundDistances []int `json:"-"`
}

// UnmarshalJSON 兼容 `around_distance` 与 `around_distances` 键名（见官方示例与表头差异）。
func (d *DrcHsiInfoPushData) UnmarshalJSON(b []byte) error {
	var s struct {
		UpDistance       int   `json:"up_distance"`
		DownDistance     int   `json:"down_distance"`
		UpEnable         bool  `json:"up_enable"`
		UpWork           bool  `json:"up_work"`
		DownEnable       bool  `json:"down_enable"`
		DownWork         bool  `json:"down_work"`
		LeftEnable       bool  `json:"left_enable"`
		LeftWork         bool  `json:"left_work"`
		RightEnable      bool  `json:"right_enable"`
		RightWork        bool  `json:"right_work"`
		FrontEnable      bool  `json:"front_enable"`
		FrontWork        bool  `json:"front_work"`
		BackEnable       bool  `json:"back_enable"`
		BackWork         bool  `json:"back_work"`
		VerticalEnable   bool  `json:"vertical_enable"`
		VerticalWork     bool  `json:"vertical_work"`
		HorizontalEnable bool  `json:"horizontal_enable"`
		HorizontalWork   bool  `json:"horizontal_work"`
		AroundDistance   []int `json:"around_distance"`
		AroundDistances  []int `json:"around_distances"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	d.UpDistance = s.UpDistance
	d.DownDistance = s.DownDistance
	d.UpEnable = s.UpEnable
	d.UpWork = s.UpWork
	d.DownEnable = s.DownEnable
	d.DownWork = s.DownWork
	d.LeftEnable = s.LeftEnable
	d.LeftWork = s.LeftWork
	d.RightEnable = s.RightEnable
	d.RightWork = s.RightWork
	d.FrontEnable = s.FrontEnable
	d.FrontWork = s.FrontWork
	d.BackEnable = s.BackEnable
	d.BackWork = s.BackWork
	d.VerticalEnable = s.VerticalEnable
	d.VerticalWork = s.VerticalWork
	d.HorizontalEnable = s.HorizontalEnable
	d.HorizontalWork = s.HorizontalWork
	if len(s.AroundDistances) > 0 {
		d.AroundDistances = s.AroundDistances
	} else {
		d.AroundDistances = s.AroundDistance
	}
	return nil
}

// DrcDelayInfoPushData 是设备在 drc/up `delay_info_push` 中上报的控制链路与图传链路时延。
type DrcDelayInfoPushData struct {
	// SdrCmdDelay SDR 控制指令链路时延，单位以官方文档为准。
	SdrCmdDelay int `json:"sdr_cmd_delay"`
	// LiveviewDelayList 多路图传码流的时延列表。
	LiveviewDelayList []DrcLiveviewDelayItem `json:"liveview_delay_list"`
}

// DrcLiveviewDelayItem 表示一路图传码流的时延信息。
type DrcLiveviewDelayItem struct {
	// VideoID 视频流标识。
	VideoID string `json:"video_id"`
	// LiveviewDelayTime 图传时延，单位以官方文档为准。
	LiveviewDelayTime int `json:"liveview_delay_time"`
}

// DrcOsdInfoPushData 是设备在 drc/up `osd_info_push` 中上报的高频姿态、位置与云台状态。
type DrcOsdInfoPushData struct {
	// AttitudeHead 飞行器航向角。
	AttitudeHead float64 `json:"attitude_head"`
	// Latitude 纬度。
	Latitude float64 `json:"latitude"`
	// Longitude 经度。
	Longitude float64 `json:"longitude"`
	// Height 高度。
	Height float64 `json:"height"`
	// SpeedX X 轴速度。
	SpeedX float64 `json:"speed_x"`
	// SpeedY Y 轴速度。
	SpeedY float64 `json:"speed_y"`
	// SpeedZ Z 轴速度。
	SpeedZ float64 `json:"speed_z"`
	// GimbalPitch 云台俯仰角。
	GimbalPitch float64 `json:"gimbal_pitch"`
	// GimbalRoll 云台横滚角。
	GimbalRoll float64 `json:"gimbal_roll"`
	// GimbalYaw 云台偏航角。
	GimbalYaw float64 `json:"gimbal_yaw"`
}

// DrcUnmarshalUpData 按 method 将 drc/up 的 data 反序列化为强类型。
// 已知 method 返回对应结构；未知 method 返回 DrcUnknownUpData 且不报错，保证 hook 分发不中断。
func DrcUnmarshalUpData(method string, data json.RawMessage) (any, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	switch method {
	case MethodStickControl:
		var v DrcStickControlAckData
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case MethodDroneEmergencyStop:
		var v DrcDroneEmergencyStopUpData
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case MethodDrcHeartBeat:
		var v DrcHeartBeatUpData
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case MethodDrcInitialStateSubscribe:
		var v DrcInitialStateSubscribeUpData
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case MethodDrcHsiInfoPush:
		var v DrcHsiInfoPushData
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case MethodDrcDelayInfoPush:
		var v DrcDelayInfoPushData
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case MethodDrcOsdInfoPush:
		var v DrcOsdInfoPushData
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	default:
		return &DrcUnknownUpData{Method: method, Raw: append(json.RawMessage(nil), data...)}, nil
	}
}

// DrcUpPayloadSummary 将已解析的 drc/up data 转为短摘要，供日志和排障使用。
func DrcUpPayloadSummary(method string, parsed any) string {
	if parsed == nil {
		return ""
	}
	switch t := parsed.(type) {
	case *DrcStickControlAckData:
		if t == nil {
			return ""
		}
		return fmt.Sprintf("result=%d", t.Result)
	case *DrcDroneEmergencyStopUpData:
		if t == nil {
			return ""
		}
		return fmt.Sprintf("result=%d", t.Result)
	case *DrcHeartBeatUpData:
		if t == nil {
			return ""
		}
		return fmt.Sprintf("ts=%d", t.Timestamp)
	case *DrcInitialStateSubscribeUpData:
		if t == nil {
			return ""
		}
		return fmt.Sprintf("result=%d", t.Result)
	case *DrcHsiInfoPushData:
		if t == nil {
			return ""
		}
		return fmt.Sprintf("up=%d down=%d around=%dpts", t.UpDistance, t.DownDistance, len(t.AroundDistances))
	case *DrcDelayInfoPushData:
		if t == nil {
			return ""
		}
		return fmt.Sprintf("sdr_cmd_delay=%d streams=%d", t.SdrCmdDelay, len(t.LiveviewDelayList))
	case *DrcOsdInfoPushData:
		if t == nil {
			return ""
		}
		return fmt.Sprintf("h=%.1f lat=%.4f lon=%.4f", t.Height, t.Latitude, t.Longitude)
	case *DrcUnknownUpData:
		if t == nil {
			return ""
		}
		return fmt.Sprintf("unknown raw_bytes=%d", len(t.Raw))
	default:
		return fmt.Sprintf("%T", parsed)
	}
}
