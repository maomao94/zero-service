-- TDengine 不同表类型的插入语句示例

-- 1. 原始数据总表插入语句（默认都会插入）
-- INSERT INTO iec104.raw_${stationId} USING iec104.raw_point_data 
--     TAGS ('${stationId}', ${coa}, ${ioa}) 
--     VALUES ('${time}', '${msgId}', '${host}', ${port}, '${asdu}', ${typeId}, ${dataType}, ${coa}, ${ioa}, '${value}', '${rawMsg}');

-- 2. 遥信表插入语句（使用 bool 类型）
-- INSERT INTO iec104.tele_signal_${stationId} USING iec104.tele_signal_data 
--     TAGS ('${stationId}', '${deviceId}', ${coa}, ${ioa}) 
--     VALUES ('${time}', '${msgId}', ${dataType}, ${value});  -- value 为 true/false

-- 3. 遥测表插入语句
-- INSERT INTO iec104.telemetry_${stationId} USING iec104.telemetry_data 
--     TAGS ('${stationId}', '${deviceId}', ${coa}, ${ioa}) 
--     VALUES ('${time}', '${msgId}', ${dataType}, ${value});  -- value 为数值

-- 示例：根据 td_table_type 配置生成插入语句的逻辑
-- 假设从 device_point_mapping 表中查询到：
-- device_id = 'device_1_2'
-- td_table_type = '遥信表,遥测表'

-- 1. 解析 td_table_type，得到表类型列表：['遥信表', '遥测表']
-- 2. 遍历表类型列表，生成对应的插入语句：

-- 遥信表插入语句示例（使用 bool 类型）
INSERT INTO iec104.tele_signal_station_1 USING iec104.tele_signal_data 
    TAGS ('station_1', 'device_1_2', 1, 2) 
    VALUES ('2025-12-17 10:00:00', 'msg_123', 1, true);

-- 遥测表插入语句示例
INSERT INTO iec104.telemetry_station_1 USING iec104.telemetry_data 
    TAGS ('station_1', 'device_1_2', 1, 2) 
    VALUES ('2025-12-17 10:00:00', 'msg_123', 1, 123.45);

-- 示例：根据不同设备类型生成的完整插入语句

-- 设备：device_1_1 (td_table_type = '遥信表')
INSERT INTO iec104.raw_station_1 USING iec104.raw_point_data 
    TAGS ('station_1', 1, 1) 
    VALUES ('2025-12-17 10:00:00', 'msg_123', '192.168.1.1', 502, 'asdu_1', 1, 1, 1, 1, '1', '{"value":1}');

INSERT INTO iec104.tele_signal_station_1 USING iec104.tele_signal_data 
    TAGS ('station_1', 'device_1_1', 1, 1) 
    VALUES ('2025-12-17 10:00:00', 'msg_123', 1, true);

-- 设备：device_1_2 (td_table_type = '遥信表,遥测表')
INSERT INTO iec104.raw_station_1 USING iec104.raw_point_data 
    TAGS ('station_1', 1, 2) 
    VALUES ('2025-12-17 10:00:00', 'msg_124', '192.168.1.1', 502, 'asdu_2', 1, 1, 1, 2, '1', '{"value":1}');

INSERT INTO iec104.tele_signal_station_1 USING iec104.tele_signal_data 
    TAGS ('station_1', 'device_1_2', 1, 2) 
    VALUES ('2025-12-17 10:00:00', 'msg_124', 1, true);

INSERT INTO iec104.telemetry_station_1 USING iec104.telemetry_data 
    TAGS ('station_1', 'device_1_2', 1, 2) 
    VALUES ('2025-12-17 10:00:00', 'msg_124', 1, 123.45);

-- 设备：device_1_3 (td_table_type = '遥测表')
INSERT INTO iec104.raw_station_1 USING iec104.raw_point_data 
    TAGS ('station_1', 1, 3) 
    VALUES ('2025-12-17 10:00:00', 'msg_125', '192.168.1.1', 502, 'asdu_3', 1, 1, 1, 3, '123.45', '{"value":123.45}');

INSERT INTO iec104.telemetry_station_1 USING iec104.telemetry_data 
    TAGS ('station_1', 'device_1_3', 1, 3) 
    VALUES ('2025-12-17 10:00:00', 'msg_125', 1, 123.45);

-- 设备：device_1_5 (td_table_type = '遥测表'，日发电数据)
INSERT INTO iec104.raw_station_1 USING iec104.raw_point_data 
    TAGS ('station_1', 1, 5) 
    VALUES ('2025-12-17 23:59:59', 'msg_127', '192.168.1.1', 502, 'asdu_5', 1, 1, 1, 5, '1234.56', '{"value":1234.56, "type":"daily_power"}');

INSERT INTO iec104.telemetry_station_1 USING iec104.telemetry_data 
    TAGS ('station_1', 'device_1_5', 1, 5) 
    VALUES ('2025-12-17 23:59:59', 'msg_127', 1, 1234.56);
