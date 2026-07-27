package crontask

import "errors"

var (
	ErrNotFound  = errors.New("[crontask] task not found")
	ErrDuplicate = errors.New("[crontask] task code already exists")
	ErrUpdate    = errors.New("[crontask] task update affected no rows")
	// ErrUnsupportedDescription 表示 RRULE 合法，但包含无法准确转换为业务文案的组合。
	ErrUnsupportedDescription = errors.New("[crontask] unsupported rrule description")
	// ErrDeleteTask 表示业务回调确认业务任务已不存在，要求调度器删除当前调度任务。
	// Handler 可以直接返回或包装该错误。调度器识别后调用 TaskStore.Delete，删除成功或
	// ErrNotFound 都结束本次执行且不再推进 NextRun；删除失败则保留 lease，等待过期后重试。
	ErrDeleteTask = errors.New("[crontask] delete current task")
)
