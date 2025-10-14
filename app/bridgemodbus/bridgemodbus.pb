
¼*
bridgemodbus.protobridgemodbus"
Req
ping (	Rping"
Res
pong (	Rpong"´
PbModbusConfig
id (Rid

createTime (	R
createTime

updateTime (	R
updateTime

modbusCode (	R
modbusCode"
slaveAddress (	RslaveAddress
slave (Rslave
timeout (Rtimeout 
idleTimeout (RidleTimeout0
linkRecoveryTimeout	 (RlinkRecoveryTimeout8
protocolRecoveryTimeout
 (RprotocolRecoveryTimeout"
connectDelay (RconnectDelay
	enableTls (R	enableTls 
tlsCertFile (	RtlsCertFile

tlsKeyFile (	R
tlsKeyFile
	tlsCaFile (	R	tlsCaFile
status (Rstatus
remark (	Rremark"i
SaveConfigReq

modbusCode (	R
modbusCode"
slaveAddress (	RslaveAddress
slave (Rslave"
SaveConfigRes
id (Rid"#
DeleteConfigReq
ids (Rids"
DeleteConfigRes"u
PageListConfigReq
page (Rpage
pageSize (RpageSize
keyword (	Rkeyword
status (Rstatus"Y
PageListConfigRes
total (Rtotal.
cfg (2.bridgemodbus.PbModbusConfigRcfg"4
GetConfigByCodeReq

modbusCode (	R
modbusCode"D
GetConfigByCodeRes.
cfg (2.bridgemodbus.PbModbusConfigRcfg"d
ReadCoilsReq

modbusCode (	R
modbusCode
address (Raddress
quantity (Rquantity"@
ReadCoilsRes
results (Rresults
values (Rvalues"m
ReadDiscreteInputsReq

modbusCode (	R
modbusCode
address (Raddress
quantity (Rquantity"I
ReadDiscreteInputsRes
results (Rresults
values (Rvalues"d
WriteSingleCoilReq

modbusCode (	R
modbusCode
address (Raddress
value (Rvalue".
WriteSingleCoilRes
results (Rresults"…
WriteMultipleCoilsReq

modbusCode (	R
modbusCode
address (Raddress
quantity (Rquantity
values (Rvalues"1
WriteMultipleCoilsRes
results (Rresults"m
ReadInputRegistersReq

modbusCode (	R
modbusCode
address (Raddress
quantity (Rquantity"I
ReadInputRegistersRes
results (Rresults
values (	Rvalues"o
ReadHoldingRegistersReq

modbusCode (	R
modbusCode
address (Raddress
quantity (Rquantity"K
ReadHoldingRegistersRes
results (Rresults
values (	Rvalues"h
WriteSingleRegisterReq

modbusCode (	R
modbusCode
address (Raddress
value (Rvalue"2
WriteSingleRegisterRes
results (Rresults"‰
WriteMultipleRegistersReq

modbusCode (	R
modbusCode
address (Raddress
quantity (Rquantity
values (Rvalues"5
WriteMultipleRegistersRes
results (Rresults"ç
ReadWriteMultipleRegistersReq

modbusCode (	R
modbusCode 
readAddress (RreadAddress"
readQuantity (RreadQuantity"
writeAddress (RwriteAddress$
writeQuantity (RwriteQuantity
values (Rvalues"9
ReadWriteMultipleRegistersRes
results (Rresults"‚
MaskWriteRegisterReq

modbusCode (	R
modbusCode
address (Raddress
andMask (RandMask
orMask (RorMask"0
MaskWriteRegisterRes
results (Rresults"L
ReadFIFOQueueReq

modbusCode (	R
modbusCode
address (Raddress",
ReadFIFOQueueRes
results (Rresults"i
ReadDeviceIdentificationReq

modbusCode (	R
modbusCode*
readDeviceIdCode (RreadDeviceIdCode"ó
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
value (	Rvalue:82†
BridgeModbus,
Ping.bridgemodbus.Req.bridgemodbus.ResF

SaveConfig.bridgemodbus.SaveConfigReq.bridgemodbus.SaveConfigResL
DeleteConfig.bridgemodbus.DeleteConfigReq.bridgemodbus.DeleteConfigResR
PageListConfig.bridgemodbus.PageListConfigReq.bridgemodbus.PageListConfigResU
GetConfigByCode .bridgemodbus.GetConfigByCodeReq .bridgemodbus.GetConfigByCodeResC
	ReadCoils.bridgemodbus.ReadCoilsReq.bridgemodbus.ReadCoilsRes^
ReadDiscreteInputs#.bridgemodbus.ReadDiscreteInputsReq#.bridgemodbus.ReadDiscreteInputsResU
WriteSingleCoil .bridgemodbus.WriteSingleCoilReq .bridgemodbus.WriteSingleCoilRes^
WriteMultipleCoils#.bridgemodbus.WriteMultipleCoilsReq#.bridgemodbus.WriteMultipleCoilsRes^
ReadInputRegisters#.bridgemodbus.ReadInputRegistersReq#.bridgemodbus.ReadInputRegistersResd
ReadHoldingRegisters%.bridgemodbus.ReadHoldingRegistersReq%.bridgemodbus.ReadHoldingRegistersResa
WriteSingleRegister$.bridgemodbus.WriteSingleRegisterReq$.bridgemodbus.WriteSingleRegisterResj
WriteMultipleRegisters'.bridgemodbus.WriteMultipleRegistersReq'.bridgemodbus.WriteMultipleRegistersResv
ReadWriteMultipleRegisters+.bridgemodbus.ReadWriteMultipleRegistersReq+.bridgemodbus.ReadWriteMultipleRegistersRes[
MaskWriteRegister".bridgemodbus.MaskWriteRegisterReq".bridgemodbus.MaskWriteRegisterResO
ReadFIFOQueue.bridgemodbus.ReadFIFOQueueReq.bridgemodbus.ReadFIFOQueueResp
ReadDeviceIdentification).bridgemodbus.ReadDeviceIdentificationReq).bridgemodbus.ReadDeviceIdentificationResBC
com.github.bridgemodbus.grpcBBridgeModbusProtoPZ./bridgemodbusbproto3