package logic

import (
	"context"
	"encoding/json"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSipProviderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateSipProviderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSipProviderLogic {
	return &CreateSipProviderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateSipProviderLogic) CreateSipProvider(in *live.CreateSipProviderReq) (*live.CreateSipProviderRes, error) {
	code := strings.TrimSpace(in.GetCode())
	if code == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "供应商编码不能为空")
	}
	if strings.TrimSpace(in.GetName()) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "供应商名称不能为空")
	}
	if strings.TrimSpace(in.GetAddress()) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "SIP 服务器地址不能为空")
	}
	if len(in.GetNumbers()) == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "号码池不能为空")
	}

	// 检查编码唯一
	existing, err := l.svcCtx.MeetingRepo.GetSipProviderByCode(l.ctx, code)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询供应商失败")
	}
	if existing != nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_ALREADY_EXIST, "供应商编码已存在")
	}

	numbersJSON, _ := json.Marshal(in.GetNumbers())
	now := carbonx.NowStartOfSecond().StdTime()
	p := &gormmodel.LiveSipProvider{
		Code:         code,
		Name:         strings.TrimSpace(in.GetName()),
		Address:      strings.TrimSpace(in.GetAddress()),
		Numbers:      string(numbersJSON),
		AuthUsername: in.GetAuthUsername(),
		AuthPassword: in.GetAuthPassword(),
		Status:       gormmodel.SipProviderStatusEnabled,
	}
	p.CreateTime = now
	p.UpdateTime = now

	if err := l.svcCtx.MeetingRepo.CreateSipProvider(l.ctx, p); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "创建供应商失败")
	}

	l.Logger.Infof("SIP provider created: %s (%s)", code, p.Name)
	return &live.CreateSipProviderRes{Provider: toSipProviderInfo(p)}, nil
}
