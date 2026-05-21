#!/bin/sh
set -eu

psql_admin() {
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" "$@"
}

psql_admin <<EOSQL
DO \$\$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${ANALYTICS_DB_USER}') THEN
        EXECUTE format('CREATE ROLE %I LOGIN PASSWORD %L', '${ANALYTICS_DB_USER}', '${ANALYTICS_DB_PASSWORD}');
    END IF;
END
\$\$;
EOSQL

analytics_exists="$(psql_admin -tAc "SELECT 1 FROM pg_database WHERE datname = '${ANALYTICS_DB_NAME}'")"
if [ "$analytics_exists" != "1" ]; then
  psql_admin -c "CREATE DATABASE \"${ANALYTICS_DB_NAME}\" OWNER \"${ANALYTICS_DB_USER}\""
fi
