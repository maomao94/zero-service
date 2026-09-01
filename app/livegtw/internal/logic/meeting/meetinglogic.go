package meeting

import (
	"context"
	"encoding/base64"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// ===== 类型转换（types ↔ live proto）=====

func toMeetingInfo(m *live.MeetingInfo) types.MeetingInfo {
	return types.MeetingInfo{
		MeetingNo:  m.GetMeetingNo(),
		Title:      m.GetTitle(),
		Status:     m.GetStatus(),
		CreateUser: m.GetCreateUser(),
		UpdateUser: m.GetUpdateUser(),
		DeptCode:   m.GetDeptCode(),
		StartTime:  m.GetStartTime(),
		EndTime:    m.GetEndTime(),
		CreateTime: m.GetCreateTime(),
	}
}

func toParticipantInfo(p *live.ParticipantInfo) types.ParticipantInfo {
	return types.ParticipantInfo{
		Identity: p.GetIdentity(),
		Name:     p.GetName(),
		Status:   p.GetStatus(),
		JoinTime: p.GetJoinTime(),
		LeftTime: p.GetLeftTime(),
	}
}

// ===== 创建会议 =====

type CreateMeetingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMeetingLogic {
	return &CreateMeetingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 创建会议
func (l *CreateMeetingLogic) CreateMeeting(req *types.CreateMeetingReq) (resp *types.CreateMeetingRes, err error) {
	r, err := l.svcCtx.LiveRpcCli.CreateMeeting(l.ctx, &live.CreateMeetingReq{Title: req.Title})
	if err != nil {
		return nil, err
	}
	return &types.CreateMeetingRes{Meeting: toMeetingInfo(r.GetMeeting())}, nil
}

// ===== 加入会议 =====

type JoinMeetingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewJoinMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JoinMeetingLogic {
	return &JoinMeetingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 加入会议
func (l *JoinMeetingLogic) JoinMeeting(req *types.JoinMeetingReq) (resp *types.JoinMeetingRes, err error) {
	r, err := l.svcCtx.LiveRpcCli.JoinMeeting(l.ctx, &live.JoinMeetingReq{
		MeetingNo: req.MeetingNo,
		Identity:  req.Identity,
		Name:      req.Name,
	})
	if err != nil {
		return nil, err
	}
	return &types.JoinMeetingRes{
		Token:   r.GetToken(),
		WsUrl:   r.GetWsUrl(),
		Meeting: toMeetingInfo(r.GetMeeting()),
	}, nil
}

// ===== 查询会议详情 =====

type GetMeetingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMeetingLogic {
	return &GetMeetingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 查询会议详情
func (l *GetMeetingLogic) GetMeeting(req *types.GetMeetingReq) (resp *types.GetMeetingRes, err error) {
	r, err := l.svcCtx.LiveRpcCli.GetMeeting(l.ctx, &live.GetMeetingReq{MeetingNo: req.MeetingNo})
	if err != nil {
		return nil, err
	}
	return &types.GetMeetingRes{Meeting: toMeetingInfo(r.GetMeeting())}, nil
}

// ===== 分页查询会议列表 =====

type ListMeetingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMeetingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMeetingsLogic {
	return &ListMeetingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 分页查询会议列表
func (l *ListMeetingsLogic) ListMeetings(req *types.ListMeetingsReq) (resp *types.ListMeetingsRes, err error) {
	r, err := l.svcCtx.LiveRpcCli.ListMeetings(l.ctx, &live.ListMeetingsReq{
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	resp = &types.ListMeetingsRes{Total: r.GetTotal()}
	for _, m := range r.GetMeetings() {
		resp.Meetings = append(resp.Meetings, toMeetingInfo(m))
	}
	return resp, nil
}

// ===== 结束会议 =====

type EndMeetingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEndMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EndMeetingLogic {
	return &EndMeetingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 结束会议
func (l *EndMeetingLogic) EndMeeting(req *types.EndMeetingReq) error {
	_, err := l.svcCtx.LiveRpcCli.EndMeeting(l.ctx, &live.EndMeetingReq{MeetingNo: req.MeetingNo})
	return err
}

// ===== 踢人 =====

type KickParticipantLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewKickParticipantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickParticipantLogic {
	return &KickParticipantLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 踢人
func (l *KickParticipantLogic) KickParticipant(req *types.KickParticipantReq) error {
	_, err := l.svcCtx.LiveRpcCli.KickParticipant(l.ctx, &live.KickParticipantReq{
		MeetingNo: req.MeetingNo,
		Identity:  req.Identity,
	})
	return err
}

// ===== 静音/取消静音 =====

type MuteParticipantLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMuteParticipantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MuteParticipantLogic {
	return &MuteParticipantLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 静音/取消静音
func (l *MuteParticipantLogic) MuteParticipant(req *types.MuteParticipantReq) error {
	_, err := l.svcCtx.LiveRpcCli.MuteParticipant(l.ctx, &live.MuteParticipantReq{
		MeetingNo: req.MeetingNo,
		Identity:  req.Identity,
		Muted:     req.Muted,
		Kind:      req.Kind,
	})
	return err
}

// ===== 查询会议参与者 =====

type ListParticipantsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListParticipantsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListParticipantsLogic {
	return &ListParticipantsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 查询会议参与者
func (l *ListParticipantsLogic) ListParticipants(req *types.ListParticipantsReq) (resp *types.ListParticipantsRes, err error) {
	r, err := l.svcCtx.LiveRpcCli.ListParticipants(l.ctx, &live.ListParticipantsReq{MeetingNo: req.MeetingNo})
	if err != nil {
		return nil, err
	}
	resp = &types.ListParticipantsRes{}
	for _, p := range r.GetParticipants() {
		resp.Participants = append(resp.Participants, toParticipantInfo(p))
	}
	return resp, nil
}

// ===== 向会议发送 Data =====

type SendMeetingDataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendMeetingDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendMeetingDataLogic {
	return &SendMeetingDataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 向会议发送 Data（广播/定向）
func (l *SendMeetingDataLogic) SendMeetingData(req *types.SendMeetingDataReq) error {
	payload, err := base64.StdEncoding.DecodeString(req.Payload)
	if err != nil {
		return err
	}
	_, err = l.svcCtx.LiveRpcCli.SendMeetingData(l.ctx, &live.SendMeetingDataReq{
		MeetingNo:    req.MeetingNo,
		Topic:        req.Topic,
		Payload:      payload,
		Destinations: req.Destinations,
	})
	return err
}

// ===== 服务端对参与者执行 RPC =====

type PerformMeetingRpcLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPerformMeetingRpcLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PerformMeetingRpcLogic {
	return &PerformMeetingRpcLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 服务端对参与者执行 RPC
func (l *PerformMeetingRpcLogic) PerformMeetingRpc(req *types.PerformMeetingRpcReq) (resp *types.PerformMeetingRpcRes, err error) {
	r, err := l.svcCtx.LiveRpcCli.PerformMeetingRpc(l.ctx, &live.PerformMeetingRpcReq{
		MeetingNo:         req.MeetingNo,
		Identity:          req.Identity,
		Method:            req.Method,
		Payload:           req.Payload,
		ResponseTimeoutMs: req.ResponseTimeoutMs,
	})
	if err != nil {
		return nil, err
	}
	return &types.PerformMeetingRpcRes{Response: r.GetResponse()}, nil
}
