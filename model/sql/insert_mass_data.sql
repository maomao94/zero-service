-- 生成并插入1-10万个点位数据的SQL脚本
-- 注意：运行此脚本可能需要较长时间，具体取决于系统性能

-- 方法1：使用WITH RECURSIVE生成序列（SQLite 3.8.3+支持）
WITH RECURSIVE generate_series(n) AS (
    VALUES (1)
    UNION ALL
    SELECT n + 1 FROM generate_series WHERE n < 100000
)
INSERT INTO device_point_mapping (tag_station, coa, ioa, device_id, device_name, td_table_type, enable_push, enable_raw_insert, description)
SELECT
    '330KV',                     -- tag_station（使用330KV）
    1,                           -- coa
    n,                           -- ioa (1-100000)
    'device_' || n,              -- device_id
    '测试设备_' || n,             -- device_name
    '未知',                      -- td_table_type（统一为未知）
    1,
    CASE
        WHEN n % 1000 = 0 THEN 0 -- 每1000个点禁用raw插入
        ELSE 1
        END,                         -- enable_raw_insert
    '测试点位_' || n              -- description
FROM generate_series;

-- 方法2：如果WITH RECURSIVE不可用，可以使用此方法
-- 注意：此方法需要手动调整循环次数，每次插入10000条，共10次
-- INSERT INTO device_point_mapping (tag_station, coa, ioa, device_id, device_name, td_table_type, enable_push, enable_raw_insert, description)
-- SELECT
--     'station_test',
--     1,
--     seq,
--     'device_' || seq,
--     '测试设备_' || seq,
--     CASE WHEN seq % 2 = 0 THEN '遥信表' ELSE '遥测表' END,
--    1,
--     CASE WHEN seq % 1000 = 0 THEN 0 ELSE 1 END,
--     '测试点位_' || seq
-- FROM (
--     SELECT (SELECT COUNT(*) FROM device_point_mapping) + rowid AS seq
--     FROM sqlite_sequence LIMIT 10000
-- );

-- 验证插入结果
SELECT COUNT(*) FROM device_point_mapping WHERE tag_station = '330KV';
-- SELECT * FROM device_point_mapping WHERE tag_station = '330KV' LIMIT 10;

-- 删除测试数据（如果需要）
-- DELETE FROM device_point_mapping WHERE tag_station = '330KV';