package logic

import (
	"strconv"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/model/gormmodel"

	"github.com/dromara/carbon/v2"
)

func toPbDevicePointMapping(m *gormmodel.GormDevicePointMapping) *ieccaller.PbDevicePointMapping {
	if m == nil {
		return nil
	}
	id, _ := strconv.ParseInt(m.Id, 10, 64)
	return &ieccaller.PbDevicePointMapping{
		Id:          id,
		CreateTime:  carbon.CreateFromStdTime(m.CreateTime).Format(carbon.DateTimeMicroFormat),
		UpdateTime:  carbon.CreateFromStdTime(m.UpdateTime).Format(carbon.DateTimeMicroFormat),
		TagStation:  m.TagStation,
		Coa:         int64(m.Coa),
		Ioa:         int64(m.Ioa),
		DeviceId:    m.DeviceId,
		DeviceName:  m.DeviceName,
		TdTableType: m.TdTableType,
		EnablePush:  int32(m.EnablePush),
		Description: m.Description.String,
		Ext1:        m.Ext1.String,
		Ext2:        m.Ext2.String,
		Ext3:        m.Ext3.String,
		Ext4:        m.Ext4.String,
		Ext5:        m.Ext5.String,
	}
}
