package djisdk

// ==================== 设备属性（Properties） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/properties.html
// Topic: thing/product/{gateway_sn}/property/set | property/set_reply

const (
	// MethodPropertySet 设置设备属性。
	MethodPropertySet = "property_set"
)

// ==================== 设备管理（Device） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/device.html
// Topic: sys/product/{gateway_sn}/status
// Direction: up，设备拓扑更新由 status hooks 处理。

const (
	// MethodUpdateTopo 设备拓扑更新上行。
	MethodUpdateTopo = "update_topo"
)

// ==================== 组织管理（Organization） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/organization.html
// Topic: thing/product/{gateway_sn}/requests
// Direction: up，组织管理请求由 request hooks 处理。

const (
	// MethodAirportOrganizationBind 机场组织绑定请求上行。
	MethodAirportOrganizationBind = "airport_organization_bind"

	// MethodAirportOrganizationGet 获取机场绑定组织请求上行。
	MethodAirportOrganizationGet = "airport_organization_get"

	// MethodAirportBindStatus 机场绑定状态请求上行。
	MethodAirportBindStatus = "airport_bind_status"
)

// ==================== 直播功能（Live） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/live.html
// Topic: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备（Services），视频直播流控制

const (
	// MethodLiveStartPush 开始直播推流（Live Start Push）
	// 云平台 → 设备（Services），控制设备开始向指定地址推送直播流
	MethodLiveStartPush = "live_start_push"

	// MethodLiveStopPush 停止直播推流（Live Stop Push）
	// 云平台 → 设备（Services），控制设备停止直播推流
	MethodLiveStopPush = "live_stop_push"

	// MethodLiveSetQuality 设置直播质量（Live Set Quality）
	// 云平台 → 设备（Services），设置直播推流的画质参数
	MethodLiveSetQuality = "live_set_quality"

	// MethodLiveLensChange 切换直播镜头（Live Lens Change）
	// 云平台 → 设备（Services），切换直播推流使用的相机镜头
	MethodLiveLensChange = "live_lens_change"

	// MethodLiveCameraChange 切换直播相机（Live Camera Change）
	// 云平台 → 设备（Services），切换直播推流使用的相机负载
	MethodLiveCameraChange = "live_camera_change"
)

// ==================== 媒体功能（Media） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/media.html
// Topic: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备（Services），等待 services_reply。

const (
	// MethodMediaUploadFlighttaskMediaPrioritize 优先上传指定航线任务媒体（Upload Flighttask Media Prioritize）。
	// 云平台 → 设备（Services），按 flight_id 优先上传航线任务媒体。
	MethodMediaUploadFlighttaskMediaPrioritize = "upload_flighttask_media_prioritize"

	// MethodMediaFastUpload 快速上传指定媒体文件（Media Fast Upload）。
	// 云平台 → 设备（Services），按 file_id 快速上传单个媒体文件。
	MethodMediaFastUpload = "media_fast_upload"

	// MethodMediaHighestPriorityUploadFlighttask 最高优先级上传航线任务媒体（Highest Priority Upload Flighttask Media）。
	// 云平台 → 设备（Services），按 flight_id 最高优先级上传航线任务媒体。
	MethodMediaHighestPriorityUploadFlighttask = "highest_priority_upload_flighttask_media"
)

// ==================== 航线功能（Wayline） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/wayline.html
// Topic: thing/product/{gateway_sn}/services | events
// 方向: 云平台 <-> 设备

