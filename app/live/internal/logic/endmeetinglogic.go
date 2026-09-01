package logic

import (
	"context"
	"time"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/authctx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type EndMeetingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEndMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EndMeetingLogic {
	return &EndMeetingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 结束会议（删除 LiveKit 房间 + 状态流转，Redis lock 防重入）
func (l *EndMeetingLogic) EndMeeting(in *live.EndMeetingReq) (*live.EndMeetingRes, error) {
	if err := requireMeetingNo(in.MeetingNo); err != nil {
		return nil, err
	}
	meeting, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, in.MeetingNo)
	if err != nil {
		return nil, meetingErr(err)
	}
	// 已结束：幂等成功返回
	if meeting.Status == gormmodel.MeetingStatusEnded {
		return &live.EndMeetingRes{}, nil
	}
	// 分布式锁防并发结束（go-zero RedisLock，Lua 原子 + TTL 自动释放；
// 写法与 oryxserver relay Store.Lock 一致）
	lock := redis.NewRedisLock(l.svcCtx.Redis, redisEndLockPrefix+in.MeetingNo+":end")
	lock.SetExpire(endLockTTL)
	ok, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, err, "获取会议锁失败")
	}
	if !ok {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_REPEAT, "会议正在被结束，请稍后重试")
	}
	defer lock.Release()

	// 二次检查：锁等待期间可能已被其他请求结束
	meeting, err = l.svcCtx.MeetingRepo.GetMeeting(l.ctx, in.MeetingNo)
	if err != nil {
		return nil, meetingErr(err)
	}
	if meeting.Status == gormmodel.MeetingStatusEnded {
		return &live.EndMeetingRes{}, nil
	}

	// 删除 LiveKit 房间（结束会议统一入口），断开全部参与者
	if _, err := l.svcCtx.LiveKit.Room().DeleteRoom(l.ctx, &livekit.DeleteRoomRequest{Room: in.MeetingNo}); err != nil {
		l.Logger.Errorf("delete room failed: meeting=%s err=%v", in.MeetingNo, err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "删除房间失败")
	}

	now := time.Now()
	// 结束操作人/机构取自 gRPC metadata（webhook room_finished 场景不传）
	operator := authctx.GetUserId(l.ctx)
	deptCode := authctx.GetDeptCode(l.ctx)
	if updated, err := l.svcCtx.MeetingRepo.UpdateMeetingEnded(l.ctx, in.MeetingNo, now, operator, deptCode); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "更新会议状态失败")
	} else if !updated {
		l.Logger.Infof("meeting already ended by other path: %s", in.MeetingNo)
		return &live.EndMeetingRes{}, nil
	}

	// 批量标记在会参与者为已离开
	participants, err := l.svcCtx.MeetingRepo.ListParticipants(l.ctx, in.MeetingNo)
	if err == nil {
		for _, p := range participants {
			if p.Status == gormmodel.ParticipantStatusJoined {
				_ = l.svcCtx.MeetingRepo.MarkParticipantLeft(l.ctx, in.MeetingNo, p.Identity, now)
			}
		}
	}

	l.Logger.Infof("meeting ended: %s", in.MeetingNo)
	return &live.EndMeetingRes{}, nil
}