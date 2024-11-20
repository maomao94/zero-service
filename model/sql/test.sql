-- ----------------------------
-- Table structure for user
-- ----------------------------
DROP TABLE IF EXISTS `user`;
CREATE TABLE `user`
(
    `id`          bigint       NOT NULL AUTO_INCREMENT,
    `create_time` datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `delete_time` datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `del_state`   tinyint      NOT NULL DEFAULT '0',
    `version`     bigint       NOT NULL DEFAULT '0' COMMENT '版本号',
    `mobile`      char(11)     NOT NULL DEFAULT '',
    `password`    varchar(255) NOT NULL DEFAULT '',
    `nickname`    varchar(255) NOT NULL DEFAULT '',
    `sex`         tinyint(1) NOT NULL DEFAULT '0' COMMENT '性别 0:男 1:女',
    `avatar`      varchar(255) NOT NULL DEFAULT '',
    `info`        varchar(255) NOT NULL DEFAULT '',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_mobile` (`mobile`)
) ENGINE=InnoDB COMMENT='用户表';

DROP TABLE IF EXISTS `region`;
CREATE TABLE `region`
(
    `id`            bigint      NOT NULL AUTO_INCREMENT,
    `create_time`   datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time`   datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `delete_time`   datetime    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `del_state`     tinyint     NOT NULL DEFAULT '0',
    `version`       bigint      NOT NULL DEFAULT '0' COMMENT '版本号',
    `code`          varchar(12) NOT NULL COMMENT '区划编号',
    `parent_code`   varchar(12) NULL DEFAULT NULL COMMENT '父区划编号',
    `ancestors`     varchar(255) NULL DEFAULT NULL COMMENT '祖区划编号',
    `name`          varchar(32) NULL DEFAULT NULL COMMENT '区划名称',
    `province_code` varchar(12) NULL DEFAULT NULL COMMENT '省级区划编号',
    `province_name` varchar(32) NULL DEFAULT NULL COMMENT '省级名称',
    `city_code`     varchar(12) NULL DEFAULT NULL COMMENT '市级区划编号',
    `city_name`     varchar(32) NULL DEFAULT NULL COMMENT '市级名称',
    `district_code` varchar(12) NULL DEFAULT NULL COMMENT '区级区划编号',
    `district_name` varchar(32) NULL DEFAULT NULL COMMENT '区级名称',
    `town_code`     varchar(12) NULL DEFAULT NULL COMMENT '镇级区划编号',
    `town_name`     varchar(32) NULL DEFAULT NULL COMMENT '镇级名称',
    `village_code`  varchar(12) NULL DEFAULT NULL COMMENT '村级区划编号',
    `village_name`  varchar(32) NULL DEFAULT NULL COMMENT '村级名称',
    `region_level`  int(2) NULL DEFAULT 0 COMMENT '层级',
    `sort`          int(2) NULL DEFAULT 0 COMMENT '排序',
    `remark`        varchar(255) NULL DEFAULT NULL COMMENT '备注',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_code` (`code`)
) ENGINE = InnoDB COMMENT = '行政区划表';

DROP TABLE IF EXISTS `order_txn`;
CREATE TABLE `order_txn`
(
    `id`                   bigint       NOT NULL AUTO_INCREMENT,
    `create_time`          datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time`          datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `delete_time`          datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `del_state`            tinyint      NOT NULL DEFAULT '0',
    `version`              bigint       NOT NULL DEFAULT '0' COMMENT '版本号',
    `txn_id`               varchar(64)  NOT NULL DEFAULT '' COMMENT '订单号',
    `ori_txn_id`           varchar(64)  NOT NULL DEFAULT '' COMMENT '原订单号',
    `txn_time`             datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '订单时间',
    `txn_date`             date         NOT NULL DEFAULT '1970-01-01 08:00:00' COMMENT '订单日期',
    `mch_id`               varchar(64)  NOT NULL DEFAULT '' COMMENT '商户id',
    `mch_order_no`         varchar(64)  NOT NULL DEFAULT '' COMMENT '商户订单号',
    `pay_type`             varchar(32)  NOT NULL DEFAULT '' COMMENT '支付类型 alipay-支付宝,wxpay-微信',
    `txn_type`             int          NOT NULL DEFAULT '0' COMMENT '交易类型,1000-消费,2000-退款',
    `txn_channel`          varchar(32)  NOT NULL DEFAULT '' COMMENT '交易渠道',
    `txn_amt`              int          NOT NULL DEFAULT '0' COMMENT '支付金额,单位分',
    `real_amt`             int          NOT NULL DEFAULT '0' COMMENT '实付金额,单位分',
    `result`               varchar(1)   NOT NULL DEFAULT 'U' COMMENT '交易结果,U-未处理,P-交易处理中,F-失败,T-超时,C-关闭,S-成功',
    `body`                 varchar(512) NOT NULL DEFAULT '' COMMENT '商品描述信息',
    `extra`                varchar(512) NOT NULL DEFAULT '' COMMENT '特定渠道发起时额外参数',
    `user_id`              bigint       NOT NULL DEFAULT '0' COMMENT '用户id',
    `channel_user`         varchar(64)  NOT NULL DEFAULT '' COMMENT '渠道用户标识,如微信openId,支付宝账号',
    `channel_pay_time`     datetime NULL DEFAULT NULL COMMENT '渠道支付执行成功时间',
    `channel_order_no`     varchar(64)  NOT NULL DEFAULT '' COMMENT '渠道订单号',
    `payer_acct`           varchar(64)  NOT NULL DEFAULT '' COMMENT '付款款账户',
    `payer_acct_name`      varchar(64)  NOT NULL DEFAULT '' COMMENT '付款账户名称',
    `payer_acct_bank_name` varchar(64)  NOT NULL DEFAULT '' COMMENT '付款账户银行名称',
    `payee_acct`           varchar(64)  NOT NULL DEFAULT '' COMMENT '收款账户',
    `payee_acct_name`      varchar(64)  NOT NULL DEFAULT '' COMMENT '收款账户名称',
    `payee_acct_bank_name` varchar(64)  NOT NULL DEFAULT '' COMMENT '收款账户银行名称',
    `qr_code`              varchar(200) NOT NULL DEFAULT '' COMMENT '生成二维码链接',
    `expire_time`          int          NOT NULL DEFAULT 0 COMMENT '订单失效时间,单位秒',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_txn_id` (`txn_id`),
    UNIQUE KEY `idx_mch_id_order_no` (`mch_id`, `mch_order_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='交易订单表';