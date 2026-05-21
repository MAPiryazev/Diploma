#!/bin/sh
set -eu

psql_admin() {
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" "$@"
}

psql_admin <<EOSQL
DO \$\$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${LEDGER_DB_USER}') THEN
        EXECUTE format('CREATE ROLE %I LOGIN PASSWORD %L', '${LEDGER_DB_USER}', '${LEDGER_DB_PASSWORD}');
    END IF;
END
\$\$;
EOSQL

ledger_exists="$(psql_admin -tAc "SELECT 1 FROM pg_database WHERE datname = '${LEDGER_DB_NAME}'")"
if [ "$ledger_exists" != "1" ]; then
  psql_admin -c "CREATE DATABASE \"${LEDGER_DB_NAME}\" OWNER \"${LEDGER_DB_USER}\""
fi
