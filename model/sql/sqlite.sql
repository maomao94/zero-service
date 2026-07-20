CREATE TABLE IF NOT EXISTS device_point_mapping (
    id VARCHAR(64) PRIMARY KEY,       -- 去杠 UUID 字符串主键
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,  -- 创建时间
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,  -- 更新时间
    delete_time TIMESTAMP NULL,  -- 删除时间（软删除标记）
    is_deleted INTEGER NOT NULL DEFAULT 0,      -- 删除状态：0-未删除，1-已删除
    create_user VARCHAR(64) DEFAULT '', -- 创建人
    update_user VARCHAR(64) DEFAULT '', -- 更新人
    dept_code VARCHAR(64) DEFAULT '', -- 机构code

    tag_station VARCHAR(64) NOT NULL DEFAULT '', -- 与 TDengine tag_station 对应
    coa INTEGER NOT NULL DEFAULT 0,              -- 与 TDengine coa 对应
    ioa INTEGER NOT NULL DEFAULT 0,              -- 与 TDengine ioa 对应
    device_id VARCHAR(64) NOT NULL DEFAULT '',   -- 设备编号/ID
    device_name VARCHAR(128) NOT NULL DEFAULT '',-- 设备名称（可选）
    td_table_type VARCHAR(255) DEFAULT '',-- TDengine表类型（如：遥信表,遥测表等，支持逗号拼接多个类型）
    enable_push INTEGER NOT NULL DEFAULT 1,      -- 是否允许caller服务推送数据：0-不允许，1-允许
    enable_raw_insert INTEGER NOT NULL DEFAULT 1,-- 是否允许插入到raw原生数据中：0-不允许，1-允许
    description VARCHAR(256) DEFAULT '',-- 备注信息（可选）
    
    -- 扩展字段，用于存储额外的元数据
    ext_1 VARCHAR(64) DEFAULT '',      -- 扩展字段1，如：alarm, normal, control等
    ext_2 VARCHAR(64) DEFAULT '',      -- 扩展字段2
    ext_3 VARCHAR(64) DEFAULT '',      -- 扩展字段3
    ext_4 VARCHAR(64) DEFAULT '',      -- 扩展字段4
    ext_5 VARCHAR(64) DEFAULT '',      -- 扩展字段5

    UNIQUE(tag_station, coa, ioa)               -- 唯一索引，保证同一个点位只对应一个设备
    );


-- 插入设备映射数据，coa都是1，ioa是1-10
INSERT INTO device_point_mapping (id, tag_station, coa, ioa, device_id, device_name, td_table_type, enable_push, enable_raw_insert, ext_1, ext_2, description)
VALUES ('sample-330kv-1-1', '330KV', 1, 1, 'device_1_1', '遥信设备1-1', '遥信', 1, 1, 'normal', 'switch', '示例遥信设备1'),
       ('sample-330kv-1-2', '330KV', 1, 2, 'device_1_2', '遥信设备1-2', '遥信', 1, 1, 'normal', 'switch', '示例遥信设备2'),
       ('sample-330kv-1-3', '330KV', 1, 3, 'device_1_3', '遥信设备1-3', '遥信', 1, 1, 'alarm', 'temp', '示例遥信设备3'),
       ('sample-330kv-1-4', '330KV', 1, 4, 'device_1_4', '遥信设备1-4', '遥信', 1, 1, 'alarm', 'pressure', '示例遥信设备4'),
       ('sample-330kv-1-5', '330KV', 1, 5, 'device_1_5', '遥信设备1-5', '遥信', 1, 1, 'control', 'valve', '示例遥信设备5'),

       ('sample-330kv-1-6', '330KV', 1, 6, 'device_1_1', '遥测设备1-6', '遥信,告警', 1, 1, 'normal', 'voltage', '示例遥测设备6'),
       ('sample-330kv-1-7', '330KV', 1, 7, 'device_1_2', '遥测设备1-7', '遥信', 1, 1, 'normal', 'current', '示例遥测设备7'),
       ('sample-330kv-1-8', '330KV', 1, 8, 'device_1_3', '遥测设备1-8', '遥信', 1, 1, 'alarm', 'frequency', '示例遥测设备8'),
       ('sample-330kv-1-9', '330KV', 1, 9, 'device_1_4', '遥测设备1-9', '遥信', 1, 1, 'control', 'pump', '示例遥测设备9'),
       ('sample-330kv-1-10', '330KV', 1, 10, 'device_1_5', '遥测设备1-10', '遥信', 1, 1, 'normal', 'power', '示例遥测设备10');

-- 添加触发器，自动更新update_time
CREATE TRIGGER IF NOT EXISTS update_device_point_mapping_time
AFTER UPDATE ON device_point_mapping
FOR EACH ROW
BEGIN
    UPDATE device_point_mapping SET update_time = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;
