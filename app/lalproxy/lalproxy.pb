
µ 
lalproxy.protoLalproxy"3
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
pub (2.Lalproxy.PubSessionInfoRpub,
subs (2.Lalproxy.SubSessionInfoRsubs-
pull	 (2.Lalproxy.PullSessionInfoRpull/
pushs
 (2.Lalproxy.PushSessionInfoRpushs9
inFramePerSec (2.Lalproxy.FrameDataRinFramePerSec"É
LalServerData
serverId (	RserverId
binInfo (	RbinInfo

lalVersion (	R
lalVersion

apiVersion (	R
apiVersion$
notifyVersion (	RnotifyVersion
	startTime (	R	startTime"1
GetGroupInfoReq

streamName (	R
streamName"l
GetGroupInfoRes
	errorCode (R	errorCode
desp (	Rdesp'
data (2.Lalproxy.GroupDataRdata"
GetAllGroupsReq"p
GetAllGroupsRes
	errorCode (R	errorCode
desp (	Rdesp+
groups (2.Lalproxy.GroupDataRgroups"
GetLalInfoReq"n
GetLalInfoRes
	errorCode (R	errorCode
desp (	Rdesp+
data (2.Lalproxy.LalServerDataRdata"‘
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
desp (	Rdesp"p
KickSessionReq

streamName (	R
streamName
	sessionId (	R	sessionId 
sessionType (	RsessionType"B
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
desp (	Rdesp"C
StopRtpPubReq

streamName (	R
streamName
port (Rport"A
StopRtpPubRes
	errorCode (R	errorCode
desp (	Rdesp"E
AddIpBlacklistReq
ip (	Rip 
durationSec (RdurationSec"E
AddIpBlacklistRes
	errorCode (R	errorCode
desp (	Rdesp2ý
LalProxyD
GetGroupInfo.Lalproxy.GetGroupInfoReq.Lalproxy.GetGroupInfoResD
GetAllGroups.Lalproxy.GetAllGroupsReq.Lalproxy.GetAllGroupsRes>

GetLalInfo.Lalproxy.GetLalInfoReq.Lalproxy.GetLalInfoResJ
StartRelayPull.Lalproxy.StartRelayPullReq.Lalproxy.StartRelayPullResG
StopRelayPull.Lalproxy.StopRelayPullReq.Lalproxy.StopRelayPullResA
KickSession.Lalproxy.KickSessionReq.Lalproxy.KickSessionResA
StartRtpPub.Lalproxy.StartRtpPubReq.Lalproxy.StartRtpPubRes>

StopRtpPub.Lalproxy.StopRtpPubReq.Lalproxy.StopRtpPubResJ
AddIpBlacklist.Lalproxy.AddIpBlacklistReq.Lalproxy.AddIpBlacklistResB7
com.github.lalproxy.grpcBLalProxyProtoPZ
./lalproxybproto3