CREATE STABLE IF NOT EXISTS iec104.raw_point_data (
  ts TIMESTAMP,
  msg_id varchar(64),
  asdu varchar(32),
  type_id INT,
  data_type INT,
  ioa_value varchar(128),
  raw_msg varchar(5000)
) TAGS (tag_host varchar(64), tag_port INT, coa INT, ioa INT);