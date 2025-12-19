CREATE TABLE IF NOT EXISTS device_point_mapping (
                                                    id INTEGER PRIMARY KEY AUTOINCREMENT,       -- 自增主键ID
                                                    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,  -- 创建时间
                                                    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,  -- 更新时间（可通过触发器自动更新）
                                                    delete_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,  -- 删除时间（软删除标记）
                                                    del_state INTEGER NOT NULL DEFAULT 0,       -- 删除状态：0-未删除，1-已删除
                                                    version INTEGER NOT NULL DEFAULT 0,         -- 版本号（乐观锁）

                                                    tag_station VARCHAR(64) NOT NULL DEFAULT '', -- 与 TDengine tag_station 对应
    coa INTEGER NOT NULL DEFAULT 0,              -- 与 TDengine coa 对应
    ioa INTEGER NOT NULL DEFAULT 0,              -- 与 TDengine ioa 对应
    device_id VARCHAR(64) NOT NULL DEFAULT '',   -- 设备编号/ID
    device_name VARCHAR(128) NOT NULL DEFAULT '',-- 设备名称（可选）
    td_table_type VARCHAR(255) NOT NULL DEFAULT '',-- TDengine表类型（如：遥信表,遥测表等，支持逗号拼接多个类型）
    enable_push INTEGER NOT NULL DEFAULT 1,      -- 是否允许caller服务推送数据：0-不允许，1-允许
    enable_raw_insert INTEGER NOT NULL DEFAULT 1,-- 是否允许插入到raw原生数据中：0-不允许，1-允许
    description VARCHAR(256) NOT NULL DEFAULT '',-- 备注信息（可选）

    UNIQUE(tag_station, coa, ioa)               -- 唯一索引，保证同一个点位只对应一个设备
    );


-- 插入设备映射数据，coa都是1，ioa是1-10
INSERT INTO device_point_mapping (tag_station, coa, ioa, device_id, device_name, td_table_type, enable_push, enable_raw_insert,
                                  description)
VALUES ('330KV', 1, 1, 'device_1_1', '遥信设备1-1', '遥信', 1, 1, '示例遥信设备1'),
       ('330KV', 1, 2, 'device_1_2', '遥信设备1-2', '遥信', 1, 1, '示例遥信设备2'),
       ('330KV', 1, 3, 'device_1_3', '遥信设备1-3', '遥信', 1, 1, '示例遥信设备3'),
       ('330KV', 1, 4, 'device_1_4', '遥信设备1-4', '遥信', 1, 1, '示例遥信设备4'),
       ('330KV', 1, 5, 'device_1_5', '遥信设备1-5', '遥信', 1, 1, '示例遥信设备5'),

       ('330KV', 1, 6, 'device_1_1', '遥测设备1-6', '遥信,告警', 1, 1, '示例遥测设备6'),
       ('330KV', 1, 7, 'device_1_2', '遥测设备1-7', '遥信', 1, 1, '示例遥测设备7'),
       ('330KV', 1, 8, 'device_1_3', '遥测设备1-8', '遥信', 1, 1, '示例遥测设备8'),
       ('330KV', 1, 9, 'device_1_4', '遥测设备1-9', '遥信', 1, 1, '示例遥测设备9'),
       ('330KV', 1, 10, 'device_1_5', '遥测设备1-10', '遥信', 1, 1, '示例遥测设备10');