const (
	// MethodFlightTaskPrepare 航线任务准备（Flighttask Prepare）
	// 云平台 → 设备（Services），下发航线任务准备指令，设备进行航线预检查
	MethodFlightTaskPrepare = "flighttask_prepare"

	// MethodFlightTaskExecute 航线任务执行（Flighttask Execute）
	// 云平台 → 设备（Services），下发航线任务执行指令，执行已准备的航线
	MethodFlightTaskExecute = "flighttask_execute"

	// MethodFlightTaskCancel 航线任务取消（Flighttask Undo）
	// 云平台 → 设备（Services），取消未结束或未执行的航线任务
	MethodFlightTaskCancel = "flighttask_undo"

	// MethodFlightTaskPause 航线任务暂停（Flighttask Pause）
	// 云平台 → 设备（Services），暂停当前正在执行的航线任务
	MethodFlightTaskPause = "flighttask_pause"

	// MethodFlightTaskResume 航线任务恢复（Flighttask Recovery）
	// 云平台 → 设备（Services），恢复已暂停的航线任务
	MethodFlightTaskResume = "flighttask_recovery"

	// MethodFlightTaskStop 航线任务停止（Flighttask Stop）
	// 云平台 → 设备（Services），强制停止当前航线任务
	MethodFlightTaskStop = "flighttask_stop"

	// MethodReturnHome 一键返航（Return Home）
	// 云平台 → 设备（Services），控制飞行器执行返航操作
	MethodReturnHome = "return_home"

	// MethodReturnHomeCancelAutoReturn 取消自动返航（Return Home Cancel）
	// 云平台 → 设备（Services），取消飞行器自动返航
	MethodReturnHomeCancelAutoReturn = "return_home_cancel"

	// MethodReturnSpecificHome 返航至指定点（Return Specific Home）
	// 云平台 → 设备（Services），控制飞行器返航至指定的备降点
	MethodReturnSpecificHome = "return_specific_home"
)

// --- 航线管理 - Events ---
// Topic: thing/product/{gateway_sn}/events
// 方向: 设备 → 云平台（Events）

const (
	// MethodFlightTaskReady 航线任务就绪通知（Flighttask Ready）
	// 设备 → 云平台，设备通知云平台航线任务已准备就绪可执行
	MethodFlightTaskReady = "flighttask_ready"

	// MethodFlightTaskProgress 航线任务进度上报（Flighttask Progress）
	// 设备 → 云平台，设备周期性上报当前航线任务执行进度
	MethodFlightTaskProgress = "flighttask_progress"

	// MethodReturnHomeInfo 返航信息上报（Return Home Info）
	// 设备 → 云平台，设备上报返航相关状态信息
	MethodReturnHomeInfo = "return_home_info"
)

// ==================== HMS 管理（HMS） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/hms.html
// Topic: thing/product/{gateway_sn}/events
// 方向: 设备 → 云平台（Events），健康管理系统

const (
	// MethodHmsEventNotify HMS 健康事件通知（HMS Event Notify）
	// 设备 → 云平台，设备上报健康管理系统告警和状态事件
	MethodHmsEventNotify = "hms"
)

// ==================== 远程调试（Cmd） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/cmd.html
// Topic: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备（Services），机场远程调试控制指令

