package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type DialSipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDialSipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DialSipLogic {
	return &DialSipLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DialSip 发起 SIP 外呼。
func (l *DialSipLogic) DialSip(in *live.DialSipReq) (*live.DialSipRes, error) {
	calleeNumber := strings.TrimSpace(in.GetCalleeNumber())
	if calleeNumber == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "被叫号码不能为空")
	}

	meetingNo := strings.TrimSpace(in.GetMeetingNo())
	callType := "phone"

	// 1. 确定会议
	if meetingNo != "" {
		m, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, meetingNo)
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询会议失败")
		}
		if m == nil {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "会议不存在")
		}
		if m.Status != gormmodel.MeetingStatusActive {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "会议已结束")
		}
		callType = "conference"
	} else {
		var err error
		meetingNo, err = l.svcCtx.IdUtil.NextId("S", "live")
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, err, "生成会议号失败")
		}

		// 生成9位用户会议号，加分布式锁防并发生成重复 meeting_code
		codeLock := redis.NewRedisLock(l.svcCtx.Redis, redisMeetingCodeLockPrefix)
		codeLock.SetExpire(meetingCodeLockTTL)
		codeLockOk, err := codeLock.AcquireCtx(l.ctx)
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, err, "获取会议号生成锁失败")
		}
		if !codeLockOk {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_REPEAT, "会议号生成中，请稍后重试")
		}
		defer codeLock.Release()

		meetingCode, err := tool.RandomDigits(9)
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, err, "生成会议号失败")
		}
		for i := 0; i < 3; i++ {
			exists, err := l.svcCtx.MeetingRepo.IsMeetingCodeExists(l.ctx, meetingCode)
			if err != nil {
				return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询会议号失败")
			}
			if !exists {
				break
			}
			if i == 2 {
				return nil, tool.NewErrorByPbCode(extproto.Code__1_03_CACHE, "会议号生成冲突，请重试")
			}
			meetingCode, _ = tool.RandomDigits(9)
		}

		title := fmt.Sprintf("电话通话-%s", calleeNumber)
		now := carbonx.NowStartOfSecond().StdTime()
		room, err := l.svcCtx.LiveKit.Room().CreateRoom(l.ctx, &livekit.CreateRoomRequest{
			Name:             meetingNo,
			EmptyTimeout:     600,
			DepartureTimeout: 120,
			MaxParticipants:  10,
		})
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "创建房间失败")
		}
		meeting := &gormmodel.LiveMeeting{
			MeetingNo:        meetingNo,
			MeetingCode:      meetingCode,
			Title:            title,
			Status:           gormmodel.MeetingStatusActive,
			StartTime:        now,
			EmptyTimeout:     600,
			DepartureTimeout: 120,
			MaxParticipants:  10,
			RoomSid:          room.Sid,
		}
		if err := l.svcCtx.MeetingRepo.CreateMeeting(l.ctx, meeting); err != nil {
			_, _ = l.svcCtx.LiveKit.Room().DeleteRoom(l.ctx, &livekit.DeleteRoomRequest{Room: meetingNo})
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "会议落库失败")
		}
		l.Logger.Infof("SIP dial auto-created meeting: %s title=%q code=%s", meetingNo, title, meetingCode)
	}

	// 2. 查询供应商配置（必填）
	providerCode := strings.TrimSpace(in.GetProviderCode())
	if providerCode == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "供应商编码不能为空")
	}
	provider, err := l.svcCtx.MeetingRepo.GetSipProviderByCode(l.ctx, providerCode)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询供应商失败")
	}
	if provider == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "未找到可用的 SIP 供应商")
	}

	// 3. 从供应商配置解析号码池
	var numbers []string
	_ = json.Unmarshal([]byte(provider.Numbers), &numbers)

	// 4. 选择/创建 trunk（按供应商地址复用）
	trunkID := ""
	listRes, err := l.svcCtx.LiveKit.SIP().ListSIPOutboundTrunk(l.ctx, &livekit.ListSIPOutboundTrunkRequest{})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "查询 trunk 失败")
	}
	if len(listRes.Items) > 0 {
		trunkID = listRes.Items[0].SipTrunkId
	} else {
		res, err := l.svcCtx.LiveKit.SIP().CreateSIPOutboundTrunk(l.ctx, &livekit.CreateSIPOutboundTrunkRequest{
			Trunk: &livekit.SIPOutboundTrunkInfo{
				Name:         provider.Name,
				Address:      provider.Address,
				Numbers:      numbers,
				AuthUsername: provider.AuthUsername,
				AuthPassword: provider.AuthPassword,
			},
		})
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "创建 trunk 失败")
		}
		trunkID = res.SipTrunkId
	}

	// 5. 发起 SIP 外呼
	participantName := strings.TrimSpace(in.GetParticipantName())
	if participantName == "" {
		participantName = calleeNumber
	}
	participantIdentity := fmt.Sprintf("sip-%s", calleeNumber)

	participant, err := l.svcCtx.LiveKit.SIP().CreateSIPParticipant(l.ctx, &livekit.CreateSIPParticipantRequest{
		SipTrunkId:          trunkID,
		SipCallTo:           calleeNumber,
		RoomName:            meetingNo,
		ParticipantIdentity: participantIdentity,
		ParticipantName:     participantName,
		WaitUntilAnswered:   false,
	})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "发起外呼失败")
	}

	l.Logger.Infof("SIP dial: callee=%s meeting=%s callID=%s type=%s provider=%s", calleeNumber, meetingNo, participant.SipCallId, callType, provider.Code)

	// 6. 构造返回
	meetingModel, _ := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, meetingNo)
	return &live.DialSipRes{
		Meeting:   toMeetingInfo(meetingModel),
		SipCallId: participant.SipCallId,
	}, nil
}
