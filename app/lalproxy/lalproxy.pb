
§
lalproxy.protolalproxy"
Req
ping (	Rping"
Res
pong (	Rpong"1
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
data (2.lalproxy.LalServerDataRdata"ç
StartRelayPullReq
url (	Rurl

streamName (	R
streamName$
pullTimeoutMs (RpullTimeoutMs"
pullRetryNum (RpullRetryNum:
autoStopPullAfterNoOutMs (RautoStopPullAfterNoOutMs
rtspMode (RrtspMode"¹
StartRelayPullRes
	errorCode (R	errorCode
desp (	Rdesp9
data (2%.lalproxy.StartRelayPullRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"2
StopRelayPullReq

streamName (	R
streamName"·
StopRelayPullRes
	errorCode (R	errorCode
desp (	Rdesp8
data (2$.lalproxy.StopRelayPullRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"p
KickSessionReq

streamName (	R
streamName
	sessionId (	R	sessionId 
sessionType (	RsessionType"³
KickSessionRes
	errorCode (R	errorCode
desp (	Rdesp6
data (2".lalproxy.KickSessionRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"D
StartRtpPubReq
port (Rport

streamName (	R
streamName"³
StartRtpPubRes
	errorCode (R	errorCode
desp (	Rdesp6
data (2".lalproxy.StartRtpPubRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"C
StopRtpPubReq
port (Rport

streamName (	R
streamName"±
StopRtpPubRes
	errorCode (R	errorCode
desp (	Rdesp5
data (2!.lalproxy.StopRtpPubRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"I
AddIpBlacklistReq
ip (	Rip$
expireSeconds (RexpireSeconds"¹
AddIpBlacklistRes
	errorCode (R	errorCode
desp (	Rdesp9
data (2%.lalproxy.AddIpBlacklistRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"î
	GroupData

streamName (	R
streamName"
createTimeMs (RcreateTimeMs3
	publisher (2.lalproxy.SessionInfoR	publisher7
subscribers (2.lalproxy.SessionInfoRsubscribers/
pullers (2.lalproxy.SessionInfoRpullers"»
SessionInfo
	sessionId (	R	sessionId
type (	Rtype
clientIp (	RclientIp"
createTimeMs (RcreateTimeMs
	sendBytes (R	sendBytes
	recvBytes (R	recvBytes"…
LalServerData
version (	Rversion 
startTimeMs (RstartTimeMs$
uptimeSeconds (RuptimeSeconds*
totalConnections (RtotalConnections 
streamCount (RstreamCount

systemInfo (	R
systemInfo$
configSummary (	RconfigSummary2ý
LalProxyD
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