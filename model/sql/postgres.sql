-- PostgreSQL 兼容的 SQL 脚本
-- 1. 创建插入时触发的函数
CREATE OR REPLACE FUNCTION insert_modified_time()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.create_time = date_trunc('second', NOW()); -- 格式: 2023-10-05 12:34:56
    NEW.update_time = date_trunc('second', NOW());
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. 创建更新时触发的函数
CREATE OR REPLACE FUNCTION update_modified_update_time()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.update_time = date_trunc('second', NOW());
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 3. 创建设备与 IEC104 点位映射表
CREATE TABLE IF NOT EXISTS device_point_mapping (
    id BIGSERIAL PRIMARY KEY, 
    create_time TIMESTAMP NOT NULL, 
    update_time TIMESTAMP NOT NULL, 
    delete_time TIMESTAMP NULL, 
    del_state SMALLINT NOT NULL DEFAULT 0, 
    version INT NOT NULL DEFAULT 0, 
    create_user VARCHAR(64) DEFAULT '',
    update_user VARCHAR(64) DEFAULT '',
    dept_code VARCHAR(64) DEFAULT '',
    tag_station VARCHAR(64) DEFAULT '',
    coa INT NOT NULL DEFAULT 0, 
    ioa INT NOT NULL DEFAULT 0, 
    device_id VARCHAR(64) DEFAULT '',
    device_name VARCHAR(128) DEFAULT '',
    td_table_type VARCHAR(255) DEFAULT '',
    enable_push SMALLINT NOT NULL DEFAULT 1, 
    enable_raw_insert SMALLINT NOT NULL DEFAULT 1, 
    description VARCHAR(256) DEFAULT '',
    ext_1 VARCHAR(64) DEFAULT '',
    ext_2 VARCHAR(64) DEFAULT '',
    ext_3 VARCHAR(64) DEFAULT '',
    ext_4 VARCHAR(64) DEFAULT '',
    ext_5 VARCHAR(64) DEFAULT '',
    CONSTRAINT uq_device_point_mapping_tag_station_coa_ioa UNIQUE (tag_station, coa, ioa)
);

-- 为 device_point_mapping 表添加注释
COMMENT ON TABLE device_point_mapping IS '设备与 IEC104 点位映射表';

-- 为 device_point_mapping 表的列添加注释
COMMENT ON COLUMN device_point_mapping.id IS '自增主键ID';
COMMENT ON COLUMN device_point_mapping.create_time IS '创建时间';
COMMENT ON COLUMN device_point_mapping.update_time IS '更新时间';
COMMENT ON COLUMN device_point_mapping.delete_time IS '删除时间（软删除标记）';
COMMENT ON COLUMN device_point_mapping.del_state IS '删除状态：0-未删除，1-已删除';
COMMENT ON COLUMN device_point_mapping.version IS '版本号（乐观锁）';
COMMENT ON COLUMN device_point_mapping.create_user IS '创建人';
COMMENT ON COLUMN device_point_mapping.update_user IS '更新人';
COMMENT ON COLUMN device_point_mapping.dept_code IS '机构code';
COMMENT ON COLUMN device_point_mapping.tag_station IS '与 TDengine tag_station 对应';
COMMENT ON COLUMN device_point_mapping.coa IS '与 TDengine coa 对应';
COMMENT ON COLUMN device_point_mapping.ioa IS '与 TDengine ioa 对应';
COMMENT ON COLUMN device_point_mapping.device_id IS '设备编号/ID';
COMMENT ON COLUMN device_point_mapping.device_name IS '设备名称';
COMMENT ON COLUMN device_point_mapping.td_table_type IS 'TDengine 表类型（遥信表/遥测表等，逗号分隔）';
COMMENT ON COLUMN device_point_mapping.enable_push IS '是否允许caller服务推送数据：0-不允许，1-允许';
COMMENT ON COLUMN device_point_mapping.enable_raw_insert IS '是否允许插入 raw 原生数据：0-否，1-是';
COMMENT ON COLUMN device_point_mapping.description IS '备注信息';
COMMENT ON COLUMN device_point_mapping.ext_1 IS '扩展字段1，如：alarm, normal, control等，用于主题拆分';
COMMENT ON COLUMN device_point_mapping.ext_2 IS '扩展字段2';
COMMENT ON COLUMN device_point_mapping.ext_3 IS '扩展字段3';
COMMENT ON COLUMN device_point_mapping.ext_4 IS '扩展字段4';
COMMENT ON COLUMN device_point_mapping.ext_5 IS '扩展字段5';

