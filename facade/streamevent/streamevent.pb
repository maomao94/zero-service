
¶/
streamevent.protostreamevent"M
ReceiveMQTTMessageReq4
messages (2.streamevent.MqttMessageRmessages"
ReceiveMQTTMessageRes"³
MqttMessage
	sessionId (	R	sessionId
msgId (	RmsgId$
topicTemplate (	RtopicTemplate
topic (	Rtopic
payload (Rpayload
sendTime (	RsendTime"
ReceiveWSMessageReq
	sessionId (	R	sessionId
msgId (	RmsgId
payload (Rpayload
sendTime (	RsendTime"
ReceiveWSMessageRes"O
ReceiveKafkaMessageReq5
messages (2.streamevent.KafkaMessageRmessages"
ReceiveKafkaMessageRes"œ
KafkaMessage
	sessionId (	R	sessionId
topic (	Rtopic
group (	Rgroup
key (Rkey
value (Rvalue
sendTime (	RsendTime"T
PushChunkAsduReq
tId (	RtId.
msgBody (2.streamevent.MsgBodyRmsgBody"
PushChunkAsduRes"œ
MsgBody
msgId (	RmsgId
host (	Rhost
port (Rport
asdu (	Rasdu
typeId (RtypeId
dataType (RdataType
coa (Rcoa
bodyRaw (	RbodyRaw
time	 (	Rtime 
metaDataRaw
 (	RmetaDataRaw)
pm (2.streamevent.PointMappingRpm"Ð
PointMapping
deviceId (	RdeviceId

deviceName (	R
deviceName 
tdTableType (	RtdTableType
ext1 (	Rext1
ext2 (	Rext2
ext3 (	Rext3
ext4 (	Rext4
ext5 (	Rext5"É
SinglePointInfo
ioa (Rioa
value (Rvalue
qds (Rqds
qdsDesc (	RqdsDesc
ov (Rov
bl (Rbl
sb (Rsb
nt (Rnt
iv	 (Riv
time
 (	Rtime"É
DoublePointInfo
ioa (Rioa
value (Rvalue
qds (Rqds
qdsDesc (	RqdsDesc
ov (Rov
bl (Rbl
sb (Rsb
nt (Rnt
iv	 (Riv
time
 (	Rtime"Ñ
MeasuredValueScaledInfo
ioa (Rioa
value (Rvalue
qds (Rqds
qdsDesc (	RqdsDesc
ov (Rov
bl (Rbl
sb (Rsb
nt (Rnt
iv	 (Riv
time
 (	Rtime"ã
MeasuredValueNormalInfo
ioa (Rioa
value (Rvalue
nva (Rnva
qds (Rqds
qdsDesc (	RqdsDesc
ov (Rov
bl (Rbl
sb (Rsb
nt	 (Rnt
iv
 (Riv
time (	Rtime"å
StepPositionInfo
ioa (Rioa/
value (2.streamevent.StepPositionRvalue
qds (Rqds
qdsDesc (	RqdsDesc
ov (Rov
bl (Rbl
sb (Rsb
nt (Rnt
iv	 (Riv
time
 (	Rtime"D
StepPosition
val (Rval"
hasTransient (RhasTransient"É
BitString32Info
ioa (Rioa
value (Rvalue
qds (Rqds
qdsDesc (	RqdsDesc
ov (Rov
bl (Rbl
sb (Rsb
nt (Rnt
iv	 (Riv
time
 (	Rtime"Ð
MeasuredValueFloatInfo
ioa (Rioa
value (Rvalue
qds (Rqds
qdsDesc (	RqdsDesc
ov (Rov
bl (Rbl
sb (Rsb
nt (Rnt
iv	 (Riv
time
 (	Rtime"y
BinaryCounterReadingInfo
ioa (Rioa7
value (2!.streamevent.BinaryCounterReadingRvalue
time (	Rtime"¶
BinaryCounterReading&
counterReading (RcounterReading
	seqNumber (R	seqNumber
hasCarry (RhasCarry

isAdjusted (R
isAdjusted
	isInvalid (R	isInvalid"ì
EventOfProtectionEquipmentInfo
ioa (Rioa
event (Revent
qdp (Rqdp
qdpDesc (	RqdpDesc
ei (Rei
bl (Rbl
sb (Rsb
nt (Rnt
iv	 (Riv
msec
 (Rmsec
time (	Rtime"ø
*PackedStartEventsOfProtectionEquipmentInfo
ioa (Rioa
event (Revent
qdp (Rqdp
qdpDesc (	RqdpDesc
ei (Rei
bl (Rbl
sb (Rsb
nt (Rnt
iv	 (Riv
msec
 (Rmsec
time (	Rtime"§
PackedOutputCircuitInfo
ioa (Rioa
oci (Roci
gc (Rgc
cl1 (Rcl1
cl2 (Rcl2
cl3 (Rcl3
qdp (Rqdp
qdpDesc (	RqdpDesc
ei	 (Rei
bl
 (Rbl
sb (Rsb
nt (Rnt
iv (Riv
msec (Rmsec
time (	Rtime"â
PackedSinglePointWithSCDInfo
ioa (Rioa
scd (Rscd
stn (	Rstn
cdn (	Rcdn
qds (Rqds
qdsDesc (	RqdsDesc
ov (Rov
bl (Rbl
sb	 (Rsb
nt
 (Rnt
iv (Riv"l
UpSocketMessageReq
reqId (	RreqId
sId (	RsId
event (	Revent
payload (	Rpayload"
UpSocketMessageRsp"¸
PbPlan

createTimee (	R
createTime

updateTimef (	R
updateTime

createUserg (	R
createUser

updateUserh (	R
updateUser
id2 (Rid
planId (	RplanId
planName (	RplanName
type (	Rtype
groupId (	RgroupId 
description (	Rdescription
	startTime (	R	startTime
endTime (	RendTime
ext1 (	Rext1
ext2	 (	Rext2
ext3
 (	Rext3
ext4 (	Rext4
ext5 (	Rext5"‚
HandlerPlanTaskEventReq'
pland (2.streamevent.PbPlanRplan
id2 (Rid
planPk (RplanPk
planId (	RplanId
batchPk (RbatchPk
batchId (	RbatchId
itemId
 (	RitemId
itemName (	RitemName
pointId (	RpointId
payload (	Rpayload(
planTriggerTime (	RplanTriggerTime

lastResult (	R
lastResult
lastMsg (	RlastMsg"‘
HandlerPlanTaskEventRes

execResult (	R
execResult
message (	Rmessage<
delayConfig (2.streamevent.PbDelayConfigRdelayConfig"[
PbDelayConfig(
nextTriggerTime (	RnextTriggerTime 
delayReason (	RdelayReason2¬
StreamEvent\
ReceiveMQTTMessage".streamevent.ReceiveMQTTMessageReq".streamevent.ReceiveMQTTMessageResV
ReceiveWSMessage .streamevent.ReceiveWSMessageReq .streamevent.ReceiveWSMessageRes_
ReceiveKafkaMessage#.streamevent.ReceiveKafkaMessageReq#.streamevent.ReceiveKafkaMessageResM
PushChunkAsdu.streamevent.PushChunkAsduReq.streamevent.PushChunkAsduResS
UpSocketMessage.streamevent.UpSocketMessageReq.streamevent.UpSocketMessageReqb
HandlerPlanTaskEvent$.streamevent.HandlerPlanTaskEventReq$.streamevent.HandlerPlanTaskEventResB@
com.github.streamevent.grpcBStreamEventProtoPZ./streameventbproto3