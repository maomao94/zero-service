
¬
lalproxy.protoLalproxy"¡
SessionInfo
	sessionId (	R	sessionId 
sessionType (	RsessionType
clientIp (	RclientIp

clientPort (R
clientPort
protocol (	Rprotocol"
createTimeMs (RcreateTimeMs
aliveSec (RaliveSec
	recvBytes (R	recvBytes
	sendBytes	 (R	sendBytes"š

VideoCodec
codec (	Rcodec
width (Rwidth
height (Rheight
fps (Rfps

bitrateBps (R
bitrateBps
gopSec (RgopSec"~

AudioCodec
codec (	Rcodec

sampleRate (R
sampleRate
channels (Rchannels

bitrateBps (R
bitrateBps"ö
	GroupData

streamName (	R
streamName
status (	Rstatus*
video (2.Lalproxy.VideoCodecRvideo*
audio (2.Lalproxy.AudioCodecRaudio1
sessions (2.Lalproxy.SessionInfoRsessions
pubCount (RpubCount
subCount (RsubCount
	pullCount (R	pullCount,
totalSessionCount	 (RtotalSessionCount"
createTimeMs
 (RcreateTimeMs"Û
LalServerData
version (	Rversion 
startTimeMs (RstartTimeMs
runSec (RrunSec 
listenAddrs (	RlistenAddrs*
activeGroupCount (RactiveGroupCount,
totalSessionCount (RtotalSessionCount
	goVersion (	R	goVersion
osArch (	RosArch
cpuCores	 (RcpuCores(
memoryUsedBytes
 (RmemoryUsedBytes"1
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