const (
	// MethodDebugModeOpen 开启调试模式（Debug Mode Open）
	// 云平台 → 设备，开启机场调试模式
	MethodDebugModeOpen = "debug_mode_open"

	// MethodDebugModeClose 关闭调试模式（Debug Mode Close）
	// 云平台 → 设备，关闭机场调试模式
	MethodDebugModeClose = "debug_mode_close"

	// MethodCoverOpen 打开舱盖（Cover Open）
	// 云平台 → 设备，控制机场舱盖打开
	MethodCoverOpen = "cover_open"

	// MethodCoverClose 关闭舱盖（Cover Close）
	// 云平台 → 设备，控制机场舱盖关闭
	MethodCoverClose = "cover_close"

	// MethodCoverForceClose 强制关闭舱盖（Cover Force Close）
	// 云平台 → 设备，强制关闭机场舱盖（忽略安全检查）
	MethodCoverForceClose = "cover_force_close"

	// MethodDroneOpen 开启飞行器（Drone Open）
	// 云平台 → 设备，远程开启飞行器电源
	MethodDroneOpen = "drone_open"

	// MethodDroneClose 关闭飞行器（Drone Close）
	// 云平台 → 设备，远程关闭飞行器电源
	MethodDroneClose = "drone_close"

	// MethodDeviceReboot 设备重启（Device Reboot）
	// 云平台 → 设备，远程重启机场设备
	MethodDeviceReboot = "device_reboot"

	// MethodChargeOpen 开启充电（Charge Open）
	// 云平台 → 设备，开启机场对飞行器的充电
	MethodChargeOpen = "charge_open"

	// MethodChargeClose 关闭充电（Charge Close）
	// 云平台 → 设备，关闭机场对飞行器的充电
	MethodChargeClose = "charge_close"

	// MethodDroneFormat 飞行器存储格式化（Drone Format）
	// 云平台 → 设备，格式化飞行器机载存储
	MethodDroneFormat = "drone_format"

	// MethodDeviceFormat 机场存储格式化（Device Format）
	// 云平台 → 设备，格式化机场本地存储
	MethodDeviceFormat = "device_format"

	// MethodSupplementLightOpen 打开补光灯（Supplement Light Open）
	// 云平台 → 设备，开启机场补光灯
	MethodSupplementLightOpen = "supplement_light_open"

	// MethodSupplementLightClose 关闭补光灯（Supplement Light Close）
	// 云平台 → 设备，关闭机场补光灯
	MethodSupplementLightClose = "supplement_light_close"

	// MethodBatteryStoreModeSwitch 电池保养存储模式切换（Battery Store Mode Switch）
	// 云平台 → 设备，切换电池存储保养模式
	MethodBatteryStoreModeSwitch = "battery_store_mode_switch"

	// MethodAlarmStateSwitch 声光报警开关切换（Alarm State Switch）
	// 云平台 → 设备，切换机场声光报警状态
	MethodAlarmStateSwitch = "alarm_state_switch"

	// MethodBatteryMaintenanceSwitch 电池保养开关切换（Battery Maintenance Switch）
	// 云平台 → 设备，切换电池保养功能开关
	MethodBatteryMaintenanceSwitch = "battery_maintenance_switch"

	// MethodAirConditionerModeSwitch 空调模式切换（Air Conditioner Mode Switch）
	// 云平台 → 设备，切换机场空调工作模式
	MethodAirConditionerModeSwitch = "air_conditioner_mode_switch"
)

// ==================== 固件升级（Firmware） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/firmware.html
// Topic: thing/product/{gateway_sn}/services | events
// OTA 固件升级管理

const (
	// MethodOtaCreate 创建固件升级任务（OTA Create）
	// 云平台 → 设备（Services），下发固件升级任务
	MethodOtaCreate = "ota_create"

	// MethodOtaProgress 固件升级进度上报（OTA Progress）
	// 设备 → 云平台（Events），设备周期性上报固件升级进度
	MethodOtaProgress = "ota_progress"
)

// ==================== 远程日志（Log） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/log.html
// Topic: thing/product/{gateway_sn}/services | events
// 方向: fileupload_* 控制为云平台 → 设备（Services），fileupload_progress 为设备 → 云平台（Events）。

const (
	// MethodRemoteLogFileList 查询可上传的远程日志文件列表（Fileupload List）。
	// 云平台 → 设备（Services），按目标设备和模块查询日志文件。
	MethodRemoteLogFileList = "fileupload_list"

	// MethodRemoteLogFileUploadStart 开始上传远程日志文件（Fileupload Start）。
	// 云平台 → 设备（Services），下发待上传日志文件列表。
	MethodRemoteLogFileUploadStart = "fileupload_start"

	// MethodRemoteLogFileUploadUpdate 更新远程日志文件上传任务（Fileupload Update）。
	// 云平台 → 设备（Services），更新正在执行的日志上传文件列表。
	MethodRemoteLogFileUploadUpdate = "fileupload_update"

	// MethodRemoteLogFileUploadCancel 取消远程日志文件上传任务（Fileupload Cancel）。
	// 云平台 → 设备（Services），取消指定远程日志上传任务。
	MethodRemoteLogFileUploadCancel = "fileupload_cancel"

	// MethodRemoteLogFileUploadProgress 远程日志上传进度上报（Fileupload Progress）。
	// 设备 → 云平台（Events），设备上报远程日志文件上传进度。
	MethodRemoteLogFileUploadProgress = "fileupload_progress"
)

