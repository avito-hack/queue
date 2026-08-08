FROM golang:1.26-alpine AS builder

WORKDIR /src/services/queue

COPY services/queue/go.mod services/queue/go.sum ./
RUN go mod download

COPY services/queue/ ./

COPY services/avito-adapter/docs/api/ /src/services/avito-adapter/docs/api/
COPY services/tickets/docs/api/ /src/services/tickets/docs/api/
COPY schemas/services/avito-adapter/ /src/schemas/services/avito-adapter/

RUN mkdir -p /src/services/queue/gen/server

RUN go generate ./...

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/queue ./cmd/app

FROM alpine:3.22

RUN apk add --no-cache ca-certificates wget
RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /out/queue /usr/local/bin/queue
RUN chmod +x /usr/local/bin/queue

USER app
WORKDIR /app
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["queue"]