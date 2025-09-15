
£ 
lalproxy.protolalproxy"3
	FrameData
unixSec (RunixSec
v (Rv"ì
PubSessionInfo
	sessionId (	R	sessionId
protocol (	Rprotocol
baseType (	RbaseType
	startTime (	R	startTime

remoteAddr (	R
remoteAddr"
readBytesSum (RreadBytesSum$
wroteBytesSum (RwroteBytesSum"
bitrateKbits (RbitrateKbits*
readBitrateKbits	 (RreadBitrateKbits,
writeBitrateKbits
 (RwriteBitrateKbits"ì
SubSessionInfo
	sessionId (	R	sessionId
protocol (	Rprotocol
baseType (	RbaseType
	startTime (	R	startTime

remoteAddr (	R
remoteAddr"
readBytesSum (RreadBytesSum$
wroteBytesSum (RwroteBytesSum"
bitrateKbits (RbitrateKbits*
readBitrateKbits	 (RreadBitrateKbits,
writeBitrateKbits
 (RwriteBitrateKbits"í
PullSessionInfo
	sessionId (	R	sessionId
protocol (	Rprotocol
baseType (	RbaseType
	startTime (	R	startTime

remoteAddr (	R
remoteAddr"
readBytesSum (RreadBytesSum$
wroteBytesSum (RwroteBytesSum"
bitrateKbits (RbitrateKbits*
readBitrateKbits	 (RreadBitrateKbits,
writeBitrateKbits
 (RwriteBitrateKbits"
PushSessionInfo"¼
	GroupData

streamName (	R
streamName
appName (	RappName

audioCodec (	R
audioCodec

videoCodec (	R
videoCodec

videoWidth (R
videoWidth 
videoHeight (RvideoHeight*
pub (2.lalproxy.PubSessionInfoRpub,
subs (2.lalproxy.SubSessionInfoRsubs-
pull	 (2.lalproxy.PullSessionInfoRpull/
pushs
 (2.lalproxy.PushSessionInfoRpushs9
inFramePerSec (2.lalproxy.FrameDataRinFramePerSec"í
LalServerData
serverId (	RserverId
binInfo (	RbinInfo

lalVersion (	R
lalVersion

apiVersion (	R
apiVersion$
notifyVersion (	RnotifyVersion"
webUiVersion (	RWebUiVersion
	startTime (	R	startTime"1
GetGroupInfoReq

streamName (	R
streamName"l
GetGroupInfoRes
	errorCode (R	errorCode
desp (	Rdesp'
data (2.lalproxy.GroupDataRdata"
GetAllGroupsReq"p
GetAllGroupsRes
	errorCode (R	errorCode
desp (	Rdesp+
groups (2.lalproxy.GroupDataRgroups"
GetLalInfoReq"n
GetLalInfoRes
	errorCode (R	errorCode
desp (	Rdesp+
data (2.lalproxy.LalServerDataRdata"‘
StartRelayPullReq
url (	Rurl

streamName (	R
streamName$
pullTimeoutMs (RpullTimeoutMs"
pullRetryNum (RpullRetryNum:
autoStopPullAfterNoOutMs (RautoStopPullAfterNoOutMs
rtspMode (RrtspMode(
debugDumpPacket (	RdebugDumpPacket"E
StartRelayPullRes
	errorCode (R	errorCode
desp (	Rdesp"2
StopRelayPullReq

streamName (	R
streamName"D
StopRelayPullRes
	errorCode (R	errorCode
desp (	Rdesp"N
KickSessionReq

streamName (	R
streamName
	sessionId (	R	sessionId"B
KickSessionRes
	errorCode (R	errorCode
desp (	Rdesp"ª
StartRtpPubReq

streamName (	R
streamName
port (Rport
	timeoutMs (R	timeoutMs
	isTcpFlag (R	isTcpFlag(
debugDumpPacket (	RdebugDumpPacket"B
StartRtpPubRes
	errorCode (R	errorCode
desp (	Rdesp"/
StopRtpPubReq

streamName (	R
streamName"A
StopRtpPubRes
	errorCode (R	errorCode
desp (	Rdesp"E
AddIpBlacklistReq
ip (	Rip 
durationSec (RdurationSec"E
AddIpBlacklistRes
	errorCode (R	errorCode
desp (	Rdesp2ý
lalProxyD
GetGroupInfo.lalproxy.GetGroupInfoReq.lalproxy.GetGroupInfoResD
GetAllGroups.lalproxy.GetAllGroupsReq.lalproxy.GetAllGroupsRes>

GetLalInfo.lalproxy.GetLalInfoReq.lalproxy.GetLalInfoResJ
StartRelayPull.lalproxy.StartRelayPullReq.lalproxy.StartRelayPullResG
StopRelayPull.lalproxy.StopRelayPullReq.lalproxy.StopRelayPullResA
KickSession.lalproxy.KickSessionReq.lalproxy.KickSessionResA
StartRtpPub.lalproxy.StartRtpPubReq.lalproxy.StartRtpPubRes>

StopRtpPub.lalproxy.StopRtpPubReq.lalproxy.StopRtpPubResJ
AddIpBlacklist.lalproxy.AddIpBlacklistReq.lalproxy.AddIpBlacklistResB7
com.github.lalproxy.grpcBLalProxyProtoPZ
./lalproxybproto3