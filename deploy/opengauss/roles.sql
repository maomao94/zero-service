-- openGauss 业务账号体系脚本（与 kingbase/postgres/mysql 的 roles.sql 口径一致）
-- 请在目标业务库执行；角色是实例级对象，表和 default privileges 是当前库级对象。
-- 用法（以 omm 连接业务库执行）:
--   docker exec -i opengauss gosu omm bash -c 'export GAUSSHOME=/usr/local/opengauss;
--     export PATH=$GAUSSHOME/bin:$PATH; export LD_LIBRARY_PATH=$GAUSSHOME/lib:/scws/lib:$LD_LIBRARY_PATH;
--     gsql -d <业务库名> -U omm' < roles.sql
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

-- 收回 PUBLIC 在 public 模式上的默认建表权限（openGauss 内核默认对所有人放开，
-- 会导致仅授权 SELECT 的账号也能建表; PG 15+ 已默认收回，此行保持各版本口径一致）
do $$ begin
  if has_schema_privilege('public', 'public', 'CREATE') then
    execute 'revoke create on schema public from public';
  end if;
end $$;

-- 临时表权限属于数据库级权限；收回 PUBLIC 后仅业务账号和管理员保留临时表能力，
-- 避免 query_user 通过 pg_temp 建表并写入临时数据，严格保持“仅 SELECT”。
do $$ begin
  if has_database_privilege('public', current_database(), 'TEMP') then
    execute format('revoke temporary on database %I from public', current_database());
  end if;
  execute format('grant temporary on database %I to app_user, admin_user', current_database());
end $$;

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
