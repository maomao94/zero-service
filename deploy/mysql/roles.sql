-- MySQL 业务库与账号体系脚本（与 kingbase/opengauss/postgres 的 roles.sql 口径一致）
--
-- 用法（root 执行，可重复执行; 接入多个业务库时全局替换 zero 后再各跑一遍）:
--   1) docker exec -i mysql8 mysql -uroot -proot < deploy/mysql/roles.sql
--   2) 本机客户端:  mysql -h127.0.0.1 -P3306 -uroot -proot < deploy/mysql/roles.sql
--
-- 账号体系（密码为本地开发示例，生产务必修改）:
--   app_user     业务系统程序连接账号: 业务表查增删改 + 建表改表
--   admin_user   管理者账号: 建库删库 + 建账号，业务库内全权限并可向他人授权
--   query_user   现场查询账号: 客户端连接排查问题，仅 SELECT
--
-- 与 PG 系脚本差异: MySQL 账号由 用户@主机 组成（此处统一 '%' 允许任意来源连接），
--   库级授权自动覆盖库内后续新建的表，无需 PG 的 default privileges。

-- 1. 业务库（幂等; 将 zero 替换为实际业务库名）
create database if not exists `zero` default charset utf8mb4 collate utf8mb4_0900_ai_ci;

-- 2. 创建账号（幂等; 已存在的账号不会重置密码，请自行核对）
create user if not exists 'app_user'@'%' identified by 'App_user@123';
create user if not exists 'query_user'@'%' identified by 'Query_user@123';
create user if not exists 'admin_user'@'%' identified by 'Admin_user@123';

-- 3. 授权（重复执行无副作用）
-- app_user: 业务库内建表改表 + 查增删改（references 支持外键，临时表/锁表为常规业务需求）
grant select, insert, update, delete, create, alter, index, drop, references,
      create temporary tables, lock tables
  on `zero`.* to 'app_user'@'%';

-- query_user: 仅 SELECT
grant select on `zero`.* to 'query_user'@'%';

-- admin_user: 实例级建库删库 + 建账号（管理角色），业务库全权限并可授权
grant create, drop on *.* to 'admin_user'@'%';
grant create user on *.* to 'admin_user'@'%';
grant all privileges on `zero`.* to 'admin_user'@'%' with grant option;

-- 4. 常用运维操作（root 或 admin_user 执行，按需复制执行）:
--    修改账号密码:  alter user 'query_user'@'%' identified by 'new_password';
--    删除账号:      drop user 'query_user'@'%';
--    新建查询用户:  create user 'xxx'@'%' identified by 'xxx123';
--                   grant select on `zero`.* to 'xxx'@'%';
--    查看所有账号:  select user, host from mysql.user order by 1, 2;
--    查看账号权限:  show grants for 'app_user'@'%';
