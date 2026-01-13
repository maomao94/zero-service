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

	"github.com/dromara/carbon/v2"
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
		execItem, _ = s.svcCtx.PlanExecItemModel.FindOne(ctx, execItem.Id)
		plan, _ := s.svcCtx.PlanModel.FindOneByPlanId(ctx, execItem.PlanId)
		logx.Infof("Found plan exec item to trigger: id=%d, planPk=%d, planId=%s, planName=%s, itemId=%s, itemName=%s, pointId=%s, nextTriggerTime=%v",
			execItem.Id,
			execItem.PlanPk,
			execItem.PlanId,
			plan.PlanName,
			execItem.ItemId,
			execItem.ItemName,
			execItem.PointId,
			execItem.NextTriggerTime,
		)
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
	clientConf := zrpc.RpcClientConf{}
	conf.FillDefault(&clientConf)
	clientConf.Target = grpcServer
	clientConf.NonBlock = true
	clientConf.Timeout = 60000
	if execItem.RequestTimeout == 0 {
		execItem.RequestTimeout = clientConf.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(execItem.RequestTimeout)*time.Millisecond+6*time.Second)
	defer cancel()
	logx.Debugf("Executing callback for exec item %d with service: %s, planId: %s, itemId: %s",
		execItem.Id, grpcServer, execItem.PlanId, execItem.ItemId)

	if err := s.svcCtx.PlanExecItemModel.UpdateStatusToRunning(ctx, execItem.Id); err != nil {
		logx.Errorf("Error updating plan exec item %d to running: %v", execItem.Id, err)
		return
	}

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
		logx.Debugf("gRPC client inited for %s", grpcServer)
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

	req := &streamevent.HandlerPlanTaskEventReq{
		CreateTime:      carbon.CreateFromStdTime(execItem.CreateTime).ToDateTimeString(),
		UpdateTime:      carbon.CreateFromStdTime(execItem.UpdateTime).ToDateTimeString(),
		CreateUser:      execItem.CreateUser,
		UpdateUser:      execItem.UpdateUser,
		PlanId:          execItem.PlanId,
		PlanName:        plan.PlanName,
		Type:            plan.Type,
		GroupId:         plan.GroupId,
		Description:     plan.Description,
		StartTime:       carbon.NewCarbon(plan.StartTime).ToDateTimeString(),
		EndTime:         carbon.NewCarbon(plan.EndTime).ToDateTimeString(),
		PlanPk:          0,
		ItemId:          execItem.ItemId,
		ItemName:        execItem.ItemName,
		PointId:         execItem.PointId,
		Payload:         execItem.Payload,
		PlanTriggerTime: carbon.NewCarbon(execItem.PlanTriggerTime).ToDateTimeString(),
	}

	var respBytes []byte
	in, err := tool.ToProtoBytes(req)
	if err != nil {
		logx.Errorf("Error marshaling request for exec item %d: %v", execItem.Id, err)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "fail", "Error marshaling request: "+err.Error()); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}

		logEntry := &model.PlanExecLog{
			PlanId:      execItem.PlanId,
			PlanName:    plan.PlanName,
			ItemId:      execItem.ItemId,
			ItemName:    execItem.ItemName,
			PointId:     execItem.PointId,
			TriggerTime: time.Now(),
			TraceId:     "",
			ExecResult:  2, // 失败
			Message:     "Error marshaling request: " + err.Error(),
		}
		if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
			logx.Errorf("Error inserting plan exec log for item %d: %v", execItem.Id, err)
		}
		return
	}
	err = cli.Conn().Invoke(ctx, streamevent.StreamEvent_HandlerPlanTaskEvent_FullMethodName, &in, &respBytes)
	if err != nil {
		logx.Errorf("gRPC call failed for exec item %d: %v", execItem.Id, err)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "fail", "gRPC call failed: "+err.Error()); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}

		// 记录执行日志
		logEntry := &model.PlanExecLog{
			PlanId:      execItem.PlanId,
			PlanName:    plan.PlanName,
			ItemId:      execItem.ItemId,
			ItemName:    execItem.ItemName,
			PointId:     execItem.PointId,
			TriggerTime: time.Now(),
			TraceId:     "",
			ExecResult:  2, // 失败
			Message:     "gRPC call failed: " + err.Error(),
		}
		if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
			logx.Errorf("Error inserting plan exec log for item %d: %v", execItem.Id, err)
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

		// 记录执行日志
		logEntry := &model.PlanExecLog{
			PlanId:      execItem.PlanId,
			PlanName:    plan.PlanName,
			ItemId:      execItem.ItemId,
			ItemName:    execItem.ItemName,
			PointId:     execItem.PointId,
			TriggerTime: time.Now(),
			TraceId:     "",
			ExecResult:  2, // 失败
			Message:     "Error unmarshaling response: " + err.Error(),
		}
		if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
			logx.Errorf("Error inserting plan exec log for item %d: %v", execItem.Id, err)
		}
		return
	}

	// 记录执行日志
	logEntry := &model.PlanExecLog{
		PlanId:      execItem.PlanId,
		PlanName:    plan.PlanName,
		ItemId:      execItem.ItemId,
		ItemName:    execItem.ItemName,
		PointId:     execItem.PointId,
		TriggerTime: time.Now(),
		TraceId:     "",
		ExecResult:  int64(res.ExecResult),
		Message:     res.Message,
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
			var delayReason = res.Message
			if len(res.DelayConfig.DelayReason) != 0 {
				delayReason = fmt.Sprintf("reason: %s, message: %s", res.DelayConfig.DelayReason, res.Message)
			}
			if err := s.svcCtx.PlanExecItemModel.UpdateStatusToDelayed(ctx, execItem.Id, "delay", delayReason, res.DelayConfig.NextTriggerTime); err != nil {
				logx.Errorf("Error updating plan exec item %d to delayed: %v", execItem.Id, err)
			}
		} else {
			logx.Errorf("No delay config provided for exec item %d", execItem.Id)
		}
	default:
		logx.Errorf("Unknown execResult %d for exec item %d", res.ExecResult, execItem.Id)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, "complete", res.Message); err != nil {
			logx.Errorf("Error updating plan exec item %d to completed: %v", execItem.Id, err)
		}
	}
	// 插入执行日志
	if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
		logx.Errorf("Error inserting plan exec log for item %d: %v", execItem.Id, err)
	}
	logx.Infof("Successfully executed callback for plan exec item: id=%d", execItem.Id)
}
