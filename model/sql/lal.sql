CREATE
DATABASE IF NOT EXISTS `lal` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE
`lal`;

CREATE TABLE `hls_ts_files`
(
    `id`               BIGINT       NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    `create_time`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '插入时间',
    `update_time`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `delete_time`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '逻辑删除时间',
    `del_state`        TINYINT      NOT NULL DEFAULT '0' COMMENT '逻辑删除标记',
    `version`          BIGINT       NOT NULL DEFAULT '0' COMMENT '版本号',

    `event`            VARCHAR(16)  NOT NULL COMMENT '"open" 表示 TS 创建，"close" 表示 TS 写入完成',
    `stream_name`      VARCHAR(128) NOT NULL COMMENT '流名称',
    `cwd`              VARCHAR(256) NOT NULL COMMENT '当前工作路径',
    `ts_file`          VARCHAR(512) NOT NULL COMMENT 'TS 文件磁盘路径',
    `live_m3u8_file`   VARCHAR(512) NOT NULL COMMENT '直播 m3u8 文件路径',
    `record_m3u8_file` VARCHAR(512)          DEFAULT NULL COMMENT '录制 m3u8 文件路径（可为空）',
    `ts_id`            BIGINT       NOT NULL COMMENT 'TS 文件的 ID 编号，线性递增',
    `ts_timestamp`     BIGINT       NOT NULL COMMENT 'TS 文件时间戳，方便区间查询',
    `duration`         FLOAT                 DEFAULT NULL COMMENT 'TS 文件时长，单位秒，event 为 close 时有效',
    `server_id`        VARCHAR(64)  NOT NULL COMMENT 'lalserver 节点 ID',

    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_ts_file` (`ts_file`),
    KEY                `idx_event` (`event`),
    KEY                `idx_stream_name` (`stream_name`),
    KEY                `idx_ts_timestamp` (`ts_timestamp`),
) COMMENT='保存 HLS TS 文件生成事件';