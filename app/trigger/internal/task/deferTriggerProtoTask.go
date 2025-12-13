package task

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"
	"zero-service/app/trigger/internal/svc"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/asynqx"
	"zero-service/common/ctxdata"
	"zero-service/common/tool"

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/timex"
	"github.com/zeromicro/go-zero/zrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
)

type DeferTriggerProtoTaskHandler struct {
	svcCtx  *svc.ServiceContext
	metrics *stat.Metrics
}

func NewDeferTriggerProtoTask(svcCtx *svc.ServiceContext) *DeferTriggerProtoTaskHandler {
	return &DeferTriggerProtoTaskHandler{
		svcCtx:  svcCtx,
		metrics: stat.NewMetrics("proto-task"),
	}
}

func (l *DeferTriggerProtoTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	startTime := timex.Now()
	defer l.metrics.Add(stat.Task{
		Duration: timex.Since(startTime),
	})
	var msg ctxdata.ProtoMsgBody
	if err := json.Unmarshal([]byte(t.Payload()), &msg); err != nil {
		return err
	} else {
		ctx = otel.GetTextMapPropagator().Extract(ctx, msg.Carrier)
		ctx, span := asynqx.StartAsynqConsumerSpan(ctx, t.Type())
		defer span.End()
		grpcServer := tool.MayReplaceLocalhost(msg.GrpcServer)
		clientConf := zrpc.RpcClientConf{}
		conf.FillDefault(&clientConf)
		clientConf.Target = grpcServer
		clientConf.NonBlock = true
		clientConf.Timeout = 60000
		v, ok := l.svcCtx.ConnMap.Get(grpcServer)
		if !ok {
			conn, err := zrpc.NewClient(clientConf,
				zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
				zrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.ForceCodec(rawCodec{}))))
			if err == nil {
				l.svcCtx.ConnMap.Set(grpcServer, conn)
				v = conn
				logx.WithContext(ctx).Debugf("grpc client inited for %s", grpcServer)
			} else {
				logx.WithContext(ctx).Errorf("grpc client init fail for %s, %v", grpcServer, err)
			}
		}
		if v == nil {
			t.ResultWriter().Write([]byte("fail,conn is nil"))
			return errors.New("trigger fail")
		}
		cli := v.(*zrpc.RpcClient)
		if msg.RequestTimeout == 0 {
			msg.RequestTimeout = clientConf.Timeout
		}
		var cancel context.CancelFunc
		if msg.RequestTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(msg.RequestTimeout)*time.Millisecond)
			defer cancel()
		}
		var respBytes []byte
		zrpc.DontLogClientContentForMethod(msg.Method)
		err = cli.Conn().Invoke(ctx, msg.Method, msg.Payload, &respBytes)
		serverName := path.Join(cli.Conn().Target(), msg.Method)
		duration := timex.Since(startTime)
		logx.WithContext(ctx).WithDuration(duration).Infof("rpc invoke - %s", serverName)
		if err != nil {
			t.ResultWriter().Write([]byte("fail,rpcInvokeError: " + err.Error()))
			return err
		}
	}
	return nil
}

type rawCodec struct{}

func (cb rawCodec) Marshal(v any) ([]byte, error) {
	return tool.ToProtoBytes(v)
}

func (cb rawCodec) Unmarshal(data []byte, v any) error {
	ba, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("please pass in *[]byte")
	}
	*ba = append(*ba, data...)
	return nil
}

func (cb rawCodec) Name() string { return "proto_raw" }
