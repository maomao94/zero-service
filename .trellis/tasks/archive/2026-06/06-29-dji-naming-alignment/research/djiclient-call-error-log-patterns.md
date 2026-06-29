# Research: DjiClient Call + Error Log Patterns

- **Query**: Find ALL files in `app/djicloud/internal/logic/` that call a method on `l.svcCtx.DjiClient.` and then log the error with `l.Errorf` or similar.
- **Scope**: Internal
- **Date**: 2026-06-29

## Findings

Two patterns found across 90+ logic files:

### Pattern A: `l.Errorf` after DjiClient call (72 files)
The SDK call returns `(tid, err)`, if `err != nil`, log via `l.Errorf(...)` then return `errRes(tid, err)`.

### Pattern B: No `l.Errorf` — uses `errRes` helper directly (11 files)
The SDK call returns `(tid, err)`, if `err != nil`, return `errRes(tid, err)` directly without `l.Errorf`. The `errRes` helper (defined in `helper.go:23`) just wraps the error into a `CommonRes` response without logging at error level.

### Special: No error path (1 file)
`querydrcstatuslogic.go` — `DrcStatus()` never returns an error, so no error handling is needed.

### Special: Returns raw `error` (1 file)
`stickcontrollogic.go` — returns `nil, err` (propagating error to caller) instead of `errRes(tid, err)`.

---

## Complete File Listing

### Files with `l.Errorf` after DjiClient SDK call

