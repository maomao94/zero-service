package logic

import (
	"zero-service/app/djicloud/djicloud"
	"zero-service/common/djisdk"
)

func errRes(tid string, err error) *djicloud.CommonRes {
	if djiErr, ok := djisdk.IsDJIError(err); ok {
		return &djicloud.CommonRes{
			Code:       -1,
			Message:    djiErr.Message,
			Tid:        tid,
			ReasonCode: int32(djiErr.Code),
		}
	}
	return &djicloud.CommonRes{Code: -1, Message: err.Error(), Tid: tid}
}

func okRes(tid string) *djicloud.CommonRes {
	return &djicloud.CommonRes{Code: 0, Message: "success", Tid: tid}
}
