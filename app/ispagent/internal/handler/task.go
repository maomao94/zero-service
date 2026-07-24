package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"zero-service/app/ispagent/model/gormmodel"
	"zero-service/common/crontask"
	"zero-service/common/gormx"
	"zero-service/common/isp"
	"zero-service/common/tool"

	ctask "zero-service/app/ispagent/internal/crontask"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"gorm.io/gorm"
)

// HandleTaskDispatch 处理任务下发指令 (101-1)。
// 将 ISP 任务 Item 解析为 IspTaskFields，通过 store 插入或更新 cron 任务。
// 对任一 Item 校验不通过，立即返回 error 触发 251-3 下发失败响应。
func HandleTaskDispatch(ctx context.Context, msg *isp.Message, store crontask.TaskStore) error {
	if store == nil {
		logx.WithContext(ctx).Error("[ispagent] taskStore is nil, skip task dispatch")
		return nil
	}
	logx.WithContext(ctx).Infof("[ispagent] 任务下发 code=%s items=%d", msg.Code, len(msg.Items))

	for i, item := range msg.Items {
		if err := validateTaskItem(i, item); err != nil {
			return err
		}

		deviceList := strutil.SplitAndTrim(item["device_list"], ",")
		fields := itemToFields(msg.Code, item)
		logx.WithContext(ctx).Infof("[ispagent] 任务[%d] code=%s name=%s type=%s priority=%s device_level=%s device_list_size=%v",
			i, fields.TaskCode, fields.TaskName, fields.TaskType(), fields.Priority, strconv.Itoa(fields.DeviceLevel), len(deviceList))

		existing, err := store.GetByCode(ctx, fields.TaskCode)
		if err != nil && !errors.Is(err, crontask.ErrNotFound) {
			logx.WithContext(ctx).Errorf("[ispagent] 查询已存任务失败: %v", err)
			return err
		}

		cfg, err := ctask.NewTaskConfig(existing, fields)
		if err != nil {
			return fmt.Errorf("构建任务 %s 调度配置失败: %w", fields.TaskCode, err)
		}
		if fields.IsEnable == "2" {
			if cfg.ID != "" {
				if err := store.Delete(ctx, cfg.ID); err != nil {
					logx.WithContext(ctx).Errorf("[ispagent] 删除任务失败: %v", err)
					return err
				}
			}
			continue
		}

		if cfg.ID == "" {
			if err := store.Insert(ctx, cfg); err != nil {
				logx.WithContext(ctx).Errorf("[ispagent] 插入任务 %s 失败: %v", fields.TaskCode, err)
				return err
			}
		} else {
			if err := store.Update(ctx, cfg); err != nil {
				logx.WithContext(ctx).Errorf("[ispagent] 更新任务 %s 失败: %v", fields.TaskCode, err)
				return err
			}
		}
	}
	return nil
}

