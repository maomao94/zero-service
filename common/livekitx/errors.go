package livekitx

import "errors"

// 包级可判定的本地错误。SDK/管理请求错误用 %w 保留原始语义，
// 不在此包装。

// ErrInvalidConfig 表示配置缺失、非法或互斥选项同时设置（New 构造期返回）。
var ErrInvalidConfig = errors.New("livekitx: invalid configuration")

// ErrClosed 表示 Client 已关闭，不能再加入/创建房间或发起管理请求。
var ErrClosed = errors.New("livekitx: client is closed")

// ErrInvalidTokenOptions 表示 Token 构造参数缺失或互斥；
// 同时用于房间名/身份等参数校验，避免为每种参数新增错误。
var ErrInvalidTokenOptions = errors.New("livekitx: invalid token options")
