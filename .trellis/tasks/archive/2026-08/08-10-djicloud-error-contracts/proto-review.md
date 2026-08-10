# Proto Interface Review

## Decision Summary

`CommonRes` is retained. It is not a generic error envelope; it is the DJI command result envelope:

- `code`: DJI command result status (`0` success, `-1` translated device failure)
- `message`: device/application-readable command result
- `tid`: MQTT transaction ID
- `reason_code`: original DJI device result code

Platform execution failures remain gRPC errors and must not be encoded as `CommonRes{code:-1}`.

## RPC Matrix

| Proto section | RPCs | Response rule |
|---|---|---|
| Properties | `PropertySet` | `CommonRes`; device ACK failure wrapped, local failure gRPC error |
| Live | `LiveStartPush`, `LiveStopPush`, `LiveSetQuality`, `LiveLensChange`, `LiveCameraChange` | Same DJI ACK rule |
| Media | `MediaUploadFlighttaskMediaPrioritize`, `MediaFastUpload`, `MediaHighestPriorityUploadFlighttask` | Same DJI ACK rule |
| Wayline | `FlightTaskPrepare`, `FlightTaskExecute`, `FlightTaskUndo`, `PauseFlightTask`, `FlightTaskRecovery`, `StopFlightTask`, `ReturnHome`, `ReturnHomeCancelAutoReturn`, `ReturnSpecificHome` | Same DJI ACK rule |
| Cmd | `DebugModeOpen`, `DebugModeClose`, `CoverOpen`, `CoverClose`, `CoverForceClose`, `DroneOpen`, `DroneClose`, `DeviceReboot`, `ChargeOpen`, `ChargeClose`, `DroneFormat`, `DeviceFormat`, `SupplementLightOpen`, `SupplementLightClose`, `BatteryStoreModeSwitch`, `AlarmStateSwitch`, `AirConditionerModeSwitch`, `BatteryMaintenanceSwitch` | Same DJI ACK rule |
| Firmware | `OtaCreate` | Same DJI ACK rule |
| Log | `RemoteLogFileList`, `RemoteLogFileUploadStart`, `RemoteLogFileUploadUpdate`, `RemoteLogFileUploadCancel` | Same DJI ACK rule |
| Config | `ConfigUpdate` | Same DJI ACK rule |
| DRC services | `FlightAuthorityGrab`, `PayloadAuthorityGrab`, `FlyToPoint`, `FlyToPointStop`, `TakeoffToPoint`, `CameraModeSwitch`, `CameraPhotoTake`, `CameraPhotoStop`, `CameraRecordingStart`, `CameraRecordingStop`, `CameraFocalLengthSet`, `GimbalReset`, `CameraAim`, `CameraPointFocusAction`, `CameraScreenSplit`, `CameraPhotoStorageSet`, `CameraVideoStorageSet`, `CameraLookAt`, `CameraScreenDrag`, `CameraIrMeteringPoint`, `CameraIrMeteringArea`, `DrcModeEnter`, `DrcModeExit` | `CommonRes`; only the first 21 are DJI standard DRC services, the last 2 are lifecycle commands but use the same ACK contract |
| Custom fly region trigger | `FlightAreasUpdate` | DJI standard ACK, `CommonRes` |
| PSDK/ESDK/Flysafe | `PsdkUIResourceUpload`, `CustomDataTransmissionToPsdk`, `CustomDataTransmissionToEsdk`, `UnlockLicenseSwitch`, `UnlockLicenseUpdate`, `UnlockLicenseList` | `CommonRes`; device ACK failure wrapped |
| DRC fire-and-forget | `DroneEmergencyStop`, `StickControl`, `DrcForceLanding`, `DrcEmergencyLanding`, `DrcLinkageZoomSet`, `DrcVideoResolutionSet`, `DrcIntervalPhotoSet`, `DrcInitialStateSubscribe`, `DrcNightLightsStateSet`, `DrcStealthStateSet`, `DrcCameraApertureValueSet`, `DrcCameraShutterSet`, `DrcCameraIsoSet`, `DrcCameraMechanicalShutterSet`, `DrcCameraDewarpingSet` | Existing `*Res{seq}`; publish/sequence failure gRPC error; no device ACK fields |
| Platform query/state | `IsDeviceOnline`, `ListDevices`, `GetDeviceDetail`, `GetDeviceOsdSnapshot`, `GetDeviceStateSnapshot`, `ListHmsAlerts`, `ListFlightTaskProgress`, `GetFlightTaskProgressLast`, `QueryDrcStatus`, `ListFlyRegions`, `ListFlyRegionSyncStatus` | Existing data response; all platform failures gRPC error |
| Platform acknowledgement | `AckHmsAlert` | Change `CommonRes` to empty `AckHmsAlertRes`; all failures gRPC error |
| Platform fly-region orchestration | `SubmitCustomFlyRegion`, `DeleteCustomFlyRegion`, `DeleteCustomFlyRegionByFileId` | Retain result response with `tid/reason_code`; local phase gRPC error, DJI `FlightAreasUpdate` ACK failure wrapped |

## Dedicated Response Naming Review

Except for the intentionally shared `CommonRes`, a dedicated response type must use `<RpcName>Res`.

| RPC | Current response | Target response |
|---|---|---|
| `IsDeviceOnline` | `DeviceOnlineRes` | `IsDeviceOnlineRes` |
| `GetDeviceDetail` | `DeviceDetailRes` | `GetDeviceDetailRes` |
| `GetDeviceOsdSnapshot` | `DeviceOsdSnapshotRes` | `GetDeviceOsdSnapshotRes` |
| `GetDeviceStateSnapshot` | `DeviceStateSnapshotRes` | `GetDeviceStateSnapshotRes` |
| `GetFlightTaskProgressLast` | `FlightTaskProgressLastRes` | `GetFlightTaskProgressLastRes` |
| `QueryDrcStatus` | `DrcStatusRes` | `QueryDrcStatusRes` |
| `AckHmsAlert` | `CommonRes` | new empty `AckHmsAlertRes` |

All other dedicated response types already match their RPC names. The six renames preserve existing fields and field numbers.

## Request Naming Findings

The request side has intentional shared payload types that do not match every RPC name:

- `MediaUploadFlighttaskMediaPrioritize` and `MediaHighestPriorityUploadFlighttask` share `MediaFlighttaskReq`.
- `RemoteLogFileUploadStart`, `RemoteLogFileUploadUpdate` and `RemoteLogFileUploadCancel` share `RemoteLogFileUploadReq`.
- `BatteryStoreModeSwitch` and `BatteryMaintenanceSwitch` share `BatteryStoreModeReq`.

These are structurally reusable payloads, not response naming defects. Splitting them into method-paired request messages would improve mechanical naming consistency but duplicate identical contracts.

## Required Proto Changes

1. Add `AckHmsAlertRes` as an empty platform response.
2. Change `rpc AckHmsAlert(AckHmsAlertReq) returns (AckHmsAlertRes);`.
3. Rename the six mismatched dedicated platform response types listed above.
4. Rewrite `CommonRes` comments to state that it is only for DJI command results, not platform errors.
5. Rewrite custom fly-region response comments to document the mixed contract.
6. Keep all existing field numbers in retained messages.

## Explicit Non-Changes

- Do not remove `code`, `message`, `tid` or `reason_code` from `CommonRes`.
- Do not change standard DJI ACK RPCs to direct gRPC errors for device business rejection.
- Do not add `code/message` to data-only platform query responses.
