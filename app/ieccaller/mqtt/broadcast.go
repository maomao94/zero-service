package mqtt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/antsx"
	"zero-service/common/iec104/client"
	"zero-service/common/iec104/types"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
)

type Broadcast struct {
	svcCtx *svc.ServiceContext
}

func NewBroadcast(svcCtx *svc.ServiceContext) *Broadcast {
	return &Broadcast{
		svcCtx: svcCtx,
	}
}

func (l *Broadcast) Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
	logx.WithContext(ctx).Debugw("mqtt broadcast received",
		logx.Field("topic", topic),
		logx.Field("topic_template", topicTemplate),
		logx.Field("payload_size", len(payload)),
	)
	if !l.svcCtx.IsBroadcast() {
		logx.WithContext(ctx).Error("mqtt broadcast disabled")
		return nil
	}
	broadcastBody := &types.BroadcastBody{}
	err := jsonx.Unmarshal(payload, broadcastBody)
	if err != nil {
		return err
	}
	if broadcastBody.AckTopic == l.svcCtx.BroadcastAckTopic() {
		logx.WithContext(ctx).Debugw("mqtt broadcast self ignored",
			logx.Field("tid", broadcastBody.Tid),
			logx.Field("method", broadcastBody.Method),
			logx.Field("ack_topic", broadcastBody.AckTopic),
		)
		return nil
	}
	logx.WithContext(ctx).Infow("mqtt broadcast dispatch",
		logx.Field("tid", broadcastBody.Tid),
		logx.Field("method", broadcastBody.Method),
		logx.Field("ack_topic", broadcastBody.AckTopic),
	)
	switch broadcastBody.Method {
	case ieccaller.IecCaller_SendCounterInterrogationCmd_FullMethodName:
		in := &ieccaller.SendCounterInterrogationCmdReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		if err = cli.SendCounterInterrogationCmd(uint16(in.Coa)); err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, "{}", nil)
	case ieccaller.IecCaller_SendInterrogationCmd_FullMethodName:
		in := &ieccaller.SendInterrogationCmdReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		if err = cli.SendInterrogationCmd(uint16(in.Coa)); err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, "{}", nil)
	case ieccaller.IecCaller_SendReadCmd_FullMethodName:
		in := &ieccaller.SendReadCmdReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		if err = cli.SendReadCmd(uint16(in.Coa), uint(in.Ioa)); err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, "{}", nil)
	case ieccaller.IecCaller_SendTestCmd_FullMethodName:
		in := &ieccaller.SendTestCmdReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		if err = cli.SendTestCmd(uint16(in.Coa)); err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, "{}", nil)
	case ieccaller.IecCaller_SendCommand_FullMethodName:
		in := &ieccaller.SendCommandReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		if err = cli.SendCmd(uint16(in.Coa), asdu.TypeID(in.TypeId), asdu.InfoObjAddr(in.Ioa), in.Value); err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, "{}", nil)
	case ieccaller.IecCaller_SendSingleCommand_FullMethodName:
		in := &ieccaller.SendSingleCommandReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		ack, err := cli.SendSingleCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), in.Value, in.WithTime, client.WithAck())
		if err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		value, ok := ack.Value.(bool)
		if !ok {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", fmt.Errorf("unexpected ack value type"))
			return nil
		}
		resJson, _ := jsonx.Marshal(&ieccaller.SendSingleCommandRes{Value: value})
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, string(resJson), nil)
	case ieccaller.IecCaller_SendDoubleCommand_FullMethodName:
		in := &ieccaller.SendDoubleCommandReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		ack, err := cli.SendDoubleCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), asdu.DoubleCommand(in.Value), in.WithTime, client.WithAck())
		if err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		value, ok := ack.Value.(asdu.DoubleCommand)
		if !ok {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", fmt.Errorf("unexpected ack value type"))
			return nil
		}
		resJson, _ := jsonx.Marshal(&ieccaller.SendDoubleCommandRes{Value: ieccaller.DoubleCommandValue(int32(value))})
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, string(resJson), nil)
	case ieccaller.IecCaller_SendStepCommand_FullMethodName:
		in := &ieccaller.SendStepCommandReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		ack, err := cli.SendStepCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), asdu.StepCommand(in.Value), in.WithTime, client.WithAck())
		if err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		value, ok := ack.Value.(asdu.StepCommand)
		if !ok {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", fmt.Errorf("unexpected ack value type"))
			return nil
		}
		resJson, _ := jsonx.Marshal(&ieccaller.SendStepCommandRes{Value: int32(value)})
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, string(resJson), nil)
	case ieccaller.IecCaller_SendSetpointNormalized_FullMethodName:
		in := &ieccaller.SendSetpointNormalizedReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		ack, err := cli.SendSetpointNormalizedCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), int16(in.Value), in.WithTime, client.WithAck())
		if err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		value, ok := ack.Value.(asdu.Normalize)
		if !ok {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", fmt.Errorf("unexpected ack value type"))
			return nil
		}
		resJson, _ := jsonx.Marshal(&ieccaller.SendSetpointNormalizedRes{Value: int32(value)})
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, string(resJson), nil)
	case ieccaller.IecCaller_SendSetpointScaled_FullMethodName:
		in := &ieccaller.SendSetpointScaledReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		ack, err := cli.SendSetpointScaledCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), int16(in.Value), in.WithTime, client.WithAck())
		if err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		value, ok := ack.Value.(int16)
		if !ok {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", fmt.Errorf("unexpected ack value type"))
			return nil
		}
		resJson, _ := jsonx.Marshal(&ieccaller.SendSetpointScaledRes{Value: int32(value)})
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, string(resJson), nil)
	case ieccaller.IecCaller_SendSetpointFloat_FullMethodName:
		in := &ieccaller.SendSetpointFloatReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		fv, err := convertor.ToFloat(in.Value)
		if err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", fmt.Errorf("invalid float value: %s", in.Value))
			return nil
		}
		ack, err := cli.SendSetpointFloatCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), float32(fv), in.WithTime, client.WithAck())
		if err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		ackValue, ok := ack.Value.(float32)
		if !ok {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", fmt.Errorf("unexpected ack value type"))
			return nil
		}
		resJson, _ := jsonx.Marshal(&ieccaller.SendSetpointFloatRes{Value: convertor.ToString(ackValue)})
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, string(resJson), nil)
	case ieccaller.IecCaller_SendBitstringCommand_FullMethodName:
		in := &ieccaller.SendBitstringCommandReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logBroadcastClientError(ctx, broadcastBody, in.Host, in.Port, err)
			return nil
		}
		ack, err := cli.SendBitstringCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), uint32(in.Value), in.WithTime, client.WithAck())
		if err != nil {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
			return nil
		}
		value, ok := ack.Value.(uint32)
		if !ok {
			l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", fmt.Errorf("unexpected ack value type"))
			return nil
		}
		resJson, _ := jsonx.Marshal(&ieccaller.SendBitstringCommandRes{Value: uint64(value)})
		l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, string(resJson), nil)
	case ieccaller.IecCaller_ClearPointMappingCache_FullMethodName:
		in := &ieccaller.ClearPointMappingCacheReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		clearedCount := int64(0)
		if l.svcCtx.DevicePointMappingModel != nil {
			if len(in.Keys) > 0 {
				for _, key := range in.Keys {
					if _, exists := l.svcCtx.DevicePointMappingModel.GetCache(ctx, key); exists {
						if err := l.svcCtx.DevicePointMappingModel.RemoveCache(ctx, key); err != nil {
							logx.WithContext(ctx).Errorw("mqtt broadcast cache remove failed",
								logx.Field("key", key),
								logx.Field("error", err),
							)
							continue
						}
						clearedCount++
					}
				}
			}
			if len(in.KeyInfos) > 0 {
				for _, info := range in.KeyInfos {
					key := l.svcCtx.DevicePointMappingModel.GenerateCacheKey(info.TagStation, info.Coa, info.Ioa)
					if _, exists := l.svcCtx.DevicePointMappingModel.GetCache(ctx, key); exists {
						if err := l.svcCtx.DevicePointMappingModel.RemoveCache(ctx, key); err != nil {
							logx.WithContext(ctx).Errorw("mqtt broadcast cache remove failed",
								logx.Field("tag_station", info.TagStation),
								logx.Field("coa", info.Coa),
								logx.Field("ioa", info.Ioa),
								logx.Field("error", err),
							)
							continue
						}
						clearedCount++
					}
				}
			}
			logx.WithContext(ctx).Infow("mqtt broadcast cache cleared", logx.Field("cleared_count", clearedCount))
		}
	default:
		logx.WithContext(ctx).Errorw("mqtt broadcast unknown method",
			logx.Field("tid", broadcastBody.Tid),
			logx.Field("method", broadcastBody.Method),
		)
	}
	return nil
}