-- 为 device_point_mapping 表创建触发器
CREATE TRIGGER "trigger_insert_modified_time"
    BEFORE INSERT
    ON "public"."device_point_mapping"
    FOR EACH ROW
    EXECUTE PROCEDURE insert_modified_time();

CREATE TRIGGER "trigger_update_modified_time"
    BEFORE UPDATE
    ON "public"."device_point_mapping"
    FOR EACH ROW
    EXECUTE PROCEDURE update_modified_update_time();

-- 4. 创建计划任务表
CREATE TABLE IF NOT EXISTS plan (
    id BIGSERIAL PRIMARY KEY, 
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    delete_time TIMESTAMP NULL, 
    del_state SMALLINT NOT NULL DEFAULT 0, 
    version INT NOT NULL DEFAULT 0, 
    create_user VARCHAR(64) DEFAULT '',
    update_user VARCHAR(64) DEFAULT '',
    dept_code VARCHAR(64) DEFAULT '',
    plan_id VARCHAR(64) NOT NULL,
    plan_name VARCHAR(128) DEFAULT '',
    type VARCHAR(64) DEFAULT '',
    group_id VARCHAR(64) DEFAULT '',
    recurrence_rule JSONB NOT NULL DEFAULT '{}'::jsonb, 
    start_time TIMESTAMP NOT NULL, 
    end_time TIMESTAMP NOT NULL, 
    status SMALLINT NOT NULL DEFAULT 0, 
    terminated_time TIMESTAMP NULL, 
    terminated_reason VARCHAR(256) DEFAULT '',
    paused_time TIMESTAMP NULL, 
    paused_reason VARCHAR(256) DEFAULT '',
    completed_time TIMESTAMP NULL, 
    description VARCHAR(256) DEFAULT '',
    ext_1 VARCHAR(256) DEFAULT '',
    ext_2 VARCHAR(256) DEFAULT '',
    ext_3 VARCHAR(256) DEFAULT '',
    ext_4 VARCHAR(256) DEFAULT '',
    ext_5 VARCHAR(256) DEFAULT '',
    CONSTRAINT uq_plan_plan_id UNIQUE (plan_id)
);

-- 为 plan 表添加注释
COMMENT ON TABLE plan IS '计划任务表';

-- 为 plan 表的列添加注释
COMMENT ON COLUMN plan.id IS '自增主键ID';
COMMENT ON COLUMN plan.create_time IS '创建时间';
COMMENT ON COLUMN plan.update_time IS '更新时间';
COMMENT ON COLUMN plan.delete_time IS '删除时间（软删除标记）';
COMMENT ON COLUMN plan.del_state IS '删除状态：0-未删除，1-已删除';
COMMENT ON COLUMN plan.version IS '版本号（乐观锁）';
COMMENT ON COLUMN plan.create_user IS '创建人';
COMMENT ON COLUMN plan.update_user IS '更新人';
COMMENT ON COLUMN plan.dept_code IS '机构code';
COMMENT ON COLUMN plan.plan_id IS '计划唯一标识';
COMMENT ON COLUMN plan.plan_name IS '计划任务名称';
COMMENT ON COLUMN plan.type IS '任务类型';
COMMENT ON COLUMN plan.group_id IS '计划组ID,用于分组管理计划任务';
COMMENT ON COLUMN plan.recurrence_rule IS '重复规则，JSON格式存储';
COMMENT ON COLUMN plan.start_time IS '规则生效开始时间';
COMMENT ON COLUMN plan.end_time IS '规则生效结束时间';
COMMENT ON COLUMN plan.status IS '状态：0-禁用，1-启用，2-暂停，3-终止';
COMMENT ON COLUMN plan.terminated_time IS '终止时间';
COMMENT ON COLUMN plan.terminated_reason IS '终止原因';
COMMENT ON COLUMN plan.paused_time IS '暂停时间';
COMMENT ON COLUMN plan.paused_reason IS '暂停原因';
COMMENT ON COLUMN plan.completed_time IS '完成时间';
COMMENT ON COLUMN plan.description IS '备注信息';
COMMENT ON COLUMN plan.ext_1 IS '扩展字段1';
COMMENT ON COLUMN plan.ext_2 IS '扩展字段2';
COMMENT ON COLUMN plan.ext_3 IS '扩展字段3';
COMMENT ON COLUMN plan.ext_4 IS '扩展字段4';
COMMENT ON COLUMN plan.ext_5 IS '扩展字段5';

