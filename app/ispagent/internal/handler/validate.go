package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dromara/carbon/v2"
)

// ---- 枚举常量 ----

var (
	validTypes         = map[string]bool{"1": true, "2": true, "3": true, "4": true} // 巡视类型: 1=例行 2=特殊 3=专项 4=自定义
	validPriorities    = map[string]bool{"1": true, "2": true, "3": true, "4": true} // 优先级
	validEnables       = map[string]bool{"0": true, "1": true, "2": true}            // 启用: 0=启用 1=禁用 2=删除
	validIntervalTypes = map[string]bool{"1": true, "2": true}                       // 间隔: 1=小时 2=天
	validDeviceLevels  = map[int]bool{1: true, 2: true, 3: true, 4: true}            // 设备层级
)

// ---- 入口校验 ----

// validateTaskItem 对 ISP 任务下发 Item 做完整校验，不通过返回 error。
func validateTaskItem(idx int, item map[string]string) error {
	if err := validateRequiredFields(idx, item); err != nil {
		return err
	}
	if err := validateEnumFields(idx, item); err != nil {
		return err
	}
	if err := validateSchedule(idx, item); err != nil {
		return err
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

// ---- 必填字段 ----

func validateRequiredFields(idx int, item map[string]string) error {
	if code := item["task_code"]; strings.TrimSpace(code) == "" {
		return fmt.Errorf("任务[%d] task_code 不能为空", idx)
	}
	if name := item["task_name"]; strings.TrimSpace(name) == "" {
		return fmt.Errorf("任务[%d] task_name 不能为空", idx)
	}
	return nil
}

// ---- 枚举字段 ----

func validateEnumFields(idx int, item map[string]string) error {
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
	return nil
}

// ---- 调度类型校验（定期/周期/间隔） ----

func validateSchedule(idx int, item map[string]string) error {
	fixed := strings.TrimSpace(item["fixed_start_time"])
	cycleMonth := strings.TrimSpace(item["cycle_month"])
	cycleWeek := strings.TrimSpace(item["cycle_week"])
	cycleExec := strings.TrimSpace(item["cycle_execute_time"])
	intervalNum := strings.TrimSpace(item["interval_number"])
	intervalType := strings.TrimSpace(item["interval_type"])
	intervalExec := strings.TrimSpace(item["interval_execute_time"])

	switch {
	case fixed != "": // 定期任务
		if !isDateTimeFormat(fixed) {
			return fmt.Errorf("任务[%d] fixed_start_time 格式应为 yyyy-MM-dd HH:mm:ss: %s", idx, fixed)
		}

	case cycleMonth != "" && cycleWeek != "" && cycleExec != "": // 周期任务
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

	case intervalNum != "" && intervalType != "" && intervalExec != "": // 间隔任务
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
	return nil
}

// ---- 格式工具 ----

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