| # | File | SDK Call (line) | Error Log (line) |
|---|---|---|---|
| 1 | `airconditionermodeswitchlogic.go` | L28: `l.svcCtx.DjiClient.AirConditionerModeSwitch(...)` | L30: `l.Errorf("[remote-debug] air conditioner mode switch failed: %v", err)` |
| 2 | `alarmstateswitchlogic.go` | L28: `l.svcCtx.DjiClient.AlarmStateSwitch(...)` | L30: `l.Errorf("[remote-debug] alarm state switch failed: %v", err)` |
| 3 | `batterymaintenanceswitchlogic.go` | L28: `l.svcCtx.DjiClient.BatteryMaintenanceSwitch(...)` | L30: `l.Errorf("[remote-debug] battery maintenance switch failed: %v", err)` |
| 4 | `batterystoremodeswitchlogic.go` | L28: `l.svcCtx.DjiClient.BatteryStoreModeSwitch(...)` | L30: `l.Errorf("[remote-debug] battery store mode switch failed: %v", err)` |
| 5 | `cameraaimlogic.go` | L35: `l.svcCtx.DjiClient.CameraAim(...)` | L37: `l.Errorf("[camera] camera aim failed: %v", err)` |
| 6 | `camerafocallengthsetlogic.go` | L33: `l.svcCtx.DjiClient.CameraFocalLengthSet(...)` | L35: `l.Errorf("[camera] camera focal length set failed: %v", err)` |
| 7 | `camerairmeteringarealogic.go` | L35: `l.svcCtx.DjiClient.CameraIrMeteringArea(...)` | L37: `l.Errorf("[camera] camera ir metering area failed: %v", err)` |
| 8 | `camerairmeteringpointlogic.go` | L33: `l.svcCtx.DjiClient.CameraIrMeteringPoint(...)` | L35: `l.Errorf("[camera] camera ir metering point failed: %v", err)` |
| 9 | `cameralookatlogic.go` | L34: `l.svcCtx.DjiClient.CameraLookAt(...)` | L36: `l.Errorf("[camera] camera look at failed: %v", err)` |
| 10 | `cameramodeswitchlogic.go` | L33: `l.svcCtx.DjiClient.CameraModeSwitch(...)` | L35: `l.Errorf("[camera] camera mode switch failed: %v", err)` |
| 11 | `cameraphotostoplogic.go` | L27: `l.svcCtx.DjiClient.CameraPhotoStop(...)` | L29: `l.Errorf("[camera] camera photo stop failed: %v", err)` |
| 12 | `cameraphotostoragesetlogic.go` | L32: `l.svcCtx.DjiClient.CameraPhotoStorageSet(...)` | L34: `l.Errorf("[camera] camera photo storage set failed: %v", err)` |
| 13 | `cameraphototakelogic.go` | L31: `l.svcCtx.DjiClient.CameraPhotoTake(...)` | L33: `l.Errorf("[camera] camera photo take failed: %v", err)` |
| 14 | `camerapointfocusactionlogic.go` | L34: `l.svcCtx.DjiClient.CameraPointFocusAction(...)` | L36: `l.Errorf("[camera] camera point focus action failed: %v", err)` |
| 15 | `camerarecordingstartlogic.go` | L31: `l.svcCtx.DjiClient.CameraRecordingStart(...)` | L33: `l.Errorf("[camera] camera recording start failed: %v", err)` |
| 16 | `camerarecordingstoplogic.go` | L31: `l.svcCtx.DjiClient.CameraRecordingStop(...)` | L33: `l.Errorf("[camera] camera recording stop failed: %v", err)` |
| 17 | `camerascreendraglogic.go` | L33: `l.svcCtx.DjiClient.CameraScreenDrag(...)` | L35: `l.Errorf("[camera] camera screen drag failed: %v", err)` |
| 18 | `camerascreensplitlogic.go` | L32: `l.svcCtx.DjiClient.CameraScreenSplit(...)` | L34: `l.Errorf("[camera] camera screen split failed: %v", err)` |
| 19 | `cameravideostoragesetlogic.go` | L32: `l.svcCtx.DjiClient.CameraVideoStorageSet(...)` | L34: `l.Errorf("[camera] camera video storage set failed: %v", err)` |
| 20 | `chargecloselogic.go` | L28: `l.svcCtx.DjiClient.ChargeClose(...)` | L30: `l.Errorf("[remote-debug] charge close failed: %v", err)` |
| 21 | `chargeopenlogic.go` | L28: `l.svcCtx.DjiClient.ChargeOpen(...)` | L30: `l.Errorf("[remote-debug] charge open failed: %v", err)` |
| 22 | `configupdatelogic.go` | L40: `l.svcCtx.DjiClient.ConfigUpdate(...)` | L42: `l.Errorf("[config] config update failed: %v", err)` |
| 23 | `covercloselogic.go` | L28: `l.svcCtx.DjiClient.CoverClose(...)` | L30: `l.Errorf("[remote-debug] cover close failed: %v", err)` |
| 24 | `coverforcecloselogic.go` | L28: `l.svcCtx.DjiClient.CoverForceClose(...)` | L30: `l.Errorf("[remote-debug] cover force close failed: %v", err)` |
| 25 | `coveropenlogic.go` | L28: `l.svcCtx.DjiClient.CoverOpen(...)` | L30: `l.Errorf("[remote-debug] cover open failed: %v", err)` |
| 26 | `debugmodecloselogic.go` | L28: `l.svcCtx.DjiClient.DebugModeClose(...)` | L30: `l.Errorf("[remote-debug] debug mode close failed: %v", err)` |
| 27 | `debugmodeopenlogic.go` | L28: `l.svcCtx.DjiClient.DebugModeOpen(...)` | L30: `l.Errorf("[remote-debug] debug mode open failed: %v", err)` |
| 28 | `deviceformatlogic.go` | L28: `l.svcCtx.DjiClient.DeviceFormat(...)` | L30: `l.Errorf("[remote-debug] device format failed: %v", err)` |
| 29 | `devicerebootlogic.go` | L28: `l.svcCtx.DjiClient.DeviceReboot(...)` | L30: `l.Errorf("[remote-debug] device reboot failed: %v", err)` |
| 30 | `drccameraaperturevaluesetlogic.go` | L34: `l.svcCtx.DjiClient.DrcCameraApertureValueSet(...)` | L35: `l.Errorf("[drc] camera aperture value set failed device_sn=%s: %v", deviceSn, err)` |
| 31 | `drccameradewarpingsetlogic.go` | L34: `l.svcCtx.DjiClient.DrcCameraDewarpingSet(...)` | L35: `l.Errorf("[drc] camera dewarping set failed device_sn=%s: %v", deviceSn, err)` |
| 32 | `drccameraisosetlogic.go` | L34: `l.svcCtx.DjiClient.DrcCameraIsoSet(...)` | L35: `l.Errorf("[drc] camera iso set failed device_sn=%s: %v", deviceSn, err)` |
| 33 | `drccameramechanicalshuttersetlogic.go` | L34: `l.svcCtx.DjiClient.DrcCameraMechanicalShutterSet(...)` | L35: `l.Errorf("[drc] camera mechanical shutter set failed device_sn=%s: %v", deviceSn, err)` |
| 34 | `drccamerashuttersetlogic.go` | L34: `l.svcCtx.DjiClient.DrcCameraShutterSet(...)` | L35: `l.Errorf("[drc] camera shutter set failed device_sn=%s: %v", deviceSn, err)` |
| 35 | `drcemergencylandinglogic.go` | L32: `l.svcCtx.DjiClient.DrcEmergencyLanding(...)` | L33: `l.Errorf("[drc] emergency landing failed device_sn=%s: %v", deviceSn, err)` |
| 36 | `drcforcelandinglogic.go` | L32: `l.svcCtx.DjiClient.DrcForceLanding(...)` | L33: `l.Errorf("[drc] force landing failed device_sn=%s: %v", deviceSn, err)` |
| 37 | `drcinitialstatesubscribelogic.go` | L32: `l.svcCtx.DjiClient.DrcInitialStateSubscribe(...)` | L33: `l.Errorf("[drc] initial state subscribe failed device_sn=%s: %v", deviceSn, err)` |
| 38 | `drcintervalphotosetlogic.go` | L36: `l.svcCtx.DjiClient.DrcIntervalPhotoSet(...)` | L37: `l.Errorf("[drc] interval photo set failed device_sn=%s: %v", deviceSn, err)` |
| 39 | `drclinkagezoomsetlogic.go` | L34: `l.svcCtx.DjiClient.DrcLinkageZoomSet(...)` | L35: `l.Errorf("[drc] linkage zoom set failed device_sn=%s: %v", deviceSn, err)` |
| 40 | `drcmodeenterlogic.go` | L37: `l.svcCtx.DjiClient.DrcModeEnter(...)` | L39: `l.Errorf("[drc] mode enter failed device_sn=%s: %v", deviceSn, err)` |
| 41 | `drcmodeenterlogic.go` | L47: `l.svcCtx.DjiClient.EnableDrc(...)` | L48: `l.Errorf("[drc] manager enable failed device_sn=%s: %v", deviceSn, err)` |
| 42 | `drcmodeexitlogic.go` | L28: `l.svcCtx.DjiClient.DrcModeExit(...)` | L30: `l.Errorf("[drc] mode exit failed device_sn=%s: %v", deviceSn, err)` |
| 43 | `drcmodeexitlogic.go` | L33: `l.svcCtx.DjiClient.DisableDrc(...)` | L34: `l.Errorf("[drc] manager disable failed device_sn=%s: %v", deviceSn, err)` |
| 44 | `drcnightlightsstatesetlogic.go` | L34: `l.svcCtx.DjiClient.DrcNightLightsStateSet(...)` | L35: `l.Errorf("[drc] night lights state set failed device_sn=%s: %v", deviceSn, err)` |
| 45 | `drcstealthstatesetlogic.go` | L34: `l.svcCtx.DjiClient.DrcStealthStateSet(...)` | L35: `l.Errorf("[drc] stealth state set failed device_sn=%s: %v", deviceSn, err)` |
| 46 | `drcvideoresolutionsetlogic.go` | L36: `l.svcCtx.DjiClient.DrcVideoResolutionSet(...)` | L37: `l.Errorf("[drc] video resolution set failed device_sn=%s: %v", deviceSn, err)` |
| 47 | `dronecloselogic.go` | L28: `l.svcCtx.DjiClient.DroneClose(...)` | L30: `l.Errorf("[remote-debug] drone close failed: %v", err)` |
| 48 | `droneemergencystoplogic.go` | L40: `l.svcCtx.DjiClient.DroneEmergencyStop(...)` | L41: `l.Errorf("[drc] drone emergency stop failed device_sn=%s: %v", deviceSn, err)` |
| 49 | `droneformatlogic.go` | L28: `l.svcCtx.DjiClient.DroneFormat(...)` | L30: `l.Errorf("[remote-debug] drone format failed: %v", err)` |
| 50 | `droneopenlogic.go` | L28: `l.svcCtx.DjiClient.DroneOpen(...)` | L30: `l.Errorf("[remote-debug] drone open failed: %v", err)` |
| 51 | `flightauthoritygrablogic.go` | L27: `l.svcCtx.DjiClient.FlightAuthorityGrab(...)` | L29: `l.Errorf("[drc] flight authority grab failed: %v", err)` |
| 52 | `flighttaskexecutelogic.go` | L27: `l.svcCtx.DjiClient.FlightTaskExecute(...)` | L29: `l.Errorf("[flight-task] flighttask_execute failed: %v", err)` |
| 53 | `flighttaskpreparelogic.go` | L62: `l.svcCtx.DjiClient.FlightTaskPrepare(...)` | L64: `l.Errorf("[flight-task] flighttask_prepare failed: %v", err)` |
| 54 | `flytopointlogic.go` | L41: `l.svcCtx.DjiClient.FlyToPoint(...)` | L43: `l.Errorf("[drc] fly to point failed: %v", err)` |
| 55 | `flytopointstoplogic.go` | L28: `l.svcCtx.DjiClient.FlyToPointStop(...)` | L32: `l.Errorf("[drc] fly to point stop failed: %v", err)` |
| 56 | `gimbalresetlogic.go` | L32: `l.svcCtx.DjiClient.GimbalReset(...)` | L34: `l.Errorf("[camera] gimbal reset failed: %v", err)` |
| 57 | `livecamerachangelogic.go` | L32: `l.svcCtx.DjiClient.LiveCameraChange(...)` | L34: `l.Errorf("[live] live camera change failed: %v", err)` |
| 58 | `livelenschangelogic.go` | L33: `l.svcCtx.DjiClient.LiveLensChange(...)` | L35: `l.Errorf("[live] live lens change failed: %v", err)` |
| 59 | `livesetqualitylogic.go` | L32: `l.svcCtx.DjiClient.LiveSetQuality(...)` | L34: `l.Errorf("[live] live set quality failed: %v", err)` |
| 60 | `livestartpushlogic.go` | L34: `l.svcCtx.DjiClient.LiveStartPush(...)` | L36: `l.Errorf("[live] live start push failed: %v", err)` |
| 61 | `livestoppushlogic.go` | L31: `l.svcCtx.DjiClient.LiveStopPush(...)` | L33: `l.Errorf("[live] live stop push failed: %v", err)` |
| 62 | `mediafastuploadlogic.go` | L30: `l.svcCtx.DjiClient.MediaFastUpload(...)` | L32: `l.Errorf("[media] fast upload failed: %v", err)` |
| 63 | `mediahighestpriorityuploadflighttasklogic.go` | L30: `l.svcCtx.DjiClient.MediaHighestPriorityUploadFlighttask(...)` | L32: `l.Errorf("[media] highest priority upload flighttask failed: %v", err)` |
| 64 | `mediauploadflighttaskmediaprioritizelogic.go` | L30: `l.svcCtx.DjiClient.MediaUploadFlighttaskMediaPrioritize(...)` | L32: `l.Errorf("[media] upload flighttask media prioritize failed: %v", err)` |
| 65 | `otacreatelogic.go` | L38: `l.svcCtx.DjiClient.OtaCreate(...)` | L40: `l.Errorf("[ota] ota create failed: %v", err)` |
| 66 | `pauseflighttasklogic.go` | L29: `l.svcCtx.DjiClient.PauseFlightTask(...)` | L34: `l.Errorf("[flight-task] pause flight task failed: %v", err)` |
| 67 | `payloadauthoritygrablogic.go` | L27: `l.svcCtx.DjiClient.PayloadAuthorityGrab(...)` | L29: `l.Errorf("[drc] payload authority grab failed: %v", err)` |
| 68 | `psdkuiresourceuploadlogic.go` | L34: `l.svcCtx.DjiClient.PsdkUIResourceUpload(...)` | L36: `l.Errorf("[psdk] ui resource upload failed device_sn=%s tid=%s: %v", ...)` |
| 69 | `remotelogfilelistlogic.go` | L33: `l.svcCtx.DjiClient.RemoteLogFileList(...)` | L35: `l.Errorf("[remote-log] file list failed: %v", err)` |
| 70 | `remotelogfileuploadcancellogic.go` | L41: `l.svcCtx.DjiClient.RemoteLogFileUploadCancel(...)` | L43: `l.Errorf("[remote-log] file upload cancel failed: %v", err)` |
| 71 | `remotelogfileuploadstartlogic.go` | L41: `l.svcCtx.DjiClient.RemoteLogFileUploadStart(...)` | L43: `l.Errorf("[remote-log] file upload start failed: %v", err)` |
| 72 | `remotelogfileuploadupdatelogic.go` | L41: `l.svcCtx.DjiClient.RemoteLogFileUploadUpdate(...)` | L43: `l.Errorf("[remote-log] file upload update failed: %v", err)` |
| 73 | `returnhomecancelautoreturnlogic.go` | L27: `l.svcCtx.DjiClient.ReturnHomeCancelAutoReturn(...)` | L29: `l.Errorf("[drc] return home cancel failed: %v", err)` |
| 74 | `returnhomelogic.go` | L27: `l.svcCtx.DjiClient.ReturnHome(...)` | L29: `l.Errorf("[drc] return home failed: %v", err)` |
| 75 | `returnspecifichomelogic.go` | L34: `l.svcCtx.DjiClient.ReturnSpecificHome(...)` | L36: `l.Errorf("[flight-control] return specific home failed: %v", err)` |
| 76 | `stopflighttasklogic.go` | L29: `l.svcCtx.DjiClient.StopFlightTask(...)` | L34: `l.Errorf("[flight-task] stop flight task failed: %v", err)` |
| 77 | `supplementlightcloselogic.go` | L28: `l.svcCtx.DjiClient.SupplementLightClose(...)` | L30: `l.Errorf("[remote-debug] supplement light close failed: %v", err)` |
| 78 | `supplementlightopenlogic.go` | L28: `l.svcCtx.DjiClient.SupplementLightOpen(...)` | L30: `l.Errorf("[remote-debug] supplement light open failed: %v", err)` |
| 79 | `takeofftopointlogic.go` | L39: `l.svcCtx.DjiClient.TakeoffToPoint(...)` | L41: `l.Errorf("[drc] takeoff to point failed: %v", err)` |

