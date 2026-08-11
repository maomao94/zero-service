
ý=
bridgemodbus.protobridgemodbus"Å
PbModbusConfig
id (	Rid
create_time (	R
createTime
update_time (	R
updateTime
modbus_code (	R
modbusCode#
slave_address (	RslaveAddress
slave (Rslave
timeout (Rtimeout!
idle_timeout (RidleTimeout2
link_recovery_timeout	 (RlinkRecoveryTimeout:
protocol_recovery_timeout
 (RprotocolRecoveryTimeout#
connect_delay (RconnectDelay

enable_tls (R	enableTls"
tls_cert_file (	RtlsCertFile 
tls_key_file (	R
tlsKeyFile
tls_ca_file (	R	tlsCaFile
status (Rstatus
remark (	Rremark"k
SaveConfigReq
modbus_code (	R
modbusCode#
slave_address (	RslaveAddress
slave (Rslave"
SaveConfigRes
id (	Rid"#
DeleteConfigReq
ids (	Rids"
DeleteConfigRes"v
PageListConfigReq
page (Rpage
	page_size (RpageSize
keyword (	Rkeyword
status (Rstatus"Y
PageListConfigRes
total (Rtotal.
cfg (2.bridgemodbus.PbModbusConfigRcfg"5
GetConfigByCodeReq
modbus_code (	R
modbusCode"D
GetConfigByCodeRes.
cfg (2.bridgemodbus.PbModbusConfigRcfg":
BatchGetConfigByCodeReq
modbus_code (	R
modbusCode"I
BatchGetConfigByCodeRes.
cfg (2.bridgemodbus.PbModbusConfigRcfg"e
ReadCoilsReq
modbus_code (	R
modbusCode
address (Raddress
quantity (Rquantity"@
ReadCoilsRes
results (Rresults
values (Rvalues"n
ReadDiscreteInputsReq
modbus_code (	R
modbusCode
address (Raddress
quantity (Rquantity"I
ReadDiscreteInputsRes
results (Rresults
values (Rvalues"e
WriteSingleCoilReq
modbus_code (	R
modbusCode
address (Raddress
value (Rvalue".
WriteSingleCoilRes
results (Rresults"†
WriteMultipleCoilsReq
modbus_code (	R
modbusCode
address (Raddress
quantity (Rquantity
values (Rvalues"1
WriteMultipleCoilsRes
results (Rresults"n
ReadInputRegistersReq
modbus_code (	R
modbusCode
address (Raddress
quantity (Rquantity"µ
ReadInputRegistersRes
results (Rresults
uint_values (R
uintValues

int_values (R	intValues

hex_values (	R	hexValues#
binary_values (	RbinaryValues"p
ReadHoldingRegistersReq
modbus_code (	R
modbusCode
address (Raddress
quantity (Rquantity"·
ReadHoldingRegistersRes
results (Rresults
uint_values (R
uintValues

int_values (R	intValues

hex_values (	R	hexValues#
binary_values (	RbinaryValues"i
WriteSingleRegisterReq
modbus_code (	R
modbusCode
address (Raddress
value (Rvalue"2
WriteSingleRegisterRes
results (Rresults"
!WriteSingleRegisterWithDecimalReq
modbus_code (	R
modbusCode
address (Raddress
value (Rvalue
unsigned (Runsigned"=
!WriteSingleRegisterWithDecimalRes
results (Rresults"Š
WriteMultipleRegistersReq
modbus_code (	R
modbusCode
address (Raddress
quantity (Rquantity
values (Rvalues"5
WriteMultipleRegistersRes
results (Rresults"±
$WriteMultipleRegistersWithDecimalReq
modbus_code (	R
modbusCode
address (Raddress
quantity (Rquantity
values (Rvalues
unsigned (Runsigned"@
$WriteMultipleRegistersWithDecimalRes
results (Rresults"ì
ReadWriteMultipleRegistersReq
modbus_code (	R
modbusCode!
read_address (RreadAddress#
read_quantity (RreadQuantity#
write_address (RwriteAddress%
write_quantity (RwriteQuantity
values (Rvalues"½
ReadWriteMultipleRegistersRes
results (Rresults
uint_values (R
uintValues

int_values (R	intValues

hex_values (	R	hexValues#
binary_values (	RbinaryValues"…
MaskWriteRegisterReq
modbus_code (	R
modbusCode
address (Raddress
and_mask (RandMask
or_mask (RorMask"0
MaskWriteRegisterRes
results (Rresults"M
ReadFIFOQueueReq
modbus_code (	R
modbusCode
address (Raddress",
ReadFIFOQueueRes
results (Rresults"m
ReadDeviceIdentificationReq
modbus_code (	R
modbusCode-
read_device_id_code (RreadDeviceIdCode"ó
ReadDeviceIdentificationResP
results (26.bridgemodbus.ReadDeviceIdentificationRes.ResultsEntryRresultsY

hexResults (29.bridgemodbus.ReadDeviceIdentificationRes.HexResultsEntryR
hexResultsh
semanticResults (2>.bridgemodbus.ReadDeviceIdentificationRes.SemanticResultsEntryRsemanticResults:
ResultsEntry
key (Rkey
value (	Rvalue:8=
HexResultsEntry
key (	Rkey
value (	Rvalue:8B
SemanticResultsEntry
key (	Rkey
value (	Rvalue:8"i
)ReadDeviceIdentificationSpecificObjectReq
modbus_code (	R
modbusCode
	object_id (RobjectId"«
)ReadDeviceIdentificationSpecificObjectRes^
results (2D.bridgemodbus.ReadDeviceIdentificationSpecificObjectRes.ResultsEntryRresultsg

hexResults (2G.bridgemodbus.ReadDeviceIdentificationSpecificObjectRes.HexResultsEntryR
hexResultsv
semanticResults (2L.bridgemodbus.ReadDeviceIdentificationSpecificObjectRes.SemanticResultsEntryRsemanticResults:
ResultsEntry
key (Rkey
value (	Rvalue:8=
HexResultsEntry
key (	Rkey
value (	Rvalue:8B
SemanticResultsEntry
key (	Rkey
value (	Rvalue:8"V
 BatchConvertDecimalToRegisterReq
values (Rvalues
unsigned (Runsigned"Â
 BatchConvertDecimalToRegisterRes"
uint16Values (Ruint16Values 
int16Values (Rint16Values

hex_values (	R	hexValues#
binary_values (	RbinaryValues
bytes (Rbytes2ï
BridgeModbusF

SaveConfig.bridgemodbus.SaveConfigReq.bridgemodbus.SaveConfigResL
DeleteConfig.bridgemodbus.DeleteConfigReq.bridgemodbus.DeleteConfigResR
PageListConfig.bridgemodbus.PageListConfigReq.bridgemodbus.PageListConfigResU
GetConfigByCode .bridgemodbus.GetConfigByCodeReq .bridgemodbus.GetConfigByCodeResd
BatchGetConfigByCode%.bridgemodbus.BatchGetConfigByCodeReq%.bridgemodbus.BatchGetConfigByCodeResC
	ReadCoils.bridgemodbus.ReadCoilsReq.bridgemodbus.ReadCoilsRes^
ReadDiscreteInputs#.bridgemodbus.ReadDiscreteInputsReq#.bridgemodbus.ReadDiscreteInputsResU
WriteSingleCoil .bridgemodbus.WriteSingleCoilReq .bridgemodbus.WriteSingleCoilRes^
WriteMultipleCoils#.bridgemodbus.WriteMultipleCoilsReq#.bridgemodbus.WriteMultipleCoilsRes^
ReadInputRegisters#.bridgemodbus.ReadInputRegistersReq#.bridgemodbus.ReadInputRegistersResd
ReadHoldingRegisters%.bridgemodbus.ReadHoldingRegistersReq%.bridgemodbus.ReadHoldingRegistersResa
WriteSingleRegister$.bridgemodbus.WriteSingleRegisterReq$.bridgemodbus.WriteSingleRegisterRes‚
WriteSingleRegisterWithDecimal/.bridgemodbus.WriteSingleRegisterWithDecimalReq/.bridgemodbus.WriteSingleRegisterWithDecimalResj
WriteMultipleRegisters'.bridgemodbus.WriteMultipleRegistersReq'.bridgemodbus.WriteMultipleRegistersRes‹
!WriteMultipleRegistersWithDecimal2.bridgemodbus.WriteMultipleRegistersWithDecimalReq2.bridgemodbus.WriteMultipleRegistersWithDecimalResv
ReadWriteMultipleRegisters+.bridgemodbus.ReadWriteMultipleRegistersReq+.bridgemodbus.ReadWriteMultipleRegistersRes[
MaskWriteRegister".bridgemodbus.MaskWriteRegisterReq".bridgemodbus.MaskWriteRegisterResO
ReadFIFOQueue.bridgemodbus.ReadFIFOQueueReq.bridgemodbus.ReadFIFOQueueResp
ReadDeviceIdentification).bridgemodbus.ReadDeviceIdentificationReq).bridgemodbus.ReadDeviceIdentificationResš
&ReadDeviceIdentificationSpecificObject7.bridgemodbus.ReadDeviceIdentificationSpecificObjectReq7.bridgemodbus.ReadDeviceIdentificationSpecificObjectRes
BatchConvertDecimalToRegister..bridgemodbus.BatchConvertDecimalToRegisterReq..bridgemodbus.BatchConvertDecimalToRegisterResBC
com.github.bridgemodbus.grpcBBridgeModbusProtoPZ./bridgemodbusbproto3