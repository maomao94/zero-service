package logic

import (
	"context"
	"database/sql"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/models"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/payment/order/request"
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/random"
	"github.com/songzhibin97/gkit/errors"
	"strings"
	"time"
	"zero-service/common"
	"zero-service/model"

	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/zerorpc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WxPayJsApiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWxPayJsApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WxPayJsApiLogic {
	return &WxPayJsApiLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// JSAPI支付
func (l *WxPayJsApiLogic) WxPayJsApi(in *zerorpc.WxPayJsApiReq) (*zerorpc.WxPayJsApiRes, error) {
	uid, _ := random.UUIdV4()
	txnId := strings.Replace(uid, "-", "", -1)
	params := &request.RequestJSAPIPrepay{
		Amount: &request.JSAPIAmount{
			Total:    int(in.RealAmt),
			Currency: "CNY",
		},
		//Attach:      "自定义数据说明",
		Description: in.Body,
		OutTradeNo:  txnId, // 这里是商户订单号，不能重复提交给微信
		Payer: &request.JSAPIPayer{
			OpenID: in.OpenId, // 用户的openid， 记得也是动态的。
		},
	}
	// 下单
	response, err := l.svcCtx.WxPayCli.Order.JSAPITransaction(l.ctx, params)
	if err != nil {
		return nil, err
	}
	if len(response.ResponseBase.Code) != 0 {
		l.Errorf("JSAPI支付 %v", response.Code)
		return nil, errors.BadRequest("9999", "JSAPI支付失败")
	}
	if len(response.PrepayID) == 0 {
		l.Errorf("JSAPI支付 %v", response.Code)
		return nil, errors.BadRequest("9999", "JSAPI支付失败")
	}
	// 因为PrepayID签名方式都一样，所以这个和App是一样的。
	payConf, err := l.svcCtx.WxPayCli.JSSDK.BridgeConfig(response.PrepayID, true)
	if err != nil {
		return nil, err
	}
	order := &model.OrderTxn{
		TxnId:          txnId,
		OriTxnId:       "",
		TxnTime:        time.Now(),
		TxnDate:        time.Now(),
		MchId:          in.MchId,
		MchOrderNo:     in.MchOrderNo,
		PayType:        common.PayType_Wxpay,
		TxnType:        common.TxnType_Consume,
		TxnChannel:     models.WX_TRADE_STATE_,
		TxnAmt:         in.TxnAmt,
		RealAmt:        in.RealAmt,
		Result:         common.ResultProcessing,
		Body:           in.Body,
		Extra:          "",
		UserId:         0,
		ChannelUser:    in.OpenId,
		ChannelPayTime: sql.NullTime{},
		ExpireTime:     int64(common.ExpireTime),
	}
	_, err = l.svcCtx.OrderTxnModel.Insert(l.ctx, nil, order)
	if err != nil {
		return nil, err
	}
	return &zerorpc.WxPayJsApiRes{
		TxnId:    txnId,
		PayConf:  convertor.ToString(payConf),
		PrepayId: response.PrepayID,
	}, nil
}
