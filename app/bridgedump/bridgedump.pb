
ý
bridgedump.proto
bridgedump"
Req
ping (	Rping"
Res
pong (	Rpong"A
CableWorkListReq-
data (2.bridgedump.DeviceRunDataRdata"8
CableWorkListRes
code (Rcode
msg (	Rmsg"õ
DeviceRunData
dtu_id (	RdtuId
load_cur (RloadCur!
load_voltage (RloadVoltage
sltype (Rsltype!
operate_time (	RoperateTime%
am_temperature (RamTemperature
gps (Rgps
	work_time (	RworkTime":
CableFaultReq)
data (2.bridgedump.FaultDataRdata"5
CableFaultRes
code (Rcode
msg (	Rmsg"¯
	FaultData
acci_id (	RacciId
	down_time (	RdownTime
ml_name (	RmlName

fixed_type (R	fixedType
	diaElepo1 (	R	diaElepo1
	diaElepo2 (	R	diaElepo2
	dia_elepo (	RdiaElepo!
err_distance (RerrDistance
fsltype	 (Rfsltype
notice
 (	Rnotice%
warn_processed (RwarnProcessed#
warn_category (	RwarnCategory
	acci_time (	RacciTime

short_type (	R	shortType"B
CableFaultWaveReq-
data (2.bridgedump.FaultWaveDataRdata"9
CableFaultWaveRes
code (Rcode
msg (	Rmsg"Ø
FaultWaveData
acci_id (	RacciId"
wave_batch_id (	RwaveBatchId
dtu_id (	RdtuId
	wave_type (RwaveType

ahead_time (	R	aheadTime
	wave_data (	RwaveData
samprate (Rsamprate2š
BridgeDumpRpc(
Ping.bridgedump.Req.bridgedump.ResK
CableWorkList.bridgedump.CableWorkListReq.bridgedump.CableWorkListResB

CableFault.bridgedump.CableFaultReq.bridgedump.CableFaultResN
CableFaultWave.bridgedump.CableFaultWaveReq.bridgedump.CableFaultWaveResBZ./bridgedumpbproto3