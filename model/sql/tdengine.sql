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