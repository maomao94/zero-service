package logic

import (
	"context"
	"fmt"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/jinzhu/copier"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/zeromicro/go-zero/core/logx"
	"strings"
	"zero-service/alarmx"
	"zero-service/zeroalarm/internal/svc"
	"zero-service/zeroalarm/zeroalarm"
)

type AlarmLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAlarmLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlarmLogic {
	return &AlarmLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AlarmLogic) Alarm(in *zeroalarm.AlarmReq) (*zeroalarm.AlarmRes, error) {
	in.UserId = append(in.UserId, l.svcCtx.Config.Alarmx.UserId...)
	result := stream.FromSlice(in.UserId).Distinct().ToSlice()
	formatChatName := in.ChatName + fmt.Sprintf("[%s]", l.svcCtx.Config.Mode)
	// 告警
	chatId, err := l.svcCtx.AlarmX.AlarmChat(l.ctx, l.svcCtx.Config.Name, formatChatName, in.Description, result)
	if err != nil {
		return nil, err
	}
	// 发送告警通知
	var info alarmx.AlarmInfo
	_ = copier.Copy(&info, in)
	err = l.svcCtx.AlarmX.SendAlertMessage(l.ctx, l.svcCtx.Config.Alarmx.Path, chatId, &info)
	if err != nil {
		return nil, err
	}
	//  注册事件回调
	//eventHandler := dispatcher.NewEventDispatcher(l.svcCtx.Config.Alarm.VerificationToken, l.svcCtx.Config.Alarm.EncryptKey)
	//eventHandler.OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	//	return DoP2ImMessageReceiveV1(l.svcCtx, event)
	//})
	// 注册卡片回调
	//cardHandler := larkcard.NewCardActionHandler(l.svcCtx.Config.Alarm.VerificationToken, l.svcCtx.Config.Alarm.EncryptKey, func(ctx context.Context, action *larkcard.CardAction) (interface{}, error) {
	//	return DoInteractiveCard(l.svcCtx, action)
	//})

	//http.HandleFunc("/event", httpserverext.NewEventHandlerFunc(eventHandler,
	//	larkevent.WithLogLevel(larkcore.LogLevelDebug)))
	//http.HandleFunc("/card", httpserverext.NewCardActionHandlerFunc(cardHandler,
	//	larkevent.WithLogLevel(larkcore.LogLevelDebug)))
	//err = http.ListenAndServe(":7777", nil)
	return &zeroalarm.AlarmRes{}, nil
}

// 上传图片
//func uploadImage() (string, error) {
//	image, err := os.Open("./quick_start/robot/alert.png")
//	if err != nil {
//		return "", err
//	}
//	req := larkim.NewCreateImageReqBuilder().
//		Body(larkim.NewCreateImageReqBodyBuilder().
//			ImageType("message").
//			Image(image).
//			Build()).
//		Build()
//
//	resp, err := oapi_sdk_go_demo.Client.Im.Image.Create(context.Background(), req)
//	if err != nil {
//		return "", err
//	}
//	if !resp.Success() {
//		return "", resp.CodeError
//	}
//
//	return *resp.Data.ImageKey, nil
//}

// DoP2ImMessageReceiveV1 处理消息回调
func DoP2ImMessageReceiveV1(svcCtx *svc.ServiceContext, data *larkim.P2MessageReceiveV1) error {
	msg := data.Event.Message
	if strings.Contains(*msg.Content, "/solve") {
		req := larkim.NewCreateMessageReqBuilder().
			ReceiveIdType("chat_id").
			Body(larkim.NewCreateMessageReqBodyBuilder().
				ReceiveId(*msg.ChatId).
				MsgType("text").
				Content("{\"text\":\"问题已解决，辛苦了!\"}").
				Build()).
			Build()

		resp, err := svcCtx.AlarmX.ImMessageCreate(context.Background(), req)
		if err != nil {
			return err
		}
		if !resp.Success() {
			return resp.CodeError
		}
		// 获取会话信息
		chatInfo, err := getChatInfo(svcCtx, *msg.ChatId)
		if err != nil {
			return err
		}
		name := *chatInfo.Name
		if strings.HasPrefix(name, "[跟进中]") {
			name = "[已解决]" + name[len("[跟进中]"):]
		} else if !strings.HasPrefix(name, "[已解决]") {
			name = "[已解决]" + name
		}
		// 修改会话名称
		err = updateChatName(svcCtx, *msg.ChatId, name)
		if err != nil {
			return err
		}
	}
	return nil
}

// 获取会话信息
func getChatInfo(svcCtx *svc.ServiceContext, chatId string) (*larkim.GetChatRespData, error) {
	req := larkim.NewGetChatReqBuilder().
		ChatId(chatId).
		Build()
	resp, err := svcCtx.AlarmX.ImChatGet(context.Background(), req)
	if err != nil {
		return nil, err
	}
	if !resp.Success() {
		return nil, resp.CodeError
	}
	return resp.Data, nil
}

// 更新会话名称
func updateChatName(svcCtx *svc.ServiceContext, chatId string, chatName string) error {
	req := larkim.NewUpdateChatReqBuilder().
		ChatId(chatId).
		Body(larkim.NewUpdateChatReqBodyBuilder().
			Name(chatName).
			Build()).
		Build()
	resp, err := svcCtx.AlarmX.ImChatUpdate(context.Background(), req)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return resp.CodeError
	}
	return nil
}

// DoInteractiveCard 处理卡片回调
//func DoInteractiveCard(svcCtx *svc.ServiceContext, data *larkcard.CardAction) (interface{}, error) {
//	if data.Action.Value["key"] == "follow" {
//		chatInfo, err := getChatInfo(svcCtx, data.OpenChatId)
//		if err != nil {
//			return nil, err
//		}
//		name := *chatInfo.Name
//		if !strings.HasPrefix(name, "[跟进中]") && !strings.HasPrefix(name, "[已解决]") {
//			name = "[跟进中] " + name
//		}
//		// 修改会话名称
//		err = updateChatName(svcCtx, data.OpenChatId, name)
//		if err != nil {
//			return nil, err
//		}
//
//		return buildCard(svcCtx, nil, "跟进中")
//	}
//
//	return nil, nil
//}
