-- KingbaseES 业务建库脚本（在金仓默认库 kingbase 上执行）
-- 用法（任选其一）:
--   1) 数据库管理工具连接 127.0.0.1:54321，库选 kingbase，用户 system，然后执行本文件
--   2) 命令行: docker exec -i kingbase ksql -Usystem -d kingbase -p 54321 < deploy/kingbase/init.sql
--      （容器内 local 连接为 trust 免密，无需密码）
-- 注意: 库已存在时会报 "already exists"，忽略即可；create database 不支持 if not exists，
--       也不能包在事务/DO 块中，故不做幂等处理
-- 按需增删建库语句，defaultdb 仅为连接工具常用的默认库名示例

create database defaultdb;
