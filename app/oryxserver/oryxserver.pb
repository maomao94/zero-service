
ÈF
oryxserver.proto
oryxserver"
VersionsReq"'
VersionsRes
version (	Rversion"
RecordQueryReq"r
RecordQueryRes
all (Rall
home (	Rhome
globs (	Rglobs$
process_cp_dir (	RprocessCpDir""
RecordApplyReq
all (Rall"
RecordApplyRes""
RecordEndReq
uuid (	Ruuid"
RecordEndRes"‘

RecordFile
uuid (	Ruuid
vhost (	Rvhost
app (	Rapp
stream (	Rstream
progress (Rprogress
update (	Rupdate
nn (Rnn
duration (Rduration
size	 (Rsize"
RecordFilesReq">
RecordFilesRes,
files (2.oryxserver.RecordFileRfiles"%
RecordRemoveReq
uuid (	Ruuid"
RecordRemoveRes"&
RecordGlobsReq
globs (	Rglobs"
RecordGlobsRes"\
RecordPostProcessingReq!
post_process (	RpostProcess
post_cp_dir (	R	postCpDir"
RecordPostProcessingRes"e
HooksApplyReq
target (	Rtarget
opaque (	Ropaque
all (Rall
host (	Rhost"
HooksApplyRes"
HooksQueryReq"â
HooksQueryRes
req (	Rreq
res (	Rres
target (	Rtarget
opaque (	Ropaque
all (Rall
host (	Rhost"
DvrQueryReq"7
DvrQueryRes
all (Rall
secret (Rsecret"
DvrApplyReq
all (Rall"
DvrApplyRes"Å
DvrFile
uuid (	Ruuid
vhost (	Rvhost
app (	Rapp
stream (	Rstream
progress (Rprogress
update (	Rupdate
nn (Rnn
duration (Rduration
size	 (Rsize
bucket
 (	Rbucket
region (	Rregion"
DvrFilesReq"8
DvrFilesRes)
files (2.oryxserver.DvrFileRfiles"
SrsVersionsReq"r
SrsVersionsRes
major (Rmajor
minor (Rminor
revision (Rrevision
version (	Rversion";
SrsStreamsReq
start (Rstart
count (Rcount"à
SrsStreamsRes
server (	Rserver
service (	Rservice
pid (	Rpid3
streams (2.oryxserver.SrsStreamItemRstreams"?
SrsKbps
recv_30s (Rrecv30s
send_30s (Rsend30s":
SrsPublishInfo
active (Ractive
cid (	Rcid"Ç
SrsVideoInfo
codec (	Rcodec
profile (	Rprofile
level (	Rlevel
width (Rwidth
height (Rheight"y
SrsAudioInfo
codec (	Rcodec
sample_rate (R
sampleRate
channel (Rchannel
profile (	Rprofile"Ã
SrsStreamItem
id (Rid
name (	Rname
vhost (	Rvhost
app (	Rapp
tc_url (	RtcUrl
url (	Rurl
live_ms (RliveMs
clients (Rclients
frames	 (Rframes

send_bytes
 (R	sendBytes

recv_bytes (R	recvBytes'
kbps (2.oryxserver.SrsKbpsRkbps4
publish (2.oryxserver.SrsPublishInfoRpublish.
video (2.oryxserver.SrsVideoInfoRvideo.
audio (2.oryxserver.SrsAudioInfoRaudio";
SrsClientsReq
start (Rstart
count (Rcount"à
SrsClientsRes
server (	Rserver
service (	Rservice
pid (	Rpid3
clients (2.oryxserver.SrsClientItemRclients"˘
SrsClientItem
id (Rid
vhost (	Rvhost
stream (	Rstream
ip (	Rip
page_url (	RpageUrl
swf_url (	RswfUrl
tc_url (	RtcUrl
url (	Rurl
name	 (	Rname
type
 (	Rtype
publish (Rpublish
alive (Ralive

send_bytes (R	sendBytes

recv_bytes (R	recvBytes'
kbps (2.oryxserver.SrsKbpsRkbps"
SrsVhostsReq"Ñ
SrsVhostsRes
server (	Rserver
service (	Rservice
pid (	Rpid0
vhosts (2.oryxserver.SrsVhostItemRvhosts"´
SrsVhostItem
id (Rid
name (	Rname
enabled (Renabled
clients (Rclients
streams (Rstreams

send_bytes (R	sendBytes

recv_bytes (R	recvBytes'
kbps (2.oryxserver.SrsKbpsRkbps
hls_enabled	 (R
hlsEnabled!
hls_fragment
 (RhlsFragment"
SrsSummariesReq"ò
SrsSummariesRes
ok (Rok
now_ms (RnowMs+
self (2.oryxserver.SrsSelfInfoRself1
system (2.oryxserver.SrsSystemInfoRsystem"Ò
SrsSelfInfo
version (	Rversion
pid (	Rpid
ppid (	Rppid
argv (	Rargv
cwd (	Rcwd
	mem_kbyte (RmemKbyte
mem_percent (R
memPercent
cpu_percent (R
cpuPercent

srs_uptime	 (R	srsUptime"Õ
SrsSystemInfo
cpu_percent (R
cpuPercent$
disk_read_kbps (RdiskReadKBps&
disk_write_kbps (RdiskWriteKBps*
disk_busy_percent (RdiskBusyPercent"
mem_ram_kbyte (RmemRamKbyte&
mem_ram_percent (RmemRamPercent$
mem_swap_kbyte (RmemSwapKbyte(
mem_swap_percent (RmemSwapPercent
cpus	 (Rcpus
cpus_online
 (R
cpusOnline
uptime (Ruptime
	idle_time (RildeTime
load_1m (Rload1m
load_5m (Rload5m
load_15m (Rload15m&
net_sample_time (RnetSampleTime$
net_recv_bytes (RnetRecvBytes$
net_send_bytes (RnetSendBytes&
net_recvi_bytes (RnetRecviBytes&
net_sendi_bytes (RnetSendiBytes&
srs_sample_time (RsrsSampleTime$
srs_recv_bytes (RsrsRecvBytes$
srs_send_bytes (RsrsSendBytes
conn_sys (RconnSys
conn_sys_et (R	connSysEt
conn_sys_tw (R	connSysTw 
conn_sys_udp (R
connSysUdp
conn_srs (RconnSrs"
SrsRequestsReq"N
SrsRequestsRes
uri (	Ruri
path (	Rpath
method (	Rmethod"ü
RecordBeginHookReq

request_id (	R	requestId
opaque (	Ropaque
vhost (	Rvhost
app (	Rapp
stream (	Rstream
uuid (	Ruuid"
RecordBeginHookRes"ä
RecordEndHookReq

request_id (	R	requestId
opaque (	Ropaque
vhost (	Rvhost
app (	Rapp
stream (	Rstream
uuid (	Ruuid#
artifact_code (RartifactCode#
artifact_path (	RartifactPath!
artifact_url	 (	RartifactUrl"
RecordEndHookRes"Õ
RecordListReq
uuid (	Ruuid
vhost (	Rvhost
app (	Rapp
stream (	Rstream
status (Rstatus
page (Rpage
	page_size (R	page_size*
begin_time_start (	Rbegin_time_start&
begin_time_end	 (	Rbegin_time_end&
end_time_start
 (	Rend_time_start"
end_time_end (	Rend_time_end"π

RecordItem
uuid (	Ruuid
vhost (	Rvhost
app (	Rapp
stream (	Rstream
opaque (	Ropaque
status (Rstatus

begin_time (R
begin_time
end_time (Rend_time#
artifact_code	 (RartifactCode#
artifact_path
 (	RartifactPath!
artifact_url (	RartifactUrl"â
RecordListRes0
records (2.oryxserver.RecordItemRrecords
total (Rtotal
page (Rpage
	page_size (R	page_size"%
RecordDeleteReq
uuid (	Ruuid"
RecordDeleteRes"–
StartRelayPullReq

source_url (	R	sourceUrl
app (	Rapp
stream (	Rstream

secret_key (	R	secretKey!
secret_value (	RsecretValue0
max_duration_seconds (RmaxDurationSeconds"X
StartRelayPullRes
relay_id (	RrelayId
app (	Rapp
stream (	Rstream"<
StopRelayPullReq
app (	Rapp
stream (	Rstream"
StopRelayPullRes"D
StopRelayAndRecordingReq
app (	Rapp
stream (	Rstream"
StopRelayAndRecordingRes2Ê

OryxServer<
Versions.oryxserver.VersionsReq.oryxserver.VersionsResE
RecordQuery.oryxserver.RecordQueryReq.oryxserver.RecordQueryResE
RecordApply.oryxserver.RecordApplyReq.oryxserver.RecordApplyRes?
	RecordEnd.oryxserver.RecordEndReq.oryxserver.RecordEndResE
RecordFiles.oryxserver.RecordFilesReq.oryxserver.RecordFilesResH
RecordRemove.oryxserver.RecordRemoveReq.oryxserver.RecordRemoveResE
RecordGlobs.oryxserver.RecordGlobsReq.oryxserver.RecordGlobsRes`
RecordPostProcessing#.oryxserver.RecordPostProcessingReq#.oryxserver.RecordPostProcessingRes<
DvrQuery.oryxserver.DvrQueryReq.oryxserver.DvrQueryRes<
DvrApply.oryxserver.DvrApplyReq.oryxserver.DvrApplyRes<
DvrFiles.oryxserver.DvrFilesReq.oryxserver.DvrFilesResB

HooksApply.oryxserver.HooksApplyReq.oryxserver.HooksApplyResB

HooksQuery.oryxserver.HooksQueryReq.oryxserver.HooksQueryResE
SrsVersions.oryxserver.SrsVersionsReq.oryxserver.SrsVersionsResB

SrsStreams.oryxserver.SrsStreamsReq.oryxserver.SrsStreamsResB

SrsClients.oryxserver.SrsClientsReq.oryxserver.SrsClientsRes?
	SrsVhosts.oryxserver.SrsVhostsReq.oryxserver.SrsVhostsResH
SrsSummaries.oryxserver.SrsSummariesReq.oryxserver.SrsSummariesResE
SrsRequests.oryxserver.SrsRequestsReq.oryxserver.SrsRequestsResB

RecordList.oryxserver.RecordListReq.oryxserver.RecordListResH
RecordDelete.oryxserver.RecordDeleteReq.oryxserver.RecordDeleteResN
StartRelayPull.oryxserver.StartRelayPullReq.oryxserver.StartRelayPullResK
StopRelayPull.oryxserver.StopRelayPullReq.oryxserver.StopRelayPullResc
StopRelayAndRecording$.oryxserver.StopRelayAndRecordingReq$.oryxserver.StopRelayAndRecordingResQ
RecordBeginHook.oryxserver.RecordBeginHookReq.oryxserver.RecordBeginHookResK
RecordEndHook.oryxserver.RecordEndHookReq.oryxserver.RecordEndHookResB=
com.github.oryxserver.grpcBOryxServerProtoPZ./oryxserverbproto3