func logBroadcastClientError(ctx context.Context, body *types.BroadcastBody, host string, port uint32, err error) {
	logx.WithContext(ctx).Errorw("mqtt broadcast client get failed",
		logx.Field("tid", body.Tid),
		logx.Field("method", body.Method),
		logx.Field("ack_topic", body.AckTopic),
		logx.Field("host", host),
		logx.Field("port", port),
		logx.Field("error", err),
	)
}

func (l *Broadcast) publishAckReply(ctx context.Context, tId, ackTopic, method string, success bool, responseBody string, ackErr error) {
	if tId == "" {
		logx.WithContext(ctx).Debugw("mqtt broadcast ack reply skipped",
			logx.Field("reason", "empty_tid"),
			logx.Field("method", method),
			logx.Field("ack_topic", ackTopic),
		)
		return
	}
	errorKind := ""
	errMsg := ""
	if ackErr != nil {
		errMsg = ackErr.Error()
		switch {
		case errors.Is(ackErr, antsx.ErrReplyExpired):
			errorKind = "timeout"
		case errors.Is(ackErr, antsx.ErrDuplicateID):
			errorKind = "duplicate"
		case isCommandRejectedError(ackErr):
			errorKind = "iec_rejected"
		default:
			errorKind = "unknown"
		}
		logx.WithContext(ctx).Infow("mqtt broadcast ack reply prepared",
			logx.Field("tid", tId),
			logx.Field("method", method),
			logx.Field("ack_topic", ackTopic),
			logx.Field("success", success),
			logx.Field("error_kind", errorKind),
			logx.Field("error", errMsg),
		)
	} else {
		logx.WithContext(ctx).Infow("mqtt broadcast ack reply prepared",
			logx.Field("tid", tId),
			logx.Field("method", method),
			logx.Field("ack_topic", ackTopic),
			logx.Field("success", true),
		)
	}
	ack := &types.BroadcastAckBody{
		Tid:          tId,
		Method:       method,
		Success:      success,
		ResponseBody: responseBody,
		Error:        errMsg,
		ErrorKind:    errorKind,
	}
	data, err := jsonx.Marshal(ack)
	if err != nil {
		logx.WithContext(ctx).Errorw("mqtt broadcast ack marshal failed", logx.Field("error", err))
		return
	}
	if l.svcCtx.MqttClient == nil {
		logx.WithContext(ctx).Error("mqtt broadcast client is nil")
		return
	}
	pushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if ackTopic == "" {
		logx.WithContext(ctx).Errorw("mqtt broadcast ack topic is empty",
			logx.Field("tid", tId),
			logx.Field("method", method),
		)
		return
	}
	if _, err := l.svcCtx.MqttClient.PublishWithTrace(pushCtx, ackTopic, data); err != nil {
		logx.WithContext(pushCtx).Errorw("mqtt broadcast ack publish failed",
			logx.Field("tid", tId),
			logx.Field("method", method),
			logx.Field("ack_topic", ackTopic),
			logx.Field("error", err),
		)
	} else {
		logx.WithContext(pushCtx).Debugw("mqtt broadcast ack reply sent",
			logx.Field("tid", tId),
			logx.Field("method", method),
			logx.Field("ack_topic", ackTopic),
		)
	}
}

func isCommandRejectedError(err error) bool {
	var rejected *client.CommandRejectedError
	return errors.As(err, &rejected)
}
