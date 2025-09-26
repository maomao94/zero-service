
²
bridgemodbus.protobridgemodbus"
Req
ping (	Rping"
Res
pong (	Rpong"D
ReadCoilsReq
address (Raddress
quantity (Rquantity"@
ReadCoilsRes
results (Rresults
values (Rvalues"M
ReadDiscreteInputsReq
address (Raddress
quantity (Rquantity"I
ReadDiscreteInputsRes
results (Rresults
values (Rvalues"D
WriteSingleCoilReq
address (Raddress
value (Rvalue".
WriteSingleCoilRes
results (Rresults"e
WriteMultipleCoilsReq
address (Raddress
quantity (Rquantity
values (Rvalues"1
WriteMultipleCoilsRes
results (Rresults"M
ReadInputRegistersReq
address (Raddress
quantity (Rquantity"I
ReadInputRegistersRes
results (Rresults
values (	Rvalues"O
ReadHoldingRegistersReq
address (Raddress
quantity (Rquantity"K
ReadHoldingRegistersRes
results (Rresults
values (	Rvalues"H
WriteSingleRegisterReq
address (Raddress
value (Rvalue"2
WriteSingleRegisterRes
results (Rresults"i
WriteMultipleRegistersReq
address (Raddress
quantity (Rquantity
values (Rvalues"5
WriteMultipleRegistersRes
results (Rresults"Ç
ReadWriteMultipleRegistersReq 
readAddress (RreadAddress"
readQuantity (RreadQuantity"
writeAddress (RwriteAddress$
writeQuantity (RwriteQuantity
values (Rvalues"9
ReadWriteMultipleRegistersRes
results (Rresults"b
MaskWriteRegisterReq
address (Raddress
andMask (RandMask
orMask (RorMask"0
MaskWriteRegisterRes
results (Rresults",
ReadFIFOQueueReq
address (Raddress",
ReadFIFOQueueRes
results (Rresults"L
ReadDeviceIdentificationReq-
read_device_id_code (RreadDeviceIdCode"ó
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
value (	Rvalue:82Å	
BridgeModbus,
Ping.bridgemodbus.Req.bridgemodbus.ResC
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