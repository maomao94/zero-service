package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"zero-service/app/ispagent/model/gormmodel"
	"zero-service/common/crontask"
	"zero-service/common/gormx"
	"zero-service/common/isp"

	ctask "zero-service/app/ispagent/internal/crontask"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm/clause"
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
		if err != nil && err != crontask.ErrNotFound {
			logx.WithContext(ctx).Errorf("[ispagent] 查询已存任务失败: %v", err)
			continue
		}

		var existingID int64
		if existing != nil {
			existingID = existing.ID
		}

		cfg := ctask.NewTaskConfig(existingID, fields)
		if fields.IsEnable == "2" {
			if existingID != 0 {
				if err := store.Delete(ctx, existingID); err != nil {
					logx.WithContext(ctx).Errorf("[ispagent] 删除任务失败: %v", err)
				}
			}
			continue
		}

		if existingID == 0 {
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

// ResponseCode 根据 error 返回 ISP 状态码。
func ResponseCode(err error) string {
	if err != nil {
		return isp.StatusError
	}
	return isp.StatusSuccess
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

var (
	validTypes         = map[string]bool{"1": true, "2": true, "3": true, "4": true}
	validPriorities    = map[string]bool{"1": true, "2": true, "3": true, "4": true}
	validEnables       = map[string]bool{"0": true, "1": true, "2": true}
	validIntervalTypes = map[string]bool{"1": true, "2": true}
	validDeviceLevels  = map[int]bool{1: true, 2: true, 3: true, 4: true}
)

func validateTaskItem(idx int, item map[string]string) error {
	if code := item["task_code"]; strings.TrimSpace(code) == "" {
		return fmt.Errorf("任务[%d] task_code 不能为空", idx)
	}
	if name := item["task_name"]; strings.TrimSpace(name) == "" {
		return fmt.Errorf("任务[%d] task_name 不能为空", idx)
	}
	if typ := item["type"]; typ != "" && !validTypes[typ] {
		return fmt.Errorf("任务[%d] type 无效: %s", idx, typ)
	}
	if pri := item["priority"]; pri != "" && !validPriorities[pri] {
		return fmt.Errorf("任务[%d] priority 无效: %s", idx, pri)
	}
	if en := item["isenable"]; en != "" && !validEnables[en] {
		return fmt.Errorf("任务[%d] isenable 无效: %s", idx, en)
	}
	if dl := item["device_level"]; dl != "" {
		if n, err := strconv.Atoi(dl); err != nil || !validDeviceLevels[n] {
			return fmt.Errorf("任务[%d] device_level 无效: %s", idx, dl)
		}
	}

	fixed := strings.TrimSpace(item["fixed_start_time"])
	cycleMonth := strings.TrimSpace(item["cycle_month"])
	cycleWeek := strings.TrimSpace(item["cycle_week"])
	cycleExec := strings.TrimSpace(item["cycle_execute_time"])
	intervalNum := strings.TrimSpace(item["interval_number"])
	intervalType := strings.TrimSpace(item["interval_type"])
	intervalExec := strings.TrimSpace(item["interval_execute_time"])

	switch {
	case fixed != "":
		if !isDateTimeFormat(fixed) {
			return fmt.Errorf("任务[%d] fixed_start_time 格式应为 yyyy-MM-dd HH:mm:ss: %s", idx, fixed)
		}

	case cycleMonth != "" && cycleWeek != "" && cycleExec != "":
		if err := validateCSVIntRange(cycleMonth, 1, 12); err != nil {
			return fmt.Errorf("任务[%d] cycle_month: %w", idx, err)
		}
		if err := validateCSVIntRange(cycleWeek, 1, 7); err != nil {
			return fmt.Errorf("任务[%d] cycle_week: %w", idx, err)
		}
		if !isTimeFormat(cycleExec) {
			return fmt.Errorf("任务[%d] cycle_execute_time 格式应为 HH:mm:ss: %s", idx, cycleExec)
		}
		if err := validateOptionalDateTime(item, "cycle_start_time", idx); err != nil {
			return err
		}
		if err := validateOptionalDateTime(item, "cycle_end_time", idx); err != nil {
			return err
		}

	case intervalNum != "" && intervalType != "" && intervalExec != "":
		if n, err := strconv.Atoi(intervalNum); err != nil || n <= 0 {
			return fmt.Errorf("任务[%d] interval_number 应为正整数: %s", idx, intervalNum)
		}
		if !validIntervalTypes[intervalType] {
			return fmt.Errorf("任务[%d] interval_type 应为 1 或 2: %s", idx, intervalType)
		}
		if !isTimeFormat(intervalExec) {
			return fmt.Errorf("任务[%d] interval_execute_time 格式应为 HH:mm:ss: %s", idx, intervalExec)
		}
		if err := validateOptionalDateTime(item, "interval_start_time", idx); err != nil {
			return err
		}
		if err := validateOptionalDateTime(item, "interval_end_time", idx); err != nil {
			return err
		}

	default:
		return fmt.Errorf("任务[%d] 参数不满足条件: 未匹配定期/周期/间隔任务", idx)
	}

	if err := validateOptionalDateTime(item, "invalid_start_time", idx); err != nil {
		return err
	}
	if err := validateOptionalDateTime(item, "invalid_end_time", idx); err != nil {
		return err
	}
	if err := validateOptionalDateTime(item, "create_time", idx); err != nil {
		return err
	}
	return nil
}

func isDateTimeFormat(s string) bool {
	return carbon.Parse(s).Error == nil
}

func isTimeFormat(s string) bool {
	if len(s) != 8 || s[2] != ':' || s[5] != ':' {
		return false
	}
	h, e1 := strconv.Atoi(s[:2])
	m, e2 := strconv.Atoi(s[3:5])
	sec, e3 := strconv.Atoi(s[6:])
	return e1 == nil && e2 == nil && e3 == nil && h >= 0 && h <= 23 && m >= 0 && m <= 59 && sec >= 0 && sec <= 59
}

func validateCSVIntRange(s string, min, max int) error {
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		n, err := strconv.Atoi(p)
		if err != nil || n < min || n > max {
			return fmt.Errorf("值 %s 不在范围 [%d,%d]", p, min, max)
		}
	}
	return nil
}

func validateOptionalDateTime(item map[string]string, key string, idx int) error {
	val := strings.TrimSpace(item[key])
	if val == "" {
		return nil
	}
	if !isDateTimeFormat(val) {
		return fmt.Errorf("任务[%d] %s 格式应为 yyyy-MM-dd HH:mm:ss: %s", idx, key, val)
	}
	return nil
}

// taskControlName 任务控制指令 → 中文名称。
var taskControlName = map[int32]string{
	isp.CommandTaskStart:  "启动",
	isp.CommandTaskPause:  "暂停",
	isp.CommandTaskResume: "继续",
	isp.CommandTaskStop:   "停止",
}

// taskControlToState 任务控制指令 → 任务状态。
var taskControlToState = map[int32]string{
	isp.CommandTaskStart:  "2", // 正在执行
	isp.CommandTaskPause:  "3", // 暂停
	isp.CommandTaskResume: "2", // 正在执行
	isp.CommandTaskStop:   "4", // 终止
}

// HandleTaskControl 处理任务控制指令 (41-1/2/3/4)。
// Command=1(启动): msg.Code 为 task_code。
// Command=2/3/4(暂停/继续/停止): msg.Code 为 变电站编码_任务编码_时间。
func HandleTaskControl(ctx context.Context, msg *isp.Message, store crontask.TaskStore, db *gormx.DB, sendCode, receiveCode string, notify func(ctx context.Context, code string, items []isp.Item)) (string, error) {
	if store == nil {
		return "", fmt.Errorf("store is nil")
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

	var taskPatrolledID, substationCode, taskCode, execTime string
	if msg.Command == isp.CommandTaskStart {
		taskCode = msg.Code
	} else {
		parts := strings.SplitN(msg.Code, "_", 3)
		if len(parts) < 2 {
			return "", fmt.Errorf("无效的 patrol ID: %s", msg.Code)
		}
		substationCode = parts[0]
		taskCode = parts[1]
		taskPatrolledID = msg.Code
		if len(parts) >= 3 {
			execTime = parts[2]
		}
	}

	task, err := store.GetByCode(ctx, taskCode)
	if err != nil {
		return "", fmt.Errorf("查询任务配置失败: %w", err)
	}

	if msg.Command == isp.CommandTaskStart {
		fields := ctask.DeserializeExtra(string(task.Extra))
		if fields != nil {
			substationCode = fields.SubstationCode
		}
		if substationCode == "" {
			return "", fmt.Errorf("任务 %s 缺少变电站编码", taskCode)
		}
		now := carbon.Now()
		execTime = now.ToDateTimeString()
		taskPatrolledID = fmt.Sprintf("%s_%s_%s", substationCode, taskCode, now.Format("YmdHis"))
		if err := store.UpdateNextRun(ctx, task.ID, task.NextRun, now.StdTime()); err != nil {
			logx.WithContext(ctx).Errorf("[ispagent] 更新 last_run 失败: %v", err)
		}
	}

	items := []isp.Item{{
		"task_patrolled_id":   taskPatrolledID,
		"task_name":           task.TaskName,
		"task_code":           taskCode,
		"task_state":          state,
		"plan_start_time":     execTime,
		"start_time":          execTime,
		"task_progress":       "0",
		"task_estimated_time": "",
		"description":         "",
	}}
	updateColumns := []string{
		"send_code",
		"receive_code",
		"code",
		"task_name",
		"task_code",
		"task_state",
		"task_progress",
		"task_estimated_time",
		"description",
	}
	if msg.Command == isp.CommandTaskStart {
		updateColumns = append(updateColumns, "plan_start_time", "start_time")
	}
	if err := upsertPatrolTask(ctx, db, &gormmodel.GormIspPatrolTask{
		SendCode:          sendCode,
		ReceiveCode:       receiveCode,
		Code:              substationCode,
		TaskPatrolledID:   taskPatrolledID,
		TaskName:          task.TaskName,
		TaskCode:          taskCode,
		TaskState:         state,
		PlanStartTime:     execTime,
		StartTime:         execTime,
		TaskProgress:      "0",
		TaskEstimatedTime: "",
		Description:       "",
	}, updateColumns); err != nil {
		return "", fmt.Errorf("同步巡视任务表失败: %w", err)
	}
	if notify != nil {
		go notify(context.Background(), substationCode, items)
	}
	return taskPatrolledID, nil
}

func upsertPatrolTask(ctx context.Context, db *gormx.DB, task *gormmodel.GormIspPatrolTask, updateColumns []string) error {
	if db == nil || task == nil {
		return nil
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "task_patrolled_id"}},
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).Create(task).Error
}