// ==================== 配置更新（Config） ====================
// Topic: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备（Services），等待 services_reply。

const (
	// MethodConfigUpdate 下发设备配置更新（Config Update）。
	// 云平台 → 设备（Services），向设备下发配置更新内容。
	MethodConfigUpdate = "config_update"
)

// ==================== 指令飞行（DRC） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html
// services 通道：drc_mode_*、飞行控制权、FlyTo 等发布到 thing/product/{gateway_sn}/services 并等待 services_reply。
// drc/down 通道：stick_control、heart_beat、drone_emergency_stop 等实时控制消息即发即忘。
// drc/up 通道：设备回传 stick_control 回执、heart_beat、hsi_info_push、delay_info_push、osd_info_push 等状态。
// 方向: services 与 drc/down 为云平台 → 设备；drc/up 为设备 → 云平台。

const (
	// --- 飞行控制指令（DRC Commands） ---

	// MethodFlightAuthorityGrab 飞行控制权抢夺（Flight Authority Grab）
	// 云平台 → 设备，抢夺飞行器飞行控制权
	MethodFlightAuthorityGrab = "flight_authority_grab"

	// MethodPayloadAuthorityGrab 负载控制权抢夺（Payload Authority Grab）
	// 云平台 → 设备，抢夺负载设备控制权
	MethodPayloadAuthorityGrab = "payload_authority_grab"

	// MethodDrcModeEnter 进入 DRC 模式（DRC Mode Enter）
	// 云平台 → 设备，进入指令飞行（DRC）控制模式
	MethodDrcModeEnter = "drc_mode_enter"

	// MethodDrcModeExit 退出 DRC 模式（DRC Mode Exit）
	// 云平台 → 设备，退出指令飞行（DRC）控制模式
	MethodDrcModeExit = "drc_mode_exit"

	// MethodStickControl DRC 杆量控制，使用 drc/down 即发即忘下发，设备可经 drc/up 回执。
	MethodStickControl = "stick_control"

	// MethodDroneEmergencyStop 飞行器紧急停桨，通过 drc/down 即发即忘地下发 DRC 通道急停。
	MethodDroneEmergencyStop = "drone_emergency_stop"

	// --- Flyto 指令（Flyto Commands） ---

	// MethodFlyToPoint 飞向指定点（Fly To Point）
	// 云平台 → 设备，控制飞行器飞往指定坐标点
	MethodFlyToPoint = "fly_to_point"

	// MethodFlyToPointStop 停止飞向指定点（Fly To Point Stop）
	// 云平台 → 设备，停止当前飞向指定点的飞行任务
	MethodFlyToPointStop = "fly_to_point_stop"

	// --- 一键起飞指令（One-key Taking Off Commands） ---

	// MethodTakeoffToPoint 一键起飞至指定点（Takeoff To Point）
	// 云平台 → 设备，控制飞行器起飞并飞往指定坐标点
	MethodTakeoffToPoint = "takeoff_to_point"
)

// --- DRC 通道 method（drc/down 与 drc/up，与 services 通道独立）---
const (
	// MethodDrcHeartBeat DRC 心跳，drc/down 下发，drc/up 回传。
	MethodDrcHeartBeat = "heart_beat"
	// MethodDrcHsiInfoPush 设备经 drc/up 上报避障/水平态势。
	MethodDrcHsiInfoPush = "hsi_info_push"
	// MethodDrcDelayInfoPush 设备经 drc/up 上报图传链路时延。
	MethodDrcDelayInfoPush = "delay_info_push"
	// MethodDrcOsdInfoPush 设备经 drc/up 上报高频 OSD。
	MethodDrcOsdInfoPush = "osd_info_push"
)

// ==================== 指令飞行（DRC）相机/云台控制 ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html
// Topic: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备（Services），相机和云台远程控制指令

