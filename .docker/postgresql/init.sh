#!/bin/sh

set -eu

create_service_database() {
    service_user="$1"
    service_password="$2"
    service_database="$3"

    psql \
        --set ON_ERROR_STOP=1 \
        --username "$POSTGRES_USER" \
        --dbname "$POSTGRES_DB" \
        --set service_user="$service_user" \
        --set service_password="$service_password" <<'SQL'
SELECT format(
    'CREATE ROLE %I LOGIN PASSWORD %L',
    :'service_user',
    :'service_password'
)
WHERE NOT EXISTS (
    SELECT 1
    FROM pg_catalog.pg_roles
    WHERE rolname = :'service_user'
)
\gexec

SELECT format(
    'ALTER ROLE %I WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %L',
    :'service_user',
    :'service_password'
)
\gexec
SQL

    database_exists="$(
        psql \
            --set ON_ERROR_STOP=1 \
            --username "$POSTGRES_USER" \
            --dbname "$POSTGRES_DB" \
            --tuples-only \
            --no-align \
            --set service_database="$service_database" <<'SQL'
SELECT 1
FROM pg_database
WHERE datname = :'service_database';
SQL
    )"

    if [ "$database_exists" != "1" ]; then
        createdb \
            --username "$POSTGRES_USER" \
            --maintenance-db "$POSTGRES_DB" \
            --owner "$service_user" \
            "$service_database"
    fi

    psql \
        --set ON_ERROR_STOP=1 \
        --username "$POSTGRES_USER" \
        --dbname "$POSTGRES_DB" \
        --set service_user="$service_user" \
        --set service_database="$service_database" <<'SQL'
SELECT format(
    'ALTER DATABASE %I OWNER TO %I',
    :'service_database',
    :'service_user'
)
\gexec

SELECT format(
    'REVOKE CONNECT ON DATABASE %I FROM PUBLIC',
    :'service_database'
)
\gexec

SELECT format(
    'GRANT CONNECT ON DATABASE %I TO %I',
    :'service_database',
    :'service_user'
)
\gexec
SQL
}

: "${QUEUE_POSTGRES_USER:?}"
: "${QUEUE_POSTGRES_PASSWORD:?}"
: "${QUEUE_POSTGRES_DB:?}"
: "${TICKETS_POSTGRES_USER:?}"
: "${TICKETS_POSTGRES_PASSWORD:?}"
: "${TICKETS_POSTGRES_DB:?}"
: "${AVITO_ADAPTER_POSTGRES_USER:?}"
: "${AVITO_ADAPTER_POSTGRES_PASSWORD:?}"
: "${AVITO_ADAPTER_POSTGRES_DB:?}"

create_service_database \
    "$QUEUE_POSTGRES_USER" \
    "$QUEUE_POSTGRES_PASSWORD" \
    "$QUEUE_POSTGRES_DB"

create_service_database \
    "$TICKETS_POSTGRES_USER" \
    "$TICKETS_POSTGRES_PASSWORD" \
    "$TICKETS_POSTGRES_DB"

create_service_database \
    "$AVITO_ADAPTER_POSTGRES_USER" \
    "$AVITO_ADAPTER_POSTGRES_PASSWORD" \
    "$AVITO_ADAPTER_POSTGRES_DB"
