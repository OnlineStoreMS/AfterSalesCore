#!/usr/bin/env bash
# 修复 aftersalescore 库权限：表由 postgres 超级用户创建时，应用用户无法 AutoMigrate / 建索引。
# 用法（需能连 postgres 超级用户）:
#   ./deploy/fix_db_permissions.sh
#   PGHOST=127.0.0.1 ./deploy/fix_db_permissions.sh

set -euo pipefail

DB_NAME="${DB_NAME:-aftersalescore}"
DB_USER="${DB_USER:-aftersalescore}"
PGHOST="${PGHOST:-127.0.0.1}"

psql -h "$PGHOST" -U postgres -d "$DB_NAME" -v ON_ERROR_STOP=1 <<SQL
GRANT USAGE ON SCHEMA public TO ${DB_USER};
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ${DB_USER};
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ${DB_USER};
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ${DB_USER};
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO ${DB_USER};

DO \$\$
DECLARE r record;
BEGIN
  FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'public'
  LOOP
    EXECUTE format('ALTER TABLE public.%I OWNER TO ${DB_USER}', r.tablename);
  END LOOP;
  FOR r IN SELECT sequence_name FROM information_schema.sequences WHERE sequence_schema = 'public'
  LOOP
    EXECUTE format('ALTER SEQUENCE public.%I OWNER TO ${DB_USER}', r.sequence_name);
  END LOOP;
END
\$\$;
SQL

echo "permissions fixed for ${DB_USER} on ${DB_NAME}"