const (
	// MethodCameraModeSwitch 相机模式切换（Camera Mode Switch）
	// 云平台 → 设备，切换相机工作模式（拍照/录像等）
	MethodCameraModeSwitch = "camera_mode_switch"

	// MethodCameraPhotoTake 拍照（Camera Photo Take）
	// 云平台 → 设备，控制相机执行拍照
	MethodCameraPhotoTake = "camera_photo_take"

	// MethodCameraPhotoStop 停止拍照（Camera Photo Stop）
	// 云平台 → 设备，停止相机连续拍照（如定时拍照等模式）
	MethodCameraPhotoStop = "camera_photo_stop"

	// MethodCameraRecordingStart 开始录像（Camera Recording Start）
	// 云平台 → 设备，控制相机开始录像
	MethodCameraRecordingStart = "camera_recording_start"

	// MethodCameraRecordingStop 停止录像（Camera Recording Stop）
	// 云平台 → 设备，控制相机停止录像
	MethodCameraRecordingStop = "camera_recording_stop"

	// MethodCameraFocalLengthSet 相机焦距设置（Camera Focal Length Set）
	// 云平台 → 设备，设置相机变焦焦距
	MethodCameraFocalLengthSet = "camera_focal_length_set"

	// MethodGimbalReset 云台重置（Gimbal Reset）
	// 云平台 → 设备，重置云台角度至默认位置
	MethodGimbalReset = "gimbal_reset"

	// MethodCameraAim 相机框选/指点目标（Camera Aim）
	// 云平台 → 设备，控制相机视角指向目标
	MethodCameraAim = "camera_aim"

	// MethodCameraPointFocusAction 相机指点对焦（Camera Point Focus Action）
	// 云平台 → 设备，控制相机在指定屏幕坐标点执行对焦
	MethodCameraPointFocusAction = "camera_point_focus_action"

	// MethodCameraScreenSplit 相机画面分屏（Camera Screen Split）
	// 云平台 → 设备，控制相机画面分屏显示
	MethodCameraScreenSplit = "camera_screen_split"

	// MethodCameraPhotoStorageSet 设置拍照存储位置（Camera Photo Storage Set）
	// 云平台 → 设备，设置相机拍照存储位置
	MethodCameraPhotoStorageSet = "camera_photo_storage_set"

	// MethodCameraVideoStorageSet 设置录像存储位置（Camera Video Storage Set）
	// 云平台 → 设备，设置相机录像存储位置
	MethodCameraVideoStorageSet = "camera_video_storage_set"

	// MethodCameraLookAt 相机朝向指定坐标（Camera Look At）
	// 云平台 → 设备，控制相机持续朝向指定地理坐标
	MethodCameraLookAt = "camera_look_at"

	// MethodCameraScreenDrag 相机画面拖动（Camera Screen Drag）
	// 云平台 → 设备，通过屏幕拖拽方式控制云台转动
	MethodCameraScreenDrag = "camera_screen_drag"

	// MethodCameraIrMeteringPoint 红外测温点设置（Camera IR Metering Point）
	// 云平台 → 设备，设置红外相机指定点测温
	MethodCameraIrMeteringPoint = "camera_ir_metering_point"

	// MethodCameraIrMeteringArea 红外区域测温设置（Camera IR Metering Area）
	// 云平台 → 设备，设置红外相机指定区域测温
	MethodCameraIrMeteringArea = "camera_ir_metering_area"
)

// ==================== 自定义飞行区（Custom Fly Region） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/custom-fly-region.html
// Topic: thing/product/{gateway_sn}/services

const (
	// MethodFlightAreasUpdate 触发自定义飞行区文件更新。
	MethodFlightAreasUpdate = "flight_areas_update"

	// MethodFlightAreasGet 自定义飞行区文件获取请求上行。
	MethodFlightAreasGet = "flight_areas_get"
)

// ==================== PSDK 功能与互联互通（PSDK / PSDK Transmit） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/psdk.html
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/psdk-transmit-custom-data.html
// Topic: thing/product/{gateway_sn}/services | events