### Files with DjiClient call but NO `l.Errorf` (use `errRes` helper directly)

| # | File | SDK Call (line) | Error handling |
|---|---|---|---|
| 1 | `customdatatransmissiontoesdklogic.go` | L27: `l.svcCtx.DjiClient.CustomDataTransmissionToEsdk(...)` | L28-30: `if err != nil { return errRes(tid, err), nil }` |
| 2 | `customdatatransmissiontopsdklogic.go` | L36: `l.svcCtx.DjiClient.CustomDataTransmissionToPsdk(...)` | L37-39: `if err != nil { return errRes(tid, err), nil }` |
| 3 | `flightareasupdatelogic.go` | L36: `l.svcCtx.DjiClient.FlightAreasUpdate(...)` | L37-39: `if err != nil { return errRes(tid, err), nil }` |
| 4 | `flighttaskrecoverylogic.go` | L28: `l.svcCtx.DjiClient.FlightTaskRecovery(...)` | L32-34: `if err != nil { return errRes(tid, err), nil }` |
| 5 | `flighttaskundologic.go` | L27: `l.svcCtx.DjiClient.FlightTaskUndo(...)` | L28-30: `if err != nil { return errRes(tid, err), nil }` |
| 6 | `propertysetlogic.go` | L34: `l.svcCtx.DjiClient.PropertySet(...)` | L35-37: `if err != nil { return errRes(tid, err), nil }` |
| 7 | `unlocklicenselistlogic.go` | L30: `l.svcCtx.DjiClient.UnlockLicenseList(...)` | L31-33: `if err != nil { return errRes(tid, err), nil }` |
| 8 | `unlocklicenseswitchlogic.go` | L30: `l.svcCtx.DjiClient.UnlockLicenseSwitch(...)` | L31-33: `if err != nil { return errRes(tid, err), nil }` |
| 9 | `unlocklicenseupdatelogic.go` | L33: `l.svcCtx.DjiClient.UnlockLicenseUpdate(...)` | L34-36: `if err != nil { return errRes(tid, err), nil }` |