// itemToFields 将 ISP Item map 转为 IspTaskFields。
func itemToFields(substationCode string, item map[string]string) *ctask.IspTaskFields {
	return &ctask.IspTaskFields{
		SubstationCode:      substationCode,
		PatrolType:          item["type"],
		TaskCode:            item["task_code"],
		TaskName:            item["task_name"],
		Priority:            item["priority"],
		DeviceLevel:         atoi(item["device_level"]),
		DeviceList:          item["device_list"],
		FixedStartTime:      item["fixed_start_time"],
		CycleMonth:          item["cycle_month"],
		CycleWeek:           item["cycle_week"],
		CycleExecuteTime:    item["cycle_execute_time"],
		CycleStartTime:      item["cycle_start_time"],
		CycleEndTime:        item["cycle_end_time"],
		IntervalNumber:      item["interval_number"],
		IntervalType:        item["interval_type"],
		IntervalExecuteTime: item["interval_execute_time"],
		IntervalStartTime:   item["interval_start_time"],
		IntervalEndTime:     item["interval_end_time"],
		InvalidStartTime:    item["invalid_start_time"],
		InvalidEndTime:      item["invalid_end_time"],
		IsEnable:            item["isenable"],
		Creator:             item["creator"],
		CreateTime:          item["create_time"],
	}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// taskControlToState 任务控制指令 → 任务状态。 todo 待定 设备没接入
var taskControlToState = map[int32]string{
	isp.CommandTaskStart:  "2", // 正在执行
	isp.CommandTaskPause:  "3", // 暂停
	isp.CommandTaskResume: "2", // 正在执行
	isp.CommandTaskStop:   "4", // 终止
}

const taskControlNotifyDelay = 3 * time.Second

// HandleTaskControl 处理任务控制指令 (41-1/2/3/4)。
// Command=1(启动): msg.Code 为 task_code。
// Command=2/3/4(暂停/继续/停止): msg.Code 为 变电站编码_任务编码_时间。
func HandleTaskControl(ctx context.Context, msg *isp.Message, store crontask.TaskStore, runTask func(context.Context, string) error, db *gormx.DB, notify func(ctx context.Context, code string, items []isp.Item)) (string, error) {
	return handleTaskControl(ctx, msg, store, runTask, db, notify, taskControlNotifyDelay)
}

func handleTaskControl(ctx context.Context, msg *isp.Message, store crontask.TaskStore, runTask func(context.Context, string) error, db *gormx.DB, notify func(ctx context.Context, code string, items []isp.Item), notifyDelay time.Duration) (string, error) {
	if msg == nil {
		return "", fmt.Errorf("任务控制消息为空")
	}
	if store == nil {
		return "", fmt.Errorf("store is nil")
	}
	if len(msg.Items) > 0 {
		return "", fmt.Errorf("任务控制指令不应包含 Item，当前 %d 条", len(msg.Items))
	}

	name := taskControlName[msg.Command]
	if name == "" {
		return "", fmt.Errorf("未知控制指令: %d", msg.Command)
	}
	state := taskControlToState[msg.Command]
	if state == "" {
		return "", fmt.Errorf("未定义的指令状态映射: %d", msg.Command)
	}
	logx.WithContext(ctx).Infof("[ispagent] 任务控制 code=%s command=%s", msg.Code, name)

	var taskPatrolledID, substationCode, taskCode, taskName, planStartTime, startTime string
	if msg.Command == isp.CommandTaskStart {
		taskCode = msg.Code
	} else {
		taskPatrolledID = msg.Code
		if db == nil {
			return "", fmt.Errorf("db is nil")
		}
		var patrolTask gormmodel.GormIspPatrolTask
		if err := db.WithContext(ctx).
			Select("code", "task_code", "task_name", "plan_start_time", "start_time", "task_progress", "task_estimated_time", "description").
			Where("task_patrolled_id = ?", taskPatrolledID).
			First(&patrolTask).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", fmt.Errorf("巡视任务不存在: %s", taskPatrolledID)
			}
			return "", fmt.Errorf("查询巡视任务失败: %w", err)
		}
		substationCode = patrolTask.Code
		taskCode = patrolTask.TaskCode
		taskName = patrolTask.TaskName
		planStartTime = carbon.CreateFromStdTime(patrolTask.PlanStartTime).ToDateTimeString()
		startTime = carbon.CreateFromStdTime(patrolTask.StartTime).ToDateTimeString()
	}

	if msg.Command == isp.CommandTaskStart {
		task, err := store.GetByCode(ctx, taskCode)
		if err != nil {
			if errors.Is(err, crontask.ErrNotFound) {
				return "", fmt.Errorf("任务不存在: %s", taskCode)
			}
			return "", fmt.Errorf("查询任务配置失败: %w", err)
		}
		if task == nil {
			return "", fmt.Errorf("任务不存在: %s", taskCode)
		}
		fields := ctask.DeserializeExtra(string(task.Extra))
		if fields != nil {
			substationCode = fields.SubstationCode
		}
		if substationCode == "" {
			return "", fmt.Errorf("任务 %s 缺少变电站编码", taskCode)
		}
		now := tool.NowStartOfSecond()
		taskPatrolledID = fmt.Sprintf("%s_%s_%s", substationCode, taskCode, now.ToShortDateTimeString())
		if runTask == nil {
			return "", fmt.Errorf("任务调度器未初始化")
		}
		runCtx := ctask.WithManualExecution(ctx, taskPatrolledID, now.AddSeconds(10).StdTime())
		if err := runTask(runCtx, taskCode); err != nil {
			return "", fmt.Errorf("立即执行任务 %s 失败: %w", taskCode, err)
		}
		return taskPatrolledID, nil
	}

	items := []isp.Item{{
		"task_patrolled_id":   taskPatrolledID,
		"task_name":           taskName,
		"task_code":           taskCode,
		"task_state":          state,
		"plan_start_time":     planStartTime,
		"start_time":          startTime,
		"task_progress":       "0",
		"task_estimated_time": "",
		"description":         "",
	}}
	if err := db.WithContext(ctx).Model(&gormmodel.GormIspPatrolTask{}).
		Where("task_patrolled_id = ?", taskPatrolledID).
		Update("task_state", state).Error; err != nil {
		return "", fmt.Errorf("更新巡视任务状态失败: %w", err)
	}
	if notify != nil {
		threading.GoSafe(func() {
			if notifyDelay > 0 {
				time.Sleep(notifyDelay)
			}
			notify(context.Background(), substationCode, items)
		})
	}
	return taskPatrolledID, nil
}

func UpsertPatrolTask(ctx context.Context, db *gormx.DB, task *gormmodel.GormIspPatrolTask) error {
	if db == nil || task == nil {
		return nil
	}
	assign := map[string]any{
		"task_state":          task.TaskState,
		"task_progress":       task.TaskProgress,
		"task_estimated_time": task.TaskEstimatedTime,
		"description":         task.Description,
	}
	return db.WithContext(ctx).Where("task_patrolled_id = ?", task.TaskPatrolledID).Assign(assign).FirstOrCreate(task).Error
}
