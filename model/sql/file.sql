create table `oss`
(
    `id`          bigint       NOT NULL AUTO_INCREMENT,
    `create_time` datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `delete_time` datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `del_state`   tinyint      NOT NULL DEFAULT '0',
    `version`     bigint       NOT NULL DEFAULT '0' COMMENT '版本号',
    `tenant_id`   varchar(12)  NOT NULL default '000000' comment '租户ID',
    `category`    bigint       NOT NULL DEFAULT '0' comment '分类 1-minio 2-qiniu 3-ali 4-tecent',
    `oss_code`    varchar(32)  NOT NULL DEFAULT '' comment '资源编号',
    `endpoint`    varchar(255) NOT NULL DEFAULT '' comment '资源地址',
    `access_key`  varchar(255) NOT NULL DEFAULT '' comment 'accessKey',
    `secret_key`  varchar(255) NOT NULL DEFAULT '' comment 'secretKey',
    `bucket_name` varchar(255) NOT NULL DEFAULT '' comment '空间名',
    `app_id`      varchar(255) NOT NULL DEFAULT '' comment '应用ID',
    `region`      varchar(255) NOT NULL DEFAULT '' comment '地域简称',
    `remark`      varchar(255) NOT NULL DEFAULT '' comment '备注',
    `status`      bigint       NOT NULL DEFAULT '0' comment '状态 1-开启 2-关闭',
    primary key (`id`),
    UNIQUE KEY `idx_tid_code` (`tenant_id`,`oss_code`)
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='oss表';

INSERT INTO oss (create_time, update_time, del_state, version, tenant_id, category, oss_code,
                            endpoint, access_key, secret_key, bucket_name, app_id, region, remark, status)
VALUES ('2022-08-25 16:33:42', '2022-08-25 17:20:36', 0, 0, '000000', 1, 'minio-1',
        '127.0.0.1:19000', 'AKIAIOSFODNN7EXAMPLE', 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY', 'default', '', '', '',
        1);