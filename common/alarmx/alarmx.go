package alarmx

import (
	"context"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpc"
	"net/http"
	"os"
	"strings"
)

type AlarmInfo struct {
	Title    string   // 报警标题
	Project  string   // 项目名称
	DateTime string   // 2019-01-01 00:00:00
	AlarmId  string   // 唯一报警 id
	Content  string   // 报警内容
	Error    string   // 错误信息
	UserId   []string // 报警人 userId
	Ip       string   // 报警 ip
}

type AlarmX struct {
	LarkClient  *lark.Client
	RedisClient *redis.Redis
}

type AlarmxHttpClient struct {
	httpc.Service
}

func NewAlarmxHttpClient(httpc httpc.Service) larkcore.HttpClient {
	return &AlarmxHttpClient{Service: httpc}
}

func (cli *AlarmxHttpClient) Do(r *http.Request) (*http.Response, error) {
	return cli.Service.DoRequest(r)
}

func NewAlarmX(larkClient *lark.Client, redisClient *redis.Redis) *AlarmX {
	return &AlarmX{
		LarkClient:  larkClient,
		RedisClient: redisClient,
	}
}

func (a *AlarmX) AlarmChat(ctx context.Context, appName, chatName, description string, userId []string) (string, error) {
	chatId, err := a.RedisClient.GetCtx(ctx, appName+":alarm:"+chatName)
	if err != nil {
		return "", err
	}
	if len(chatId) == 0 {
		// 创建告警群并拉人入群
		chatId, err = a.CreateAlertChat(ctx, chatName, description, userId)
		if err != nil {
			return "", err
		}
		err = a.RedisClient.SetexCtx(ctx, appName+":alarm:"+chatName, chatId, 60*60*24*7)
		if err != nil {
			return "", err
		}
	} else {
		// 拉人入群
		err = a.UpdateAlertChat(ctx, chatId, userId)
		if err != nil {
			return "", err
		}
	}
	return chatId, nil
}

func (a *AlarmX) CreateAlertChat(ctx context.Context, chatName, description string, userId []string) (string, error) {
	req := larkim.NewCreateChatReqBuilder().
		UserIdType("user_id").
		Body(larkim.NewCreateChatReqBodyBuilder().
			Name(chatName).
			Description(description).
			UserIdList(userId).
			Build()).
		Build()
	resp, err := a.ImChatCreate(ctx, req)
	if err != nil {
		logx.WithContext(ctx).Errorf("创建告警群失败:%+v", err)
		return "", err
	}
	if !resp.Success() {
		logx.WithContext(ctx).Errorf("创建告警群失败:%+v", resp.CodeError)
		return "", resp.CodeError
	}
	return *resp.Data.ChatId, nil
}

func (a *AlarmX) UpdateAlertChat(ctx context.Context, chatId string, userId []string) error {
	req := larkim.NewCreateChatMembersReqBuilder().
		MemberIdType("user_id").
		ChatId(chatId).
		Body(larkim.NewCreateChatMembersReqBodyBuilder().
			IdList(userId).
			Build()).
		Build()
	resp, err := a.ImChatMembersCreate(ctx, req)
	if err != nil {
		logx.WithContext(ctx).Errorf("拉人入群失败:%+v", err)
		return err
	}
	if !resp.Success() {
		logx.WithContext(ctx).Errorf("拉人入群失败:%+v", resp.CodeError)
		return resp.CodeError
	}
	return nil
}

func (a *AlarmX) SendAlertMessage(ctx context.Context, path string, chatId string, in *AlarmInfo) error {
	content, err := buildCard(path, in, "跟进处理")
	if err != nil {
		return err
	}
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType("chat_id").
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatId).
			MsgType("interactive").
			Content(content).
			Build()).
		Build()
	resp, err := a.ImMessageCreate(ctx, req)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return resp.CodeError
	}
	return nil
}

func (a *AlarmX) ImChatCreate(ctx context.Context, req *larkim.CreateChatReq, options ...larkcore.RequestOptionFunc) (*larkim.CreateChatResp, error) {
	return a.LarkClient.Im.Chat.Create(ctx, req, options...)
}

func (a *AlarmX) ImChatMembersCreate(ctx context.Context, req *larkim.CreateChatMembersReq, options ...larkcore.RequestOptionFunc) (*larkim.CreateChatMembersResp, error) {
	return a.LarkClient.Im.ChatMembers.Create(ctx, req, options...)
}

func (a *AlarmX) ImMessageCreate(ctx context.Context, req *larkim.CreateMessageReq, options ...larkcore.RequestOptionFunc) (*larkim.CreateMessageResp, error) {
	return a.LarkClient.Im.Message.Create(ctx, req, options...)
}

func (a *AlarmX) ImChatUpdate(ctx context.Context, req *larkim.UpdateChatReq, options ...larkcore.RequestOptionFunc) (*larkim.UpdateChatResp, error) {
	return a.LarkClient.Im.Chat.Update(ctx, req, options...)
}

func (a *AlarmX) ImChatGet(ctx context.Context, req *larkim.GetChatReq, options ...larkcore.RequestOptionFunc) (*larkim.GetChatResp, error) {
	return a.LarkClient.Im.Chat.Get(ctx, req, options...)
}

// 构建卡片
func buildCard(path string, in *AlarmInfo, buttonName string) (string, error) {
	//imageKey, err := uploadImage()
	//if err != nil {
	//	return "", err
	//}
	bs, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	card := string(bs)
	//card = strings.Replace(card, "${image_key}", imageKey, -1)
	if in != nil {
		card = strings.Replace(card, "${title}", in.Title, -1)
		card = strings.Replace(card, "${project}", in.Project, -1)
		card = strings.Replace(card, "${dateTime}", in.DateTime, -1)
		card = strings.Replace(card, "${alarmId}", in.AlarmId, -1)
		card = strings.Replace(card, "${content}", escape(in.Content), -1)
		card = strings.Replace(card, "${error}", escape(in.Error), -1)
		card = strings.Replace(card, "${ip}", in.Ip, -1)
		card = strings.Replace(card, "${button_name}", buttonName, -1)
	}
	return card, nil
}

func escape(input string) string {
	var b strings.Builder
	for _, ch := range input {
		switch ch {
		case '\x00':
			b.WriteString(`\x00`)
		case '\r':
			b.WriteString(`\r`)
		case '\n':
			b.WriteString(`\n`)
		case '\\':
			b.WriteString(`\\`)
		case '\'':
			b.WriteString(`\'`)
		case '"':
			b.WriteString(`\"`)
		case '\x1a':
			b.WriteString(`\x1a`)
		default:
			b.WriteRune(ch)
		}
	}
	return b.String()
}
