-- 原始数据总表
CREATE
STABLE IF NOT EXISTS iec104.raw_point_data (
  ts TIMESTAMP,
  msg_id varchar(64),
  host_v varchar(64),
  port_v INT,
  asdu varchar(32),
  type_id INT,
  data_type INT,
  coa INT,
  ioa INT,
  ioa_value varchar(1048),
  raw_msg varchar(5000)
) TAGS (tag_station varchar(64),tag_coa INT,tag_ioa INT);

-- 遥信表（开关状态）
CREATE
STABLE IF NOT EXISTS iec104.tele_signal_data (
  ts TIMESTAMP,
  msg_id varchar(64),
  data_type INT,
  signal_value bool  -- 遥信值：true/false
) TAGS (tag_station varchar(64), device_id varchar(64), tag_coa INT, tag_ioa INT);

-- 遥测表（模拟量）
CREATE
STABLE IF NOT EXISTS iec104.telemetry_data (
  ts TIMESTAMP,
  msg_id varchar(64),
  data_type INT,
  telemetry_value double  -- 遥测值：数值类型
) TAGS (tag_station varchar(64), device_id varchar(64), tag_coa INT, tag_ioa INT);
