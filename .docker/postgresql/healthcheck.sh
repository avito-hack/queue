#!/bin/sh

set -eu

check_database() {
    service_user="$1"
    service_password="$2"
    service_database="$3"

    PGPASSWORD="$service_password" \
        PGCONNECT_TIMEOUT=2 \
        psql \
            --host 127.0.0.1 \
            --port 5432 \
            --username "$service_user" \
            --dbname "$service_database" \
            --no-password \
            --tuples-only \
            --no-align \
            --command 'SELECT 1' \
        | grep -q '^1$'
}

check_database \
    "$QUEUE_POSTGRES_USER" \
    "$QUEUE_POSTGRES_PASSWORD" \
    "$QUEUE_POSTGRES_DB"

check_database \
    "$TICKETS_POSTGRES_USER" \
    "$TICKETS_POSTGRES_PASSWORD" \
    "$TICKETS_POSTGRES_DB"

check_database \
    "$AVITO_ADAPTER_POSTGRES_USER" \
    "$AVITO_ADAPTER_POSTGRES_PASSWORD" \
    "$AVITO_ADAPTER_POSTGRES_DB"
