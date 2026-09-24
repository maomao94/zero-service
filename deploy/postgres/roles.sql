-- PostgreSQL 业务账号体系脚本（与 kingbase/opengauss/mysql 的 roles.sql 口径一致）
-- 请在目标业务库执行；角色是实例级对象，表和 default privileges 是当前库级对象。
-- 用法（任选其一，连接到业务库）:
--   1) 数据库管理工具连接业务库后执行本文件
--   2) docker exec -i pgsql psql -U postgres -d <业务库名> < roles.sql
--
-- 账号体系（密码为本地开发示例，生产务必修改）:
--   app_user     业务系统程序连接账号: 业务表查增删改 + 建表
--   admin_user   管理者账号: 管理角色和用户 + 建库，业务表与 app_user 互相查增删改
--   query_user   现场查询账号: 客户端连接排查问题，仅 SELECT

do $$ begin
  if not exists (select 1 from pg_roles where rolname = 'app_user') then
    create role app_user with login password 'App_user@123';
  end if;
  if not exists (select 1 from pg_roles where rolname = 'query_user') then
    create role query_user with login password 'Query_user@123';
  end if;
  if not exists (select 1 from pg_roles where rolname = 'admin_user') then
    create role admin_user with login password 'Admin_user@123' createrole createdb;
  end if;
end $$;

-- 收回 PUBLIC 在 public 模式上的默认建表权限（PG 15+ 已默认收回，此行兼容 PG 14 及更早版本部署）
revoke create on schema public from public;

grant usage, create on schema public to app_user;
grant select, insert, update, delete on all tables in schema public to app_user;
grant usage, select on all sequences in schema public to app_user;
grant usage, create on schema public to admin_user;
grant select, insert, update, delete on all tables in schema public to admin_user;
grant usage, select on all sequences in schema public to admin_user;
grant usage on schema public to query_user;
grant select on all tables in schema public to query_user;

alter default privileges in schema public grant select, insert, update, delete on tables to app_user;
alter default privileges in schema public grant usage, select on sequences to app_user;
alter default privileges in schema public grant select, insert, update, delete on tables to admin_user;
alter default privileges in schema public grant usage, select on sequences to admin_user;
alter default privileges in schema public grant select on tables to query_user;
alter default privileges for role app_user in schema public grant select on tables to query_user;
alter default privileges for role app_user in schema public grant select, insert, update, delete on tables to admin_user;
alter default privileges for role app_user in schema public grant usage, select on sequences to admin_user;
alter default privileges for role admin_user in schema public grant select on tables to query_user;
alter default privileges for role admin_user in schema public grant select, insert, update, delete on tables to app_user;
alter default privileges for role admin_user in schema public grant usage, select on sequences to app_user;