-- 为 plan 表创建索引
CREATE INDEX idx_plan_table_type ON plan (type);
CREATE INDEX idx_plan_table_group_id ON plan (group_id);
CREATE INDEX idx_plan_table_status ON plan (status);
CREATE INDEX idx_plan_table_start_time ON plan (start_time);
CREATE INDEX idx_plan_table_end_time ON plan (end_time);
CREATE INDEX idx_plan_table_terminated_time ON plan (terminated_time);
CREATE INDEX idx_plan_table_paused_time ON plan (paused_time);

-- 为 plan 表创建触发器
CREATE TRIGGER "trigger_insert_modified_time"
    BEFORE INSERT
    ON "public"."plan"
    FOR EACH ROW
    EXECUTE PROCEDURE insert_modified_time();

CREATE TRIGGER "trigger_update_modified_time"
    BEFORE UPDATE
    ON "public"."plan"
    FOR EACH ROW
    EXECUTE PROCEDURE update_modified_update_time();

-- 5. 创建计划执行项表
CREATE TABLE IF NOT EXISTS plan_exec_item (
    id BIGSERIAL PRIMARY KEY, 
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    delete_time TIMESTAMP NULL, 
    del_state SMALLINT NOT NULL DEFAULT 0, 
    version INT NOT NULL DEFAULT 0, 
    create_user VARCHAR(64) DEFAULT '',
    update_user VARCHAR(64) DEFAULT '',
    dept_code VARCHAR(64) DEFAULT '',
    plan_pk BIGINT NOT NULL DEFAULT 0, 
    plan_id VARCHAR(64) NOT NULL DEFAULT '',
    batch_pk BIGINT NOT NULL DEFAULT 0, 
    batch_id VARCHAR(64) NOT NULL DEFAULT '', 
    exec_id VARCHAR(64) NOT NULL DEFAULT '', 
    item_id VARCHAR(64) NOT NULL DEFAULT '', 
    item_name VARCHAR(128) DEFAULT '',
    point_id VARCHAR(64) DEFAULT '',
    service_addr VARCHAR(256) NOT NULL DEFAULT '', 
    payload TEXT NOT NULL DEFAULT '', 
    request_timeout INT NOT NULL DEFAULT 0, 
    plan_trigger_time TIMESTAMP NOT NULL, 
    next_trigger_time TIMESTAMP NOT NULL, 
    last_trigger_time TIMESTAMP NULL, 
    trigger_count INT NOT NULL DEFAULT 0, 
    status SMALLINT NOT NULL DEFAULT 0, 
    last_result VARCHAR(256) DEFAULT '',
    last_message VARCHAR(1024) DEFAULT '',
    last_reason TEXT DEFAULT '',
    terminated_time TIMESTAMP NULL, 
    terminated_reason VARCHAR(256) DEFAULT '',
    paused_time TIMESTAMP NULL, 
    paused_reason VARCHAR(256) DEFAULT '',
    completed_time TIMESTAMP NULL, 
    ext_1 VARCHAR(256) DEFAULT '',
    ext_2 VARCHAR(256) DEFAULT '',
    ext_3 VARCHAR(256) DEFAULT '',
    ext_4 VARCHAR(256) DEFAULT '',
    ext_5 VARCHAR(256) DEFAULT ''
);

-- 为 plan_exec_item 表添加注释
COMMENT ON TABLE plan_exec_item IS '计划执行项表';