const (
	// MethodPsdkFloatUp PSDK UI 资源上传（PSDK UI Resource Upload）
	// 云平台 → 设备（Services），上传 PSDK 浮窗 UI 资源文件
	MethodPsdkFloatUp = "psdk_ui_resource_upload"

	// MethodCustomDataTransmissionToPsdk 自定义数据透传至 PSDK（Custom Data Transmission To PSDK）
	// 云平台 → 设备（Services），将自定义消息透传推送到 PSDK 负载设备
	MethodCustomDataTransmissionToPsdk = "custom_data_transmission_to_psdk"

	// MethodCustomDataTransmissionFromPsdk PSDK 自定义数据上报（Custom Data Transmission From PSDK）
	// 设备 → 云平台（Events），PSDK 负载设备向云平台上报自定义消息
	MethodCustomDataTransmissionFromPsdk = "custom_data_transmission_from_psdk"
)

// ==================== ESDK 互联互通（ESDK Transmit） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/esdk-transmit-custom-data.html
// Topic: thing/product/{gateway_sn}/services | events

const (
	// MethodCustomDataTransmissionToEsdk 自定义数据透传至 ESDK。
	MethodCustomDataTransmissionToEsdk = "custom_data_transmission_to_esdk"

	// MethodCustomDataTransmissionFromEsdk ESDK 自定义数据上报。
	MethodCustomDataTransmissionFromEsdk = "custom_data_transmission_from_esdk"
)

// ==================== 远程解禁（Flysafe） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/flysafe.html
// Topic: thing/product/{gateway_sn}/services

const (
	// MethodUnlockLicenseSwitch 启用或禁用设备的单个解禁证书。
	MethodUnlockLicenseSwitch = "unlock_license_switch"

	// MethodUnlockLicenseUpdate 更新设备的解禁证书。
	MethodUnlockLicenseUpdate = "unlock_license_update"

	// MethodUnlockLicenseList 获取设备的解禁证书列表。
	MethodUnlockLicenseList = "unlock_license_list"
)

// ==================== AirSense ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/airsense.html
// 该模块当前仅包含设备上行告警事件。

// ==================== 远程控制（Remote Control） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/remote-control.html
// Topic: thing/product/{gateway_sn}/services

const (
	// MethodDrcForceLanding 强制降落。
	MethodDrcForceLanding = "drc_force_landing"

	// MethodDrcEmergencyLanding 紧急降落。
	MethodDrcEmergencyLanding = "drc_emergency_landing"

	// MethodDrcLinkageZoomSet 设置红外联动变焦状态。
	MethodDrcLinkageZoomSet = "drc_linkage_zoom_set"

	// MethodDrcVideoResolutionSet 设置视频分辨率。
	MethodDrcVideoResolutionSet = "drc_video_resolution_set"

	// MethodDrcIntervalPhotoSet 设置定时拍参数。
	MethodDrcIntervalPhotoSet = "drc_interval_photo_set"

	// MethodDrcInitialStateSubscribe 订阅 DRC 初始状态。
	MethodDrcInitialStateSubscribe = "drc_initial_state_subscribe"

	// MethodDrcNightLightsStateSet 设置夜航灯状态。
	MethodDrcNightLightsStateSet = "drc_night_lights_state_set"

	// MethodDrcStealthStateSet 设置隐蔽模式状态。
	MethodDrcStealthStateSet = "drc_stealth_state_set"

	// MethodDrcCameraApertureValueSet 设置相机光圈。
	MethodDrcCameraApertureValueSet = "drc_camera_aperture_value_set"

	// MethodDrcCameraShutterSet 设置相机快门。
	MethodDrcCameraShutterSet = "drc_camera_shutter_set"

	// MethodDrcCameraIsoSet 设置相机 ISO。
	MethodDrcCameraIsoSet = "drc_camera_iso_set"

	// MethodDrcCameraMechanicalShutterSet 设置机械快门。
	MethodDrcCameraMechanicalShutterSet = "drc_camera_mechanical_shutter_set"

	// MethodDrcCameraDewarpingSet 设置镜头去畸变。
	MethodDrcCameraDewarpingSet = "drc_camera_dewarping_set"
)
