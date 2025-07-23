package logic

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/jsonx"

	"zero-service/facade/iecstream/iecstream"
	"zero-service/facade/iecstream/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PushChunkAsduLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushChunkAsduLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushChunkAsduLogic {
	return &PushChunkAsduLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PushChunkAsduLogic) PushChunkAsdu(in *iecstream.PushChunkAsduReq) (*iecstream.PushChunkAsduRes, error) {
	logx.Infof("msgBodySize:%d", len(in.MsgBody))
	//return nil, errors.BadRequest("9999", "暂不支持该类型")
	return &iecstream.PushChunkAsduRes{}, nil
}

func BuildIoaBodyFromJson(dataType int32, data []byte) (any, error) {
	switch dataType {
	case 0: // SinglePointInfo
		body := &iecstream.SinglePointInfo{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal SinglePointInfo: %w", err)
		}
		return body, nil

	case 1: // DoublePointInfo
		body := &iecstream.DoublePointInfo{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal DoublePointInfo: %w", err)
		}
		return body, nil

	case 2: // MeasuredValueScaledInfo
		body := &iecstream.MeasuredValueScaledInfo{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal MeasuredValueScaledInfo: %w", err)
		}
		return body, nil

	case 3: // MeasuredValueNormalInfo
		body := &iecstream.MeasuredValueNormalInfo{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal MeasuredValueNormalInfo: %w", err)
		}
		return body, nil

	case 4: // StepPositionInfo
		body := &iecstream.StepPositionInfo{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal StepPositionInfo: %w", err)
		}
		return body, nil

	case 5: // BitString32Info
		body := &iecstream.BitString32Info{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal BitString32Info: %w", err)
		}
		return body, nil

	case 6: // MeasuredValueFloatInfo
		body := &iecstream.MeasuredValueFloatInfo{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal MeasuredValueFloatInfo: %w", err)
		}
		return body, nil

	case 7: // BinaryCounterReadingInfo
		body := &iecstream.BinaryCounterReadingInfo{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal BinaryCounterReadingInfo: %w", err)
		}
		return body, nil

	case 8: // EventOfProtectionEquipmentInfo
		body := &iecstream.EventOfProtectionEquipmentInfo{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal EventOfProtectionEquipmentInfo: %w", err)
		}
		return body, nil

	case 9: // PackedStartEventsOfProtectionEquipmentInfo
		body := &iecstream.PackedStartEventsOfProtectionEquipmentInfo{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal PackedStartEventsOfProtectionEquipmentInfo: %w", err)
		}
		return body, nil

	case 10: // PackedOutputCircuitInfoInfo
		body := &iecstream.PackedOutputCircuitInfo{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal PackedOutputCircuitInfoInfo: %w", err)
		}
		return body, nil

	case 11: // PackedSinglePointWithSCDInfo
		body := &iecstream.PackedSinglePointWithSCDInfo{}
		if err := jsonx.Unmarshal(data, body); err != nil {
			return nil, fmt.Errorf("unmarshal PackedSinglePointWithSCDInfo: %w", err)
		}
		return body, nil

	case 19:
		return nil, fmt.Errorf("dataType 19 is UNKNOWN, skipping")

	default:
		return nil, fmt.Errorf("unsupported dataType: %d", dataType)
	}
}
