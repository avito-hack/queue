#!/bin/sh

set -eu

for migrations in /migrations/*; do
    [ -d "$migrations" ] || continue

    service="$(basename "$migrations")"

    case "$service" in
        queue)
            database="$QUEUE_POSTGRES_DB"
            user="$QUEUE_POSTGRES_USER"
            password="$QUEUE_POSTGRES_PASSWORD"
            ;;
        tickets)
            database="$TICKETS_POSTGRES_DB"
            user="$TICKETS_POSTGRES_USER"
            password="$TICKETS_POSTGRES_PASSWORD"
            ;;
        avito-adapter)
            database="$AVITO_ADAPTER_POSTGRES_DB"
            user="$AVITO_ADAPTER_POSTGRES_USER"
            password="$AVITO_ADAPTER_POSTGRES_PASSWORD"
            ;;
        *)
            printf 'Unknown service migrations directory: %s\n' "$service" >&2
            exit 1
            ;;
    esac

    has_migrations=false
    for migration in "$migrations"/*.sql; do
        [ -f "$migration" ] || continue
        has_migrations=true
        break
    done

    if [ "$has_migrations" = false ]; then
        printf 'No migrations for %s\n' "$service"
        continue
    fi

    printf 'Applying migrations for %s\n' "$service"
    PGUSER="$user" \
        PGPASSWORD="$password" \
        GOOSE_DBSTRING="dbname=$database" \
        GOOSE_MIGRATION_DIR="$migrations" \
        goose up
done