### Special cases (no `l.Errorf`, different error handling)

| # | File | SDK Call (line) | Error handling |
|---|---|---|---|
| 1 | `querydrcstatuslogic.go` | L28: `l.svcCtx.DjiClient.DrcStatus(deviceSn)` | No error returned by DrcStatus() |
| 2 | `stickcontrollogic.go` | L34: `l.svcCtx.DjiClient.StickControl(...)` | L35: `return nil, err` (propagates raw error to caller) |

## Helper Function

The `errRes` helper is defined in `helper.go:23`:

```go
func errRes(tid string, err error) *djicloud.CommonRes {
    return &djicloud.CommonRes{Code: -1, Message: err.Error(), Tid: tid}
}
```

This creates a `CommonRes` with the error message but does NOT log at error level. Files using `errRes` without `l.Errorf` will suppress the error from server logs (only the API response carries the error message).

## Caveats

- Excluded files (as instructed): `helper.go`, `drchelper.go`, `drchelper_test.go`
- Files like `listhmsalertslogic.go`, `listdeviceslogic.go`, `pinglogic.go`, `getdevicestatesnapshotlogic.go`, etc. do NOT call `l.svcCtx.DjiClient` at all — they are pure query/list handlers
- The error message formats are inconsistent: some use `%s` for device_sn, others just include it in the message string
