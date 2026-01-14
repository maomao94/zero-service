package cron

import (
	"context"
	"database/sql"
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
		for {
			itemsProcessed := s.ScanPlanExecItem()
			var sleepDuration time.Duration
			if itemsProcessed {
				sleepDuration = 10 * time.Millisecond
			} else {
				sleepDuration = time.Duration(1000+rand.Intn(1000)) * time.Millisecond // 1~2秒随机
			}
			timer := time.NewTimer(sleepDuration)
			select {
			case <-s.cancelChan:
				if !timer.Stop() {
					<-timer.C
				}
				return
			case <-timer.C:
			}
		}
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
		queryExecItem, queryErr := s.svcCtx.PlanExecItemModel.FindOne(ctx, execItem.Id)
		if queryErr != nil {
			logx.Errorf("Error querying plan exec item: %v", queryErr)
		}
		plan, queryErr := s.svcCtx.PlanModel.FindOneByPlanId(ctx, execItem.PlanId)
		if queryErr != nil {
			logx.Errorf("Error querying plan: %v", queryErr)
		}
		logx.Infof("Found plan exec item to trigger: id=%d, planPk=%d, planId=%s, planName=%s, itemId=%s, itemName=%s, pointId=%s, nextTriggerTime=%v",
			execItem.Id,
			execItem.PlanPk,
			execItem.PlanId,
			plan.PlanName.String,
			execItem.ItemId,
			execItem.ItemName.String,
			execItem.PointId.String,
			execItem.NextTriggerTime,
		)
		s.ExecuteCallback(ctx, queryExecItem, plan)
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
			if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToCallback(ctx, execItem.Id, model.ResultFailed, "Failed to create gRPC client: "+err.Error()); updateErr != nil {
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
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToCallback(ctx, execItem.Id, model.ResultFailed, "gRPC client is nil"); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}

	cli, ok := v.(*zrpc.RpcClient)
	if !ok {
		logx.Errorf("Invalid connection type in ConnMap for %s", grpcServer)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToCallback(ctx, execItem.Id, model.ResultFailed, "Invalid connection type in ConnMap"); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}

	req := &streamevent.HandlerPlanTaskEventReq{
		CreateTime:      carbon.CreateFromStdTime(execItem.CreateTime).ToDateTimeString(),
		UpdateTime:      carbon.CreateFromStdTime(execItem.UpdateTime).ToDateTimeString(),
		CreateUser:      execItem.CreateUser.String,
		UpdateUser:      execItem.UpdateUser.String,
		PlanId:          execItem.PlanId,
		PlanName:        plan.PlanName.String,
		Type:            plan.Type.String,
		GroupId:         plan.GroupId.String,
		Description:     plan.Description.String,
		StartTime:       carbon.NewCarbon(plan.StartTime).ToDateTimeString(),
		EndTime:         carbon.NewCarbon(plan.EndTime).ToDateTimeString(),
		PlanPk:          plan.Id,
		BatchId:         execItem.BatchId,
		ItemId:          execItem.ItemId,
		ItemName:        execItem.ItemName.String,
		PointId:         execItem.PointId.String,
		Payload:         execItem.Payload,
		PlanTriggerTime: carbon.NewCarbon(execItem.PlanTriggerTime).ToDateTimeString(),
	}

	var respBytes []byte
	in, err := tool.ToProtoBytes(req)
	if err != nil {
		logx.Errorf("Error marshaling request for exec item %d: %v", execItem.Id, err)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToCallback(ctx, execItem.Id, model.ResultFailed, "Error marshaling request: "+err.Error()); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}

		logEntry := &model.PlanExecLog{
			PlanId:      execItem.PlanId,
			PlanPk:      plan.Id,
			PlanName:    plan.PlanName,
			ItemPk:      execItem.Id,
			ItemId:      execItem.ItemId,
			ItemName:    execItem.ItemName,
			PointId:     execItem.PointId,
			TriggerTime: time.Now(),
			TraceId:     sql.NullString{String: "", Valid: false},
			ExecResult:  sql.NullString{String: model.ResultFailed, Valid: true}, // 失败
			Message:     sql.NullString{String: "Error marshaling request: " + err.Error(), Valid: true},
		}
		if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
			logx.Errorf("Error inserting plan exec log for item %d: %v", execItem.Id, err)
		}
		return
	}
	err = cli.Conn().Invoke(ctx, streamevent.StreamEvent_HandlerPlanTaskEvent_FullMethodName, &in, &respBytes)
	if err != nil {
		logx.Errorf("gRPC call failed for exec item %d: %v", execItem.Id, err)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToCallback(ctx, execItem.Id, model.ResultFailed, "gRPC call failed: "+err.Error()); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}

		// 记录执行日志
		logEntry := &model.PlanExecLog{
			PlanId:      execItem.PlanId,
			PlanPk:      plan.Id,
			PlanName:    plan.PlanName,
			ItemPk:      execItem.Id,
			ItemId:      execItem.ItemId,
			ItemName:    execItem.ItemName,
			PointId:     execItem.PointId,
			TriggerTime: time.Now(),
			TraceId:     sql.NullString{String: "", Valid: false},
			ExecResult:  sql.NullString{String: model.ResultFailed, Valid: true}, // 失败
			Message:     sql.NullString{String: "gRPC call failed: " + err.Error(), Valid: true},
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
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToCallback(ctx, execItem.Id, model.ResultFailed, "Error unmarshaling response: "+err.Error()); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}

		// 记录执行日志
		logEntry := &model.PlanExecLog{
			PlanId:      execItem.PlanId,
			PlanPk:      plan.Id,
			PlanName:    sql.NullString{String: plan.PlanName.String, Valid: plan.PlanName.Valid},
			ItemPk:      execItem.Id,
			ItemId:      execItem.ItemId,
			ItemName:    sql.NullString{String: execItem.ItemName.String, Valid: execItem.ItemName.Valid},
			PointId:     sql.NullString{String: execItem.PointId.String, Valid: execItem.PointId.Valid},
			TriggerTime: time.Now(),
			TraceId:     sql.NullString{String: "", Valid: false},
			ExecResult:  sql.NullString{String: model.ResultFailed, Valid: true}, // 失败
			Message:     sql.NullString{String: "Error unmarshaling response: " + err.Error(), Valid: true},
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
		ItemPk:      execItem.Id,
		ItemId:      execItem.ItemId,
		ItemName:    execItem.ItemName,
		PointId:     execItem.PointId,
		TriggerTime: time.Now(),
		TraceId:     sql.NullString{String: "", Valid: false},
		ExecResult:  sql.NullString{String: res.ExecResult, Valid: res.ExecResult != ""},
		Message:     sql.NullString{String: res.Message, Valid: res.Message != ""},
	}

	switch res.ExecResult {
	case model.ResultCompleted: // completed
		logx.Infof("gRPC call succeeded for exec item %d", execItem.Id)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, res.Message); err != nil {
			logx.Errorf("Error updating plan exec item %d to completed: %v", execItem.Id, err)
		}
	case model.ResultFailed: // Failed
		logx.Infof("gRPC call returned failure for exec item %d: %s", execItem.Id, res.Message)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCallback(ctx, execItem.Id, model.ResultFailed, res.Message); err != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, err)
		}
	case model.ResultDelayed, model.ResultRunning:
		logx.Infof("gRPC call returned delayed for exec item %d: %s", execItem.Id, res.Message)
		currentTime := carbon.Now()
		delayTriggerTime := currentTime.AddMinutes(5).ToDateTimeString()
		delayReason := ""
		if len(res.Message) == 0 {
			delayReason = res.ExecResult
		} else {
			delayReason = res.Message
		}
		if res.DelayConfig == nil {
			logx.Errorf("No delay config provided for exec item %d", execItem.Id)
		} else {
			if len(res.DelayConfig.DelayReason) != 0 {
				delayReason = fmt.Sprintf("reason: %s, message: %s", res.DelayConfig.DelayReason, res.Message)
			}
			delayTime := carbon.ParseByLayout(res.DelayConfig.NextTriggerTime, carbon.DateTimeLayout)
			isTrue := true
			if delayTime.Error != nil || delayTime.IsInvalid() {
				logx.Errorf("Invalid delay time format for exec item %d: %s", execItem.Id, res.DelayConfig.NextTriggerTime)
				isTrue = false
			} else {
				if delayTime.Lt(currentTime) {
					logx.Errorf("Delay time for exec item %d is in the past: %v, current time: %v", execItem.Id, delayTime.ToDateTimeString(), currentTime.ToDateTimeString())
					isTrue = false
				}
			}
			if isTrue {
				delayTriggerTime = delayTime.ToDateTimeString()
			}
		}
		delayReason = fmt.Sprintf("%s, delay time: %s", delayReason, delayTriggerTime)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToDelayed(ctx, execItem.Id, res.ExecResult, delayReason, delayTriggerTime); err != nil {
			logx.Errorf("Error updating plan exec item %d to delayed: %v", execItem.Id, err)
		}
	default:
		logx.Errorf("Unknown execResult %s for exec item %d", res.ExecResult, execItem.Id)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, res.Message); err != nil {
			logx.Errorf("Error updating plan exec item %d to completed: %v", execItem.Id, err)
		}
	}
	// 插入执行日志
	if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
		logx.Errorf("Error inserting plan exec log for item %d: %v", execItem.Id, err)
	}
	logx.Infof("Successfully executed callback for plan exec item: id=%d", execItem.Id)
}
