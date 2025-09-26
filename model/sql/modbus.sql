CREATE TABLE `modbus_slave_config`
(
    `id`                        bigint       NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `create_time`               datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time`               datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `delete_time`               datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '删除时间（软删除标记）',
    `del_state`                 tinyint      NOT NULL DEFAULT '0' COMMENT '删除状态：0-未删除，1-已删除',
    `version`                   bigint       NOT NULL DEFAULT '0' COMMENT '版本号（乐观锁，防并发修改）',
    `modbus_code`               varchar(32)  NOT NULL DEFAULT '' COMMENT 'Modbus配置唯一编码（如：modbus-192.168.1.100）',
    `slave_address`             varchar(64)  NOT NULL DEFAULT '' COMMENT 'TCP设备地址（格式：IP:Port，对应结构体 Address）',
    `slave_id`                  tinyint      NOT NULL DEFAULT '1' COMMENT 'Modbus从站地址（Slave ID/Unit ID，对应结构体 SlaveID）',
    `timeout`                   int          NOT NULL DEFAULT '10000' COMMENT '发送/接收超时（单位：毫秒，对应结构体 Timeout，默认10000）',
    `idle_timeout`              int          NOT NULL DEFAULT '60000' COMMENT '空闲连接自动关闭时间（单位：毫秒，对应结构体 IdleTimeout，默认60000）',
    `link_recovery_timeout`     int          NOT NULL DEFAULT '3000' COMMENT 'TCP连接出错重连间隔（单位：毫秒，对应结构体 LinkRecoveryTimeout，默认3000）',
    `protocol_recovery_timeout` int          NOT NULL DEFAULT '2000' COMMENT '协议异常重试间隔（单位：毫秒，对应结构体 ProtocolRecoveryTimeout，默认2000）',
    `connect_delay`             int          NOT NULL DEFAULT '100' COMMENT '连接建立后等待时间（单位：毫秒，对应结构体 ConnectDelay，默认100）',
    `enable_tls`                tinyint      NOT NULL DEFAULT '0' COMMENT '是否启用TLS（对应结构体 TLS.Enable：0-不启用，1-启用）',
    `tls_cert_file`             varchar(255) NOT NULL DEFAULT '' COMMENT 'TLS客户端证书路径（对应结构体 TLS.CertFile，enable_tls=1时生效）',
    `tls_key_file`              varchar(255) NOT NULL DEFAULT '' COMMENT 'TLS客户端密钥路径（对应结构体 TLS.KeyFile，enable_tls=1时生效）',
    `tls_ca_file`               varchar(255) NOT NULL DEFAULT '' COMMENT 'TLS根证书路径（对应结构体 TLS.CAFile，enable_tls=1时生效）',
    `status`                    tinyint      NOT NULL DEFAULT '1' COMMENT '配置状态：1-启用（可初始化连接池），2-禁用（不加载）',
    `remark`                    varchar(255) NOT NULL DEFAULT '' COMMENT '备注（如：生产车间A-水泵控制从站）',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_modbus_code` (`modbus_code`) COMMENT '配置编码唯一索引（避免重复配置）'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Modbus从站连接配置表';

INSERT INTO `modbus_slave_config` (`modbus_code`, `slave_address`, `slave_id`, `status`, `remark`)
VALUES ('local', -- 唯一配置编码
        '127.0.0.1:5020', -- 从站地址（IP:Port）
        1, -- 从站ID（非默认值，区分同网段其他设备）
        1, -- 启用状态
        '备注');