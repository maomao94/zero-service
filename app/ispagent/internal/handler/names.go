package handler

import "zero-service/common/isp"

var robotBodyName = map[int32]string{
	isp.CommandRobotRemoteReset:    "远方复位",
	isp.CommandRobotSelfCheck:      "系统自检",
	isp.CommandRobotReturnHome:     "一键返航",
	isp.CommandRobotManualCharge:   "手动充电",
	isp.CommandRobotModeSwitch:     "控制模式切换",
	isp.CommandRobotTakeControl:    "控制权获得",
	isp.CommandRobotReleaseControl: "控制权释放",
}

var robotChassisName = map[int32]string{
	isp.CommandChassisForward:    "前进",
	isp.CommandChassisBackward:   "后退",
	isp.CommandChassisTurnLeft:   "左转",
	isp.CommandChassisTurnRight:  "右转",
	isp.CommandChassisStop:       "停止",
	isp.CommandChassisUp:         "上升",
	isp.CommandChassisDown:       "下降",
	isp.CommandChassisShiftLeft:  "左平移",
	isp.CommandChassisShiftRight: "右平移",
	isp.CommandChassisGaitSwitch: "步态切换",
}

var robotPTZName = map[int32]string{
	isp.CommandPTZTiltUp:   "上仰",
	isp.CommandPTZTiltDown: "下俯",
	isp.CommandPTZPanLeft:  "左转",
	isp.CommandPTZPanRight: "右转",
	isp.CommandPTZRise:     "上升",
	isp.CommandPTZLower:    "下降",
	isp.CommandPTZPreset:   "预置位调用",
	isp.CommandPTZStop:     "停止",
	isp.CommandPTZReset:    "复位",
}

var robotAuxName = map[int32]string{
	isp.CommandAuxIRPower:    "红外电源",
	isp.CommandAuxWiper:      "雨刷",
	isp.CommandAuxUltrasound: "超声",
	isp.CommandAuxIRLamp:     "红外射灯",
	isp.CommandAuxLighting:   "辅助照明",
}

var visibleCameraName = map[int32]string{
	isp.CommandVisZoomIn:      "镜头拉近",
	isp.CommandVisZoomOut:     "镜头拉远",
	isp.CommandVisZoomStop:    "镜头拉焦停止",
	isp.CommandVisFocusInc:    "焦距增加",
	isp.CommandVisFocusDec:    "焦距减少",
	isp.CommandVisAutoFocus:   "自动聚焦",
	isp.CommandVisCapture:     "抓图",
	isp.CommandVisReboot:      "重启",
	isp.CommandVisRecordStart: "启动录像",
	isp.CommandVisRecordStop:  "停止录像",
	isp.CommandVisZoomSet:     "倍率值设置",
	isp.CommandVisFocusSet:    "聚焦值设置",
}

var thermalCameraName = map[int32]string{
	isp.CommandThermalFocusSet:  "设定焦距值",
	isp.CommandThermalAutoFocus: "自动聚焦",
	isp.CommandThermalCapture:   "抓图",
	isp.CommandThermalReboot:    "重启",
}

var partialDischargeName = map[int32]string{
	isp.CommandPartialDischargeExtend:  "伸长",
	isp.CommandPartialDischargeRetract: "收缩",
	isp.CommandPartialDischargeStop:    "停止",
	isp.CommandPartialDischargeReset:   "复位",
}

var robotControlNameByType = map[int32]map[int32]string{
	isp.TypeRobotBody:        robotBodyName,
	isp.TypeRobotChassis:     robotChassisName,
	isp.TypeRobotPTZ:         robotPTZName,
	isp.TypeRobotAux:         robotAuxName,
	isp.TypeVisibleCamera:    visibleCameraName,
	isp.TypeThermalCamera:    thermalCameraName,
	isp.TypePartialDischarge: partialDischargeName,
}

var modelTypeName = map[string]string{
	"1":  "区域主机及边缘节点装置模型",
	"2":  "机器人模型",
	"3":  "摄像机模型及硬盘录像机模型",
	"4":  "点位模型",
	"5":  "无人机模型及无人机机巢模型",
	"6":  "声纹模型",
	"7":  "任务文件",
	"8":  "检修区域配置文件",
	"9":  "地图文件",
	"10": "维护记录文件",
	"11": "联动配置文件",
	"12": "告警阈值模型",
}

var modelSyncCommandName = map[int32]string{
	1:  modelTypeName["1"],
	2:  modelTypeName["2"],
	3:  modelTypeName["3"],
	4:  modelTypeName["4"],
	5:  modelTypeName["5"],
	6:  modelTypeName["6"],
	7:  modelTypeName["7"],
	8:  modelTypeName["8"],
	9:  modelTypeName["9"],
	10: modelTypeName["10"],
	11: modelTypeName["11"],
	12: modelTypeName["12"],
}

// taskControlName 任务控制指令 → 中文名称。
var taskControlName = map[int32]string{
	isp.CommandTaskStart:  "启动",
	isp.CommandTaskPause:  "暂停",
	isp.CommandTaskResume: "继续",
	isp.CommandTaskStop:   "停止",
}
