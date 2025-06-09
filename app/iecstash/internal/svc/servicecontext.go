package svc

import (
	"context"
	"github.com/tidwall/gjson"
	"github.com/zeromicro/go-zero/core/executors"
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/app/iecstash/internal/config"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/facade/iecstream/iecstream"
)

type ServiceContext struct {
	Config          config.Config
	IecStreamRpcCli iecstream.IecStreamRpcClient
	AsduPusher      *AsduPusher
}

func NewServiceContext(c config.Config) *ServiceContext {
	iecStreamRpcCli := iecstream.NewIecStreamRpcClient(zrpc.MustNewClient(c.IecStreamRpcConf,
		zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor)).Conn())
	return &ServiceContext{
		Config:          c,
		IecStreamRpcCli: iecStreamRpcCli,
		AsduPusher:      NewAsduPusher(iecStreamRpcCli),
	}
}

type AsduPusher struct {
	inserter        *executors.ChunkExecutor
	IecStreamRpcCli iecstream.IecStreamRpcClient
}

func (w *AsduPusher) Write(val string) error {
	return w.inserter.Add(val, len(val))
}

func (w *AsduPusher) execute(vals []interface{}) {
	msgBodyList := make([]*iecstream.MsgBody, 0)
	for _, val := range vals {
		s := val.(string)
		result := gjson.Parse(s)
		bodyRaw := result.Get("body").Raw
		typeId := result.Get("typeId").Int()
		msgBody := &iecstream.MsgBody{
			Host:        result.Get("host").String(),
			Port:        int32(result.Get("port").Int()),
			Asdu:        result.Get("asdu").String(),
			TypeId:      int32(typeId),
			Coa:         uint32(result.Get("coa").Int()),
			BodyRaw:     bodyRaw,
			Time:        result.Get("time").String(),
			MetaDataRaw: result.Get("metaData").Raw,
		}
		msgBodyList = append(msgBodyList, msgBody)
	}
	if len(msgBodyList) == 0 {
		return
	}
	w.IecStreamRpcCli.PushChunkAsdu(context.Background(), &iecstream.PushChunkAsduReq{
		MsgBody: msgBodyList,
	})
}

func NewAsduPusher(iecStreamRpcCli iecstream.IecStreamRpcClient) *AsduPusher {
	writer := AsduPusher{}
	writer.inserter = executors.NewChunkExecutor(writer.execute, executors.WithChunkBytes(1048))
	writer.IecStreamRpcCli = iecStreamRpcCli
	return &writer
}
