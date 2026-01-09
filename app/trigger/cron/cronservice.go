package cron

import (
	"context"
	"fmt"
	"log"
	"time"
	"zero-service/app/trigger/internal/svc"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model"

	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type CronService struct {
	c      *cron.Cron
	svcCtx *svc.ServiceContext
}

func NewCronService(svcCtx *svc.ServiceContext) *CronService {
	service := &CronService{
		c:      cron.New(cron.WithSeconds()),
		svcCtx: svcCtx,
	}

	// Add scan table job to scan plan_exec_item every second
	_, err := service.c.AddFunc("@every 10s", service.ScanPlanExecItem)
	if err != nil {
		log.Fatalf("Failed to add scan plan exec item job: %v", err)
	}

	return service
}

func (s *CronService) Start() {
	s.c.Start()
	log.Print("Starting cron server \n")
}

func (s *CronService) Stop() {
	s.c.Stop()
}

// ScanPlanExecItem scans the plan_exec_item table and triggers items that need execution
func (s *CronService) ScanPlanExecItem() {
	threading.GoSafe(func() {
		ctx := context.Background()

		// Try to lock one plan exec item that needs triggering
		// Set an expiration of 10 seconds to prevent long-term locking
		execItem, err := s.svcCtx.PlanExecItemModel.LockTriggerItem(ctx, 10*time.Second)
		if err != nil {
			if err == sqlx.ErrNotFound {
				return
			}
			logx.Errorf("Error locking plan exec item: %v", err)
			return
		}

		if execItem == nil {
			// No item needs triggering, just return
			return
		}

		logx.Infof("Found plan exec item to trigger: id=%d, planId=%s, itemId=%s, nextTriggerTime=%v",
			execItem.Id,
			execItem.PlanId,
			execItem.ItemId,
			execItem.NextTriggerTime,
		)

		// Execute the callback logic
		s.ExecuteCallback(ctx, execItem)
	})
}

// rawCodec is a custom codec for gRPC that handles raw protobuf bytes
// This is needed for compatibility with the existing codebase pattern

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

// ExecuteCallback executes the callback to the streamevent service
func (s *CronService) ExecuteCallback(ctx context.Context, execItem *model.PlanExecItem) {
	grpcServer := tool.MayReplaceLocalhost(execItem.ServiceAddr)
	logx.Infof("Executing callback for exec item %d with service: %s, planId: %s, itemId: %s",
		execItem.Id, grpcServer, execItem.PlanId, execItem.ItemId)

	// Update status to running before executing callback
	if err := s.svcCtx.PlanExecItemModel.UpdateStatusToRunning(ctx, execItem.Id); err != nil {
		logx.Errorf("Error updating plan exec item %d to running: %v", execItem.Id, err)
		return
	}
	clientConf := zrpc.RpcClientConf{}
	conf.FillDefault(&clientConf)
	clientConf.Target = grpcServer
	clientConf.NonBlock = true
	clientConf.Timeout = 60000

	// Get existing connection from ConnMap or create new one
	v, ok := s.svcCtx.ConnMap.Get(grpcServer)
	if !ok {
		conn, err := zrpc.NewClient(clientConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			zrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.ForceCodec(rawCodec{}))))
		if err != nil {
			logx.Errorf("Failed to create gRPC client for %s: %v", grpcServer, err)
			// Update status to failed
			if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "Failed to create gRPC client: "+err.Error()); updateErr != nil {
				logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
			}
			return
		}
		// Store the connection in ConnMap for reuse
		s.svcCtx.ConnMap.Set(grpcServer, conn)
		v = conn
		logx.Infof("gRPC client inited for %s", grpcServer)
	}
	if v == nil {
		logx.Errorf("gRPC client is nil for %s", grpcServer)
		// Update status to failed
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "gRPC client is nil"); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}

	// Use the connection to create client
	cli, ok := v.(*zrpc.RpcClient)
	if !ok {
		logx.Errorf("Invalid connection type in ConnMap for %s", grpcServer)
		// Update status to failed
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "Invalid connection type in ConnMap"); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}

	// Set request timeout
	if execItem.RequestTimeout == 0 {
		execItem.RequestTimeout = clientConf.Timeout
	}

	// Create context with timeout if needed
	var cancel context.CancelFunc
	if execItem.RequestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(execItem.RequestTimeout)*time.Millisecond)
		defer cancel()
	}

	// Create request message
	req := &streamevent.HandlerPlanTaskEventReq{
		PlanId:   execItem.PlanId,
		PlanName: "",
		Type:     "",
		ItemId:   execItem.ItemId,
		ItemName: execItem.ItemName,
		Payload:  execItem.Payload,
	}
	var respBytes []byte
	in, err := tool.ToProtoBytes(req)
	if err != nil {
		logx.Errorf("Error marshaling request for exec item %d: %v", execItem.Id, err)
		// Update status to failed
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "Error marshaling request: "+err.Error()); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}
	err = cli.Conn().Invoke(ctx, streamevent.StreamEvent_HandlerPlanTaskEvent_FullMethodName, &in, &respBytes)
	if err != nil {
		logx.Errorf("gRPC call failed for exec item %d: %v", execItem.Id, err)
		// Update status to failed
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "gRPC call failed: "+err.Error()); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}
	res := &streamevent.HandlerPlanTaskEventRes{}
	err = proto.Unmarshal(respBytes, res)
	if err != nil {
		logx.Errorf("Error unmarshaling response for exec item %d: %v", execItem.Id, err)
		// Update status to failed
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, "Error unmarshaling response: "+err.Error()); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		}
		return
	}
	// Handle response based on execResult
	switch res.ExecResult {
	case 1: // Success
		logx.Infof("gRPC call succeeded for exec item %d", execItem.Id)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, res.Message); err != nil {
			logx.Errorf("Error updating plan exec item %d to completed: %v", execItem.Id, err)
		}
	case 2: // Failed
		logx.Infof("gRPC call returned failure for exec item %d: %s", execItem.Id, res.Message)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToFailed(ctx, execItem.Id, res.Message); err != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, err)
		}
	case 3: // Delay
		logx.Infof("gRPC call requested delay for exec item %d: %s", execItem.Id, res.Message)
		if res.DelayConfig != nil {
			// Update with next trigger time from delay config
			if err := s.svcCtx.PlanExecItemModel.UpdateStatusToDelayed(ctx, execItem.Id, res.Message, res.DelayConfig.NextTriggerTime); err != nil {
				logx.Errorf("Error updating plan exec item %d to delayed: %v", execItem.Id, err)
			}
		}
	default:
		logx.Errorf("Unknown execResult %d for exec item %d", res.ExecResult, execItem.Id)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, res.Message); err != nil {
			logx.Errorf("Error updating plan exec item %d to completed: %v", execItem.Id, err)
		}
	}

	logx.Infof("Successfully executed callback for plan exec item: ID=%d", execItem.Id)
}