-- 为 plan_exec_item 表的列添加注释
COMMENT ON COLUMN plan_exec_item.id IS '自增主键ID';
COMMENT ON COLUMN plan_exec_item.create_time IS '创建时间';
COMMENT ON COLUMN plan_exec_item.update_time IS '更新时间';
COMMENT ON COLUMN plan_exec_item.delete_time IS '删除时间（软删除标记）';
COMMENT ON COLUMN plan_exec_item.del_state IS '删除状态：0-未删除，1-已删除';
COMMENT ON COLUMN plan_exec_item.version IS '版本号（乐观锁）';
COMMENT ON COLUMN plan_exec_item.create_user IS '创建人';
COMMENT ON COLUMN plan_exec_item.update_user IS '更新人';
COMMENT ON COLUMN plan_exec_item.dept_code IS '机构code';
COMMENT ON COLUMN plan_exec_item.plan_pk IS '关联的计划主键ID';
COMMENT ON COLUMN plan_exec_item.plan_id IS '关联的计划ID';
COMMENT ON COLUMN plan_exec_item.batch_pk IS '批主键ID';
COMMENT ON COLUMN plan_exec_item.batch_id IS '批ID';
COMMENT ON COLUMN plan_exec_item.exec_id IS '执行ID';
COMMENT ON COLUMN plan_exec_item.item_id IS '执行项ID';
COMMENT ON COLUMN plan_exec_item.item_name IS '执行项名称';
COMMENT ON COLUMN plan_exec_item.point_id IS '点位id';
COMMENT ON COLUMN plan_exec_item.service_addr IS '业务服务地址';
COMMENT ON COLUMN plan_exec_item.payload IS '业务负载';
COMMENT ON COLUMN plan_exec_item.request_timeout IS '请求超时时间（毫秒）';
COMMENT ON COLUMN plan_exec_item.plan_trigger_time IS '计划触发时间';
COMMENT ON COLUMN plan_exec_item.next_trigger_time IS '下次触发时间（扫表核心字段）';
COMMENT ON COLUMN plan_exec_item.last_trigger_time IS '上次触发时间';
COMMENT ON COLUMN plan_exec_item.trigger_count IS '触发次数';
COMMENT ON COLUMN plan_exec_item.status IS '状态：0-等待调度，10-延期等待，100-执行中，150-暂停，200-完成，300-终止';
COMMENT ON COLUMN plan_exec_item.last_result IS '上次执行结果';
COMMENT ON COLUMN plan_exec_item.last_message IS '上次结果描述';
COMMENT ON COLUMN plan_exec_item.last_reason IS '上次结果原因';
COMMENT ON COLUMN plan_exec_item.terminated_time IS '终止时间';
COMMENT ON COLUMN plan_exec_item.terminated_reason IS '终止原因';
COMMENT ON COLUMN plan_exec_item.paused_time IS '暂停时间';
COMMENT ON COLUMN plan_exec_item.paused_reason IS '暂停原因';
COMMENT ON COLUMN plan_exec_item.completed_time IS '完成时间';
COMMENT ON COLUMN plan_exec_item.ext_1 IS '扩展字段1';
COMMENT ON COLUMN plan_exec_item.ext_2 IS '扩展字段2';
COMMENT ON COLUMN plan_exec_item.ext_3 IS '扩展字段3';
COMMENT ON COLUMN plan_exec_item.ext_4 IS '扩展字段4';
COMMENT ON COLUMN plan_exec_item.ext_5 IS '扩展字段5';

-- 为 plan_exec_item 表创建索引
CREATE UNIQUE INDEX uk_plan_exec_item_exec_id ON plan_exec_item (exec_id);
CREATE INDEX idx_plan_exec_item_batch_pk ON plan_exec_item (batch_pk);
CREATE INDEX idx_plan_exec_item_batch_id ON plan_exec_item (batch_id);
CREATE INDEX idx_plan_exec_item_plan_pk_item_id ON plan_exec_item (plan_pk, item_id);
CREATE INDEX idx_plan_exec_item_plan_id_item_id ON plan_exec_item (plan_id, item_id);
CREATE INDEX idx_plan_exec_item_point_id ON plan_exec_item (point_id);
CREATE INDEX idx_plan_exec_item_status ON plan_exec_item (status);
CREATE INDEX idx_plan_exec_item_core_scan ON plan_exec_item (del_state, next_trigger_time, status);

-- 为 plan_exec_item 表创建触发器
CREATE TRIGGER "trigger_insert_modified_time"
    BEFORE INSERT
    ON "public"."plan_exec_item"
    FOR EACH ROW
    EXECUTE PROCEDURE insert_modified_time();

CREATE TRIGGER "trigger_update_modified_time"
    BEFORE UPDATE
    ON "public"."plan_exec_item"
    FOR EACH ROW
    EXECUTE PROCEDURE update_modified_update_time();

