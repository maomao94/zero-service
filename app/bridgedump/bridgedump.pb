
æ
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
msg (	Rmsg"ï
DeviceRunData
dtuId (	RdtuId
loadCur (RloadCur 
loadVoltage (RloadVoltage
sltype (Rsltype 
operateTime (	RoperateTime$
amTemperature (RamTemperature
gps (Rgps
workTime (	RworkTime":
CableFaultReq)
data (2.bridgedump.FaultDataRdata"5
CableFaultRes
code (Rcode
msg (	Rmsg"¥
	FaultData
acciId (	RacciId
downTime (	RdownTime
mlName (	RmlName
	fixedType (R	fixedType
	diaElepo1 (	R	diaElepo1
	diaElepo2 (	R	diaElepo2
diaElepo (	RdiaElepo 
errDistance (RerrDistance
fsltype	 (Rfsltype
notice
 (	Rnotice$
warnProcessed (RwarnProcessed"
warnCategory (	RwarnCategory
acciTime (	RacciTime
	shortType (	R	shortType"B
CableFaultWaveReq-
data (2.bridgedump.FaultWaveDataRdata"9
CableFaultWaveRes
code (Rcode
msg (	Rmsg"Ñ
FaultWaveData
acciId (	RacciId 
waveBatchId (	RwaveBatchId
dtuId (	RdtuId
waveType (RwaveType
	aheadTime (	R	aheadTime
waveData (	RwaveData
samprate (Rsamprate2š
BridgeDumpRpc(
Ping.bridgedump.Req.bridgedump.ResK
CableWorkList.bridgedump.CableWorkListReq.bridgedump.CableWorkListResB

CableFault.bridgedump.CableFaultReq.bridgedump.CableFaultResN
CableFaultWave.bridgedump.CableFaultWaveReq.bridgedump.CableFaultWaveResBZ./bridgedumpbproto3