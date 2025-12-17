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
    td_table_type VARCHAR(32) NOT NULL DEFAULT '',-- TDengine表类型（如：遥信表、遥测表等）
    description VARCHAR(256) NOT NULL DEFAULT '',-- 备注信息（可选）

    UNIQUE(tag_station, coa, ioa)               -- 唯一索引，保证同一个点位只对应一个设备
);