-- 6. 创建计划任务执行日志表
CREATE TABLE IF NOT EXISTS plan_exec_log (
    id BIGSERIAL PRIMARY KEY, 
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    delete_time TIMESTAMP NULL, 
    del_state SMALLINT NOT NULL DEFAULT 0, 
    version INT NOT NULL DEFAULT 0, 
    create_user VARCHAR(64) DEFAULT '',
    update_user VARCHAR(64) DEFAULT '',
    dept_code VARCHAR(64) DEFAULT '',
    plan_pk BIGINT NOT NULL DEFAULT 0, 
    plan_id VARCHAR(64) NOT NULL DEFAULT '',
    plan_name VARCHAR(128) DEFAULT '',
    batch_pk BIGINT NOT NULL DEFAULT 0,
    batch_id VARCHAR(64) NOT NULL DEFAULT '', 
    item_pk BIGINT NOT NULL DEFAULT 0,
    exec_id VARCHAR(64) NOT NULL DEFAULT '',
    item_id VARCHAR(64) NOT NULL DEFAULT '',
    item_name VARCHAR(128) DEFAULT '',
    point_id VARCHAR(64) DEFAULT '',
    trigger_time TIMESTAMP NOT NULL, 
    trace_id VARCHAR(64) DEFAULT '',
    exec_result VARCHAR(256) DEFAULT '',
    message VARCHAR(1024) DEFAULT '',
    reason TEXT DEFAULT ''
);

-- 为 plan_exec_log 表添加注释
COMMENT ON TABLE plan_exec_log IS '计划任务执行日志表';

-- 为 plan_exec_log 表的列添加注释
COMMENT ON COLUMN plan_exec_log.id IS '自增主键ID';
COMMENT ON COLUMN plan_exec_log.create_time IS '创建时间';
COMMENT ON COLUMN plan_exec_log.update_time IS '更新时间';
COMMENT ON COLUMN plan_exec_log.delete_time IS '删除时间（软删除标记）';
COMMENT ON COLUMN plan_exec_log.del_state IS '删除状态：0-未删除，1-已删除';
COMMENT ON COLUMN plan_exec_log.version IS '版本号（乐观锁）';
COMMENT ON COLUMN plan_exec_log.create_user IS '创建人';
COMMENT ON COLUMN plan_exec_log.update_user IS '更新人';
COMMENT ON COLUMN plan_exec_log.dept_code IS '机构code';
COMMENT ON COLUMN plan_exec_log.plan_pk IS '关联的计划主键ID';
COMMENT ON COLUMN plan_exec_log.plan_id IS '计划任务ID';
COMMENT ON COLUMN plan_exec_log.plan_name IS '计划任务名称';
COMMENT ON COLUMN plan_exec_log.batch_pk IS '批主键ID';
COMMENT ON COLUMN plan_exec_log.batch_id IS '批ID';
COMMENT ON COLUMN plan_exec_log.item_pk IS '关联的执行项主键ID';
COMMENT ON COLUMN plan_exec_log.exec_id IS '执行ID';
COMMENT ON COLUMN plan_exec_log.item_id IS '执行项ID';
COMMENT ON COLUMN plan_exec_log.item_name IS '执行项名称';
COMMENT ON COLUMN plan_exec_log.point_id IS '点位id';
COMMENT ON COLUMN plan_exec_log.trigger_time IS '触发时间';
COMMENT ON COLUMN plan_exec_log.trace_id IS '唯一追踪ID';
COMMENT ON COLUMN plan_exec_log.exec_result IS '执行结果';
COMMENT ON COLUMN plan_exec_log.message IS '结果描述';
COMMENT ON COLUMN plan_exec_log.reason IS '结果原因';

-- 为 plan_exec_log 表创建索引
CREATE INDEX idx_plan_exec_log_plan_pk ON plan_exec_log (plan_pk);
CREATE INDEX idx_plan_exec_log_plan_id ON plan_exec_log (plan_id);
CREATE INDEX idx_plan_exec_log_batch_pk ON plan_exec_log (batch_pk);
CREATE INDEX idx_plan_exec_log_batch_id ON plan_exec_log (batch_id);
CREATE INDEX idx_plan_exec_log_item_pk ON plan_exec_log (item_pk);
CREATE INDEX idx_plan_exec_log_exec_id ON plan_exec_log (exec_id);
CREATE INDEX idx_plan_exec_log_item_id ON plan_exec_log (item_id);
CREATE INDEX idx_plan_exec_log_trigger_time ON plan_exec_log (trigger_time);
CREATE INDEX idx_plan_exec_log_trace_id ON plan_exec_log (trace_id);
CREATE INDEX idx_plan_exec_log_exec_result ON plan_exec_log (exec_result);

-- 为 plan_exec_log 表创建触发器
CREATE TRIGGER "trigger_insert_modified_time"
    BEFORE INSERT
    ON "public"."plan_exec_log"
    FOR EACH ROW
    EXECUTE PROCEDURE insert_modified_time();

