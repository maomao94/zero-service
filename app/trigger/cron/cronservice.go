package cron

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"
	"zero-service/app/trigger/internal/svc"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type CronService struct {
	cancelChan chan struct{}
	svcCtx     *svc.ServiceContext
}

func NewCronService(svcCtx *svc.ServiceContext) *CronService {
	return &CronService{
		svcCtx: svcCtx,
	}
}

func (s *CronService) Start() {
	if s.cancelChan != nil {
		return
	}
	// Create cancellation channel for proper goroutine termination
	s.cancelChan = make(chan struct{})
	log.Print("Starting cron service \n")

	go s.scanLoop()
}

func (s *CronService) Stop() {
	if s.cancelChan != nil {
		// Close the cancellation channel to signal goroutine termination
		close(s.cancelChan)
		s.cancelChan = nil
		log.Print("Stopping cron service \n")
	}
}

func (s *CronService) scanLoop() {
	for {
		select {
		case <-s.cancelChan:
			return
		default:
		}

		itemsProcessed := s.ScanPlanExecItem()
		var sleepDuration time.Duration
		if itemsProcessed {
			sleepDuration = 10 * time.Millisecond
		} else {
			sleepDuration = time.Duration(1000+rand.Intn(1000)) * time.Millisecond
		}

		timer := time.NewTimer(sleepDuration)
		select {
		case <-s.cancelChan:
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
			// Timer fired, automatically stopped
		}
		// No need to stop timer here since it already fired
	}
}

func (s *CronService) ScanPlanExecItem() bool {
	ctx := context.Background()
	execItem, err := s.svcCtx.PlanExecItemModel.LockTriggerItem(ctx, 5*60*time.Second)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return false
		}
		logx.Errorf("Error locking plan exec item: %v", err)
		return false
	}
	if execItem == nil {
		return false
	}
	threading.GoSafe(func() {
		logx.Infof("Found plan exec item to trigger: id=%d, planId=%s, itemId=%s, nextTriggerTime=%v",
			execItem.Id,
			execItem.PlanId,
			execItem.ItemId,
			execItem.NextTriggerTime,
		)

		plan, _ := s.svcCtx.PlanModel.FindOneByPlanId(ctx, execItem.PlanId)
		s.ExecuteCallback(ctx, execItem, plan)
	})
	return true
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

func (s *CronService) ExecuteCallback(ctx context.Context, execItem *model.PlanExecItem, plan *model.Plan) {
	grpcServer := tool.MayReplaceLocalhost(execItem.ServiceAddr)
	logx.Infof("Executing callback for exec item %d with service: %s, planId: %s, itemId: %s",
		execItem.Id, grpcServer, execItem.PlanId, execItem.ItemId)

	if err := s.svcCtx.PlanExecItemModel.UpdateStatusToRunning(ctx, execItem.Id); err != nil {
		logx.Errorf("Error updating plan exec item %d to running: %v", execItem.Id, err)
		return
	}
	clientConf := zrpc.RpcClientConf{}
	conf.FillDefault(&clientConf)
	clientConf.Target = grpcServer
	clientConf.NonBlock = true
	clientConf.Timeout = 60000

	v, ok := s.svcCtx.ConnMap.Get(grpcServer)
	if !ok {
		conn, err := zrpc.NewClient(clientConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			zrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.ForceCodec(rawCodec{}))))
		if err != nil {
			logx.Errorf("Failed to create gRPC client for %s: %v", grpcServer, err)
			if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "fail", "Failed to create gRPC client: "+err.Error()); updateErr != nil {
				logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
			}
			return
		}
		s.svcCtx.ConnMap.Set(grpcServer, conn)
		v = conn
		logx.Infof("gRPC client inited for %s", grpcServer)
	}
	if v == nil {
		logx.Errorf("gRPC client is nil for %s", grpcServer)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "fail", "gRPC client is nil"); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}

	cli, ok := v.(*zrpc.RpcClient)
	if !ok {
		logx.Errorf("Invalid connection type in ConnMap for %s", grpcServer)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "fail", "Invalid connection type in ConnMap"); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}

	if execItem.RequestTimeout == 0 {
		execItem.RequestTimeout = clientConf.Timeout
	}

	var cancel context.CancelFunc
	if execItem.RequestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(execItem.RequestTimeout)*time.Millisecond)
		defer cancel()
	}

	req := &streamevent.HandlerPlanTaskEventReq{
		PlanId:   execItem.PlanId,
		PlanName: plan.PlanName,
		Type:     plan.Type,
		ItemId:   execItem.ItemId,
		ItemName: execItem.ItemName,
		Payload:  execItem.Payload,
	}
	var respBytes []byte
	in, err := tool.ToProtoBytes(req)
	if err != nil {
		logx.Errorf("Error marshaling request for exec item %d: %v", execItem.Id, err)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "fail", "Error marshaling request: "+err.Error()); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}
	err = cli.Conn().Invoke(ctx, execItem.Method, &in, &respBytes)
	if err != nil {
		logx.Errorf("gRPC call failed for exec item %d: %v", execItem.Id, err)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "fail", "gRPC call failed: "+err.Error()); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}
	res := &streamevent.HandlerPlanTaskEventRes{}
	err = proto.Unmarshal(respBytes, res)
	if err != nil {
		logx.Errorf("Error unmarshaling response for exec item %d: %v", execItem.Id, err)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "fail", "Error unmarshaling response: "+err.Error()); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}
	switch res.ExecResult {
	case 1: // Success
		logx.Infof("gRPC call succeeded for exec item %d", execItem.Id)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, "complete", res.Message); err != nil {
			logx.Errorf("Error updating plan exec item %d to completed: %v", execItem.Id, err)
		}
	case 2: // Failed
		logx.Infof("gRPC call returned failure for exec item %d: %s", execItem.Id, res.Message)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "fail", res.Message); err != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, err)
		}
	case 3: // Delay
		logx.Infof("gRPC call requested delay for exec item %d: %s", execItem.Id, res.Message)
		if res.DelayConfig != nil {
			var delayReason = res.DelayConfig.DelayReason
			if len(delayReason) == 0 {
				delayReason = res.Message
			}
			// Update with next trigger time from delay config
			if err := s.svcCtx.PlanExecItemModel.UpdateStatusToDelayed(ctx, execItem.Id, "delay", delayReason, res.DelayConfig.NextTriggerTime); err != nil {
				logx.Errorf("Error updating plan exec item %d to delayed: %v", execItem.Id, err)
			}
		}
	default:
		logx.Errorf("Unknown execResult %d for exec item %d", res.ExecResult, execItem.Id)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, "complete", res.Message); err != nil {
			logx.Errorf("Error updating plan exec item %d to completed: %v", execItem.Id, err)
		}
	}

	logx.Infof("Successfully executed callback for plan exec item: ID=%d", execItem.Id)
}
