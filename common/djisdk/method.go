package djisdk

// ==================== 一、航线管理（Wayline Management） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/wayline.html
// Topic: thing/product/{gateway_sn}/services | events
// 方向: 云平台 <-> 设备

const (
	// MethodFlightTaskPrepare 航线任务准备（Flighttask Prepare）
	// 云平台 → 设备，下发航线任务前的准备指令，设备进行航线预检查
	MethodFlightTaskPrepare = "flighttask_prepare"

	// MethodFlightTaskExecute 航线任务执行（Flighttask Execute）
	// 云平台 → 设备，下发航线任务执行指令，设备开始执行已准备的航线
	MethodFlightTaskExecute = "flighttask_execute"

	// MethodFlightTaskCancel 航线任务取消（Flighttask Undo）
	// 云平台 → 设备，取消已下发但未执行完成的航线任务
	MethodFlightTaskCancel = "flighttask_undo"

	// MethodFlightTaskPause 航线任务暂停（Flighttask Pause）
	// 云平台 → 设备，暂停当前正在执行的航线任务
	MethodFlightTaskPause = "flighttask_pause"

	// MethodFlightTaskResume 航线任务恢复（Flighttask Recovery）
	// 云平台 → 设备，恢复已暂停的航线任务继续执行
	MethodFlightTaskResume = "flighttask_recovery"

	// MethodFlightTaskStop 航线任务停止（Flighttask Stop）
	// 云平台 → 设备，强制停止当前航线任务
	MethodFlightTaskStop = "flighttask_stop"

	// MethodReturnHome 一键返航（Return Home）
	// 云平台 → 设备，控制飞行器执行返航操作
	MethodReturnHome = "return_home"

	// MethodReturnHomeCancelAutoReturn 取消自动返航（Return Home Cancel）
	// 云平台 → 设备，取消飞行器自动返航
	MethodReturnHomeCancelAutoReturn = "return_home_cancel"

	// MethodReturnSpecificHome 返航至指定点（Return Specific Home）
	// 云平台 → 设备，控制飞行器返航至指定的备降点
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

// ==================== 二、PSDK 自定义数据透传（PSDK Custom Data Transmission） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/psdk-transmit-custom-data.html
// Topic: thing/product/{gateway_sn}/services | events

const (
	// MethodPsdkWrite PSDK 数据写入（PSDK Write）
	// 云平台 → 设备（Services），向 PSDK 负载设备写入数据
	MethodPsdkWrite = "psdk_write"

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

// ==================== 三、指令飞行控制（Live Flight Controls / DRC） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html
// Topic: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备（Services）

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

	// MethodDroneControl 飞行器虚拟摇杆控制（Drone Control）
	// 云平台 → 设备，通过虚拟摇杆实时控制飞行器姿态和运动
	MethodDroneControl = "drone_control"

	// MethodDroneEmergencyStop 飞行器紧急停桨（Drone Emergency Stop）
	// 云平台 → 设备，紧急停止飞行器电机（慎用，飞行器将直接坠落）
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

// ==================== 四、远程调试 - 机巢控制（Remote Debug） ====================
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

// ==================== 五、相机/云台控制（Camera & Gimbal） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/drc.html
// Topic: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备（Services），相机和云台远程控制指令

const (
	// MethodCameraModeSwitchCamera 相机模式切换（Camera Mode Switch）
	// 云平台 → 设备，切换相机工作模式（拍照/录像等）
	MethodCameraModeSwitchCamera = "camera_mode_switch"

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

	// MethodCameraAimCamera 相机对准目标点（Camera Aim）
	// 云平台 → 设备，控制相机/云台对准指定目标坐标
	MethodCameraAimCamera = "camera_aim"

	// MethodCameraFocalLengthSet 相机焦距设置（Camera Focal Length Set）
	// 云平台 → 设备，设置相机变焦焦距
	MethodCameraFocalLengthSet = "camera_focal_length_set"

	// MethodGimbalReset 云台重置（Gimbal Reset）
	// 云平台 → 设备，重置云台角度至默认位置
	MethodGimbalReset = "gimbal_reset"

	// MethodCameraPointFocusAction 相机指点对焦（Camera Point Focus Action）
	// 云平台 → 设备，控制相机在指定屏幕坐标点执行对焦
	MethodCameraPointFocusAction = "camera_point_focus_action"

	// MethodCameraScreenSplit 相机画面分屏（Camera Screen Split）
	// 云平台 → 设备，控制相机画面分屏显示
	MethodCameraScreenSplit = "camera_screen_split"

	// MethodCameraPhotoStorageSet 照片存储设置（Photo Storage Set）
	// 云平台 → 设备，设置拍照存储位置
	MethodCameraPhotoStorageSet = "photo_storage_set"

	// MethodCameraVideoStorageSet 视频存储设置（Video Storage Set）
	// 云平台 → 设备，设置录像存储位置
	MethodCameraVideoStorageSet = "video_storage_set"

	// MethodCameraLookAt 相机朝向指定坐标（Camera Look At）
	// 云平台 → 设备，控制相机持续朝向指定地理坐标
	MethodCameraLookAt = "camera_look_at"

	// MethodCameraScreenDrag 相机画面拖动（Camera Screen Drag）
	// 云平台 → 设备，通过屏幕拖拽方式控制云台转动
	MethodCameraScreenDrag = "camera_screen_drag"

	// MethodCameraIrMeteringPoint 红外测温点测温（IR Metering Point）
	// 云平台 → 设备，设置红外相机指定点测温
	MethodCameraIrMeteringPoint = "ir_metering_point"

	// MethodCameraIrMeteringArea 红外测温区域测温（IR Metering Area）
	// 云平台 → 设备，设置红外相机指定区域测温
	MethodCameraIrMeteringArea = "ir_metering_area"
)

// ==================== 六、直播管理（Live） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/live.html
// Topic: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备（Services），视频直播流控制

const (
	// MethodLiveStartPush 开始直播推流（Live Start Push）
	// 云平台 → 设备，控制设备开始向指定地址推送直播流
	MethodLiveStartPush = "live_start_push"

	// MethodLiveStopPush 停止直播推流（Live Stop Push）
	// 云平台 → 设备，控制设备停止直播推流
	MethodLiveStopPush = "live_stop_push"

	// MethodLiveSetQuality 设置直播质量（Live Set Quality）
	// 云平台 → 设备，设置直播推流的画质参数
	MethodLiveSetQuality = "live_set_quality"

	// MethodLiveLensChange 切换直播镜头（Live Lens Change）
	// 云平台 → 设备，切换直播推流使用的相机镜头
	MethodLiveLensChange = "live_lens_change"

	// MethodLiveCameraChange 切换直播相机（Live Camera Change）
	// 云平台 → 设备，切换直播推流使用的相机（如机巢内/外摄像头）
	MethodLiveCameraChange = "live_camera_change"
)

// ==================== 七、属性设置（Property Set） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/property.html
// Topic: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备（Services）

const (
	// MethodPropertySet 设备属性设置（Property Set）
	// 云平台 → 设备（Services），远程设置设备属性参数
	MethodPropertySet = "property_set"
)

// ==================== 八、固件管理（Firmware） ====================
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

// ==================== 九、设备管理（Device Management） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/device.html
// Topic: thing/product/{gateway_sn}/services | events
// 设备拓扑与属性管理

const (
	// MethodUpdateTopo 更新设备拓扑（Update Topo）
	// 设备 → 云平台（Events），设备上报拓扑结构变化（如飞行器上下线）
	MethodUpdateTopo = "update_topo"
)

// ==================== 十、HMS 健康管理（HMS） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/hms.html
// Topic: thing/product/{gateway_sn}/events
// 方向: 设备 → 云平台（Events），健康管理系统

const (
	// MethodHmsEventNotify HMS 健康事件通知（HMS Event Notify）
	// 设备 → 云平台，设备上报健康管理系统告警和状态事件
	MethodHmsEventNotify = "hms"
)

// ==================== 十一、模拟器（Simulator） ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/api-reference/dock-to-cloud/mqtt/dock/dock3/simulator.html
// Topic: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备（Services），模拟器任务控制

const (
	// MethodSimulateMission 模拟任务执行（Simulate Mission）
	// 云平台 → 设备，下发模拟飞行任务，用于仿真调试
	MethodSimulateMission = "simulate_mission"
)
