INSERT INTO `modbus_slave_config` (`modbus_code`, `slave_address`, `slave`, `status`, `remark`)
VALUES ('local', -- 唯一配置编码
        '127.0.0.1:5020', -- 从站地址（IP:Port）
        1, -- 从站ID（非默认值，区分同网段其他设备）
        1, -- 启用状态
        '备注');