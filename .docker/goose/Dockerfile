FROM golang:1.25-alpine AS builder

ARG GOOSE_VERSION=v3.27.1
RUN CGO_ENABLED=0 GOBIN=/out go install \
    -tags="no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb" \
    github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S goose \
    && adduser -S goose -G goose
COPY --from=builder /out/goose /usr/local/bin/goose
COPY entrypoint.sh /usr/local/bin/migrate
RUN chmod 755 /usr/local/bin/goose /usr/local/bin/migrate

USER goose
ENTRYPOINT ["/usr/local/bin/migrate"]
