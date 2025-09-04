

lalproxy.protolalproxy"
Req
ping (	Rping"
Res
pong (	Rpong"2
GetGroupInfoReq

streamName (	Rstream_name"m
GetGroupInfoRes
	errorCode (R
error_code
desp (	Rdesp'
data (2.lalproxy.GroupDataRdata"
GetAllGroupsReq"q
GetAllGroupsRes
	errorCode (R
error_code
desp (	Rdesp+
groups (2.lalproxy.GroupDataRgroups"
GetLalInfoReq"o
GetLalInfoRes
	errorCode (R
error_code
desp (	Rdesp+
data (2.lalproxy.LalServerDataRdata"ó
StartRelayPullReq
url (	Rurl

streamName (	Rstream_name&
pullTimeoutMs (Rpull_timeout_ms$
pullRetryNum (Rpull_retry_num@
autoStopPullAfterNoOutMs (Rauto_stop_pull_after_no_out_ms
rtspMode (R	rtsp_mode"º
StartRelayPullRes
	errorCode (R
error_code
desp (	Rdesp9
data (2%.lalproxy.StartRelayPullRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"3
StopRelayPullReq

streamName (	Rstream_name"¸
StopRelayPullRes
	errorCode (R
error_code
desp (	Rdesp8
data (2$.lalproxy.StopRelayPullRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"s
KickSessionReq

streamName (	Rstream_name
	sessionId (	R
session_id!
sessionType (	Rsession_type"´
KickSessionRes
	errorCode (R
error_code
desp (	Rdesp6
data (2".lalproxy.KickSessionRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"E
StartRtpPubReq
port (Rport

streamName (	Rstream_name"´
StartRtpPubRes
	errorCode (R
error_code
desp (	Rdesp6
data (2".lalproxy.StartRtpPubRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"D
StopRtpPubReq
port (Rport

streamName (	Rstream_name"²
StopRtpPubRes
	errorCode (R
error_code
desp (	Rdesp5
data (2!.lalproxy.StopRtpPubRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"J
AddIpBlacklistReq
ip (	Rip%
expireSeconds (Rexpire_seconds"º
AddIpBlacklistRes
	errorCode (R
error_code
desp (	Rdesp9
data (2%.lalproxy.AddIpBlacklistRes.DataEntryRdata7
	DataEntry
key (	Rkey
value (	Rvalue:8"ñ
	GroupData

streamName (	Rstream_name$
createTimeMs (Rcreate_time_ms3
	publisher (2.lalproxy.SessionInfoR	publisher7
subscribers (2.lalproxy.SessionInfoRsubscribers/
pullers (2.lalproxy.SessionInfoRpullers"Á
SessionInfo
	sessionId (	R
session_id
type (	Rtype
clientIp (	R	client_ip$
createTimeMs (Rcreate_time_ms
	sendBytes (R
send_bytes
	recvBytes (R
recv_bytes"Õ
LalServerData
	server_id (	R	server_id
bin_info (	Rbin_info 
lal_version (	Rlal_version 
api_version (	Rapi_version&
notify_version (	Rnotify_version

start_time (	R
start_time2ý
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