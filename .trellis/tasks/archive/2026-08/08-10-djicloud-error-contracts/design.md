# Technical Design

## Contract Categories

### DJI ACK response

`PropertySet` and the DJI standard downlink methods under Live, Media, Wayline, Cmd, Firmware, Log, Config, DRC services, PSDK, ESDK and Flysafe call `SendCommand`-style APIs and wait for `services_reply`/property reply. Their `CommonRes` is the application result of the DJI command. A non-zero DJI result is not a gRPC transport failure and must retain `tid` and `reason_code`.

`DrcModeEnter` and `DrcModeExit` are also in this category because they use the `services` topic and wait for a device reply.

### DRC fire-and-forget response

`DroneEmergencyStop`, `StickControl`, `DrcForceLanding`, `DrcEmergencyLanding`, `DrcLinkageZoomSet`, `DrcVideoResolutionSet`, `DrcIntervalPhotoSet`, `DrcInitialStateSubscribe`, `DrcNightLightsStateSet`, `DrcStealthStateSet`, `DrcCameraApertureValueSet`, `DrcCameraShutterSet`, `DrcCameraIsoSet`, `DrcCameraMechanicalShutterSet` and `DrcCameraDewarpingSet` publish to `drc/down`. Their response only confirms local publication/sequence allocation through `seq`; no device business result is available.

### Platform response

The platform-owned RPCs are `IsDeviceOnline`, `ListDevices`, `GetDeviceDetail`, `GetDeviceOsdSnapshot`, `GetDeviceStateSnapshot`, `ListHmsAlerts`, `AckHmsAlert`, `ListFlightTaskProgress`, `GetFlightTaskProgressLast`, `QueryDrcStatus`, `SubmitCustomFlyRegion`, `DeleteCustomFlyRegion`, `DeleteCustomFlyRegionByFileId`, `ListFlyRegions` and `ListFlyRegionSyncStatus`.

Platform query responses keep their existing fields, but dedicated response type names follow `<RpcName>Res`. The following types are renamed without changing fields: `DeviceOnlineRes` to `IsDeviceOnlineRes`, `DeviceDetailRes` to `GetDeviceDetailRes`, `DeviceOsdSnapshotRes` to `GetDeviceOsdSnapshotRes`, `DeviceStateSnapshotRes` to `GetDeviceStateSnapshotRes`, `FlightTaskProgressLastRes` to `GetFlightTaskProgressLastRes`, and `DrcStatusRes` to `QueryDrcStatusRes`. `AckHmsAlert` uses a new empty `AckHmsAlertRes` success response rather than `CommonRes`, because it has no DJI command result to expose.

The three custom fly-region write responses remain result wrappers with `tid` and `reason_code`. They are platform orchestration APIs with a final DJI ACK phase, so the response must distinguish a persisted/OSS-completed operation from a DJI rejection.

## Error Boundary

- Request validation, local database, OSS, configuration, publish, timeout and DRC lifecycle errors return `tool.NewErrorByPbCode*` errors.
- A typed `djisdk.DJIError` returned after a device ACK is converted to `CommonRes` (or the equivalent custom fly-region response), including `ReasonCode` and `Tid`.
- `CommonRes.Code` remains a DJI command result field, not a generic platform error code.

## Compatibility

This task permits the requested breaking proto changes for dedicated response type normalization and `AckHmsAlert`. Published field numbers in all retained messages remain unchanged. Generated Go files and server signatures are regenerated from the proto source.

## Data Flow

`djicloud.proto` -> `app/djicloud/gen.sh` -> generated client/server types -> `app/djicloud/internal/server` -> Logic -> `common/djisdk`/DB/OSS. All direct references to `AckHmsAlert` and changed response types must be searched and regenerated together.
