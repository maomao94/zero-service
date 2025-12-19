CREATE TABLE IF NOT EXISTS `device_point_mapping` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '自增主键ID',
    `create_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `delete_time` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间（软删除标记）',
    `del_state` TINYINT NOT NULL DEFAULT 0 COMMENT '删除状态：0-未删除，1-已删除',
    `version` INT NOT NULL DEFAULT 0 COMMENT '版本号（乐观锁）',
    `tag_station` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '与 TDengine tag_station 对应',
    `coa` INT NOT NULL DEFAULT 0 COMMENT '与 TDengine coa 对应',
    `ioa` INT NOT NULL DEFAULT 0 COMMENT '与 TDengine ioa 对应',
    `device_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '设备编号/ID',
    `device_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备名称',
    `td_table_type` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'TDengine 表类型（遥信表/遥测表等，逗号分隔）',
    `enable_push` TINYINT NOT NULL DEFAULT 1 COMMENT '是否允许caller服务推送数据：0-不允许，1-允许',
    `enable_raw_insert` TINYINT NOT NULL DEFAULT 1 COMMENT '是否允许插入 raw 原生数据：0-否，1-是',
    `description` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '备注信息',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_station_coa_ioa` (`tag_station`, `coa`, `ioa`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='设备与 IEC104 点位映射表';