CREATE TRIGGER "trigger_update_modified_time"
    BEFORE UPDATE
    ON "public"."plan_exec_log"
    FOR EACH ROW
    EXECUTE PROCEDURE update_modified_update_time();

-- 7. 创建计划批次表
CREATE TABLE IF NOT EXISTS plan_batch (
    id BIGSERIAL PRIMARY KEY, 
    create_time TIMESTAMP NOT NULL, 
    update_time TIMESTAMP NOT NULL, 
    delete_time TIMESTAMP NULL, 
    del_state SMALLINT NOT NULL DEFAULT 0, 
    version INT NOT NULL DEFAULT 0, 
    create_user VARCHAR(64) DEFAULT '',
    update_user VARCHAR(64) DEFAULT '',
    dept_code VARCHAR(64) DEFAULT '',
    plan_pk BIGINT NOT NULL DEFAULT 0, 
    plan_id VARCHAR(64) NOT NULL DEFAULT '', 
    batch_id VARCHAR(64) NOT NULL DEFAULT '', 
    batch_name VARCHAR(128) DEFAULT '',
    status SMALLINT NOT NULL DEFAULT 0,
    plan_trigger_time TIMESTAMP NULL,
    completed_time TIMESTAMP NULL,
    ext_1 VARCHAR(256) DEFAULT '',
    ext_2 VARCHAR(256) DEFAULT '',
    ext_3 VARCHAR(256) DEFAULT '',
    ext_4 VARCHAR(256) DEFAULT '',
    ext_5 VARCHAR(256) DEFAULT '',
    CONSTRAINT uq_plan_batch_batch_id UNIQUE (batch_id)
);

-- 为 plan_batch 表添加注释
COMMENT ON TABLE plan_batch IS '计划批次表';

-- 为 plan_batch 表的列添加注释
COMMENT ON COLUMN plan_batch.id IS '自增主键ID';
COMMENT ON COLUMN plan_batch.create_time IS '创建时间';
COMMENT ON COLUMN plan_batch.update_time IS '更新时间';
COMMENT ON COLUMN plan_batch.delete_time IS '删除时间（软删除标记）';
COMMENT ON COLUMN plan_batch.del_state IS '删除状态：0-未删除，1-已删除';
COMMENT ON COLUMN plan_batch.version IS '版本号（乐观锁）';
COMMENT ON COLUMN plan_batch.create_user IS '创建人';
COMMENT ON COLUMN plan_batch.update_user IS '更新人';
COMMENT ON COLUMN plan_batch.dept_code IS '机构code';
COMMENT ON COLUMN plan_batch.plan_pk IS '关联的计划主键ID';
COMMENT ON COLUMN plan_batch.plan_id IS '关联的计划ID';
COMMENT ON COLUMN plan_batch.batch_id IS '批ID';
COMMENT ON COLUMN plan_batch.batch_name IS '批次名称';
COMMENT ON COLUMN plan_batch.status IS '状态：0-禁用，1-启用，2-暂停，3-终止';
COMMENT ON COLUMN plan_batch.plan_trigger_time IS '计划触发时间';
COMMENT ON COLUMN plan_batch.completed_time IS '完成时间';
COMMENT ON COLUMN plan_batch.ext_1 IS '扩展字段1';
COMMENT ON COLUMN plan_batch.ext_2 IS '扩展字段2';
COMMENT ON COLUMN plan_batch.ext_3 IS '扩展字段3';
COMMENT ON COLUMN plan_batch.ext_4 IS '扩展字段4';
COMMENT ON COLUMN plan_batch.ext_5 IS '扩展字段5';

-- 为 plan_batch 表创建索引
CREATE INDEX idx_plan_batch_plan_id ON plan_batch (plan_id);
CREATE INDEX idx_plan_batch_plan_pk ON plan_batch (plan_pk);
CREATE INDEX idx_plan_batch_status ON plan_batch (status);

-- 为 plan_batch 表创建触发器
CREATE TRIGGER "trigger_insert_modified_time"
    BEFORE INSERT
    ON "public"."plan_batch"
    FOR EACH ROW
    EXECUTE PROCEDURE insert_modified_time();

CREATE TRIGGER "trigger_update_modified_time"
    BEFORE UPDATE
    ON "public"."plan_batch"
    FOR EACH ROW
    EXECUTE PROCEDURE update_modified_update_time();
