-- PostgreSQL 兼容数据库业务账号体系脚本（KingbaseES/openGauss/PostgreSQL 通用）
--
-- 用法（任选其一，连接到业务库）:
--   1) 数据库管理工具连接业务库后执行本文件
--   2) Kingbase: docker exec -i kingbase ksql -Usystem -d <业务库名> -p 54321 < roles.sql
--   3) PostgreSQL: docker exec -i pgsql psql -Upostgres -d <业务库名> < roles.sql
--   4) openGauss: docker exec -i opengauss bash -c '... gsql -U omm -d <业务库名>' < roles.sql
-- 首次执行请使用 system 等具备角色管理权限的账号；角色是实例级对象，业务库授权在当前库生效。
--
-- 账号体系（密码为本地开发示例，生产务必修改）:
--   app_user     业务系统程序连接账号: 业务表查增删改 + 建表
--   admin_user   管理者账号: 管理角色和用户 + 建库，业务表与 app_user 互相查增删改
--   query_user   现场查询账号: 客户端连接排查问题，仅 SELECT

-- 1. 创建账号（幂等; create role 不支持 if not exists，用 DO 块判断）
-- 已存在的角色不会重置密码或属性；请先核对其登录、CREATEROLE/CREATEDB 属性和密码。
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

-- 2. 授权（对当前连接的数据库生效; 重复执行无副作用）
-- 先收回 PUBLIC 在 public 模式上的默认建表权限（PG 15 以前的内核及兼容库默认对所有人放开，
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

-- app_user 与 admin_user: schema 建表权限 + 业务表查增删改（含自增序列使用权）
grant usage, create on schema public to app_user;
grant select, insert, update, delete on all tables in schema public to app_user;
grant usage, select on all sequences in schema public to app_user;

grant usage, create on schema public to admin_user;
grant select, insert, update, delete on all tables in schema public to admin_user;
grant usage, select on all sequences in schema public to admin_user;

-- 查询用户: 仅 SELECT
grant usage on schema public to query_user;
grant select on all tables in schema public to query_user;

-- 3. 后续由 system 新建的表/序列自动继承授权
alter default privileges in schema public grant select, insert, update, delete on tables to app_user;
alter default privileges in schema public grant usage, select on sequences to app_user;
alter default privileges in schema public grant select, insert, update, delete on tables to admin_user;
alter default privileges in schema public grant usage, select on sequences to admin_user;
alter default privileges in schema public grant select on tables to query_user;

-- app_user/admin_user 分别创建的表都自动授予对方完整 DML 和序列使用权，query_user 仅可查询。
alter default privileges for role app_user in schema public grant select on tables to query_user;
alter default privileges for role app_user in schema public grant select, insert, update, delete on tables to admin_user;
alter default privileges for role app_user in schema public grant usage, select on sequences to admin_user;
alter default privileges for role admin_user in schema public grant select on tables to query_user;
alter default privileges for role admin_user in schema public grant select, insert, update, delete on tables to app_user;
alter default privileges for role admin_user in schema public grant usage, select on sequences to app_user;

-- 4. 常用运维操作（管理用户 admin_user 或 system 执行，按需复制执行）:
--    修改用户密码:  alter user query_user with password 'new_password';
--    删除用户:      drop owned by query_user;  -- 先清权限依赖（在其有权限的库执行）
--                   drop role query_user;
--    新建查询用户:  create role xxx with login password 'xxx123';
--                   grant usage on schema public to xxx;
--                   grant select on all tables in schema public to xxx;
--    查看所有登录用户:  select rolname, rolcanlogin, rolcreaterole from pg_roles where rolcanlogin order by 1;
--    注意: 密码修改统一使用 alter user；openGauss 也建议使用 alter user。
