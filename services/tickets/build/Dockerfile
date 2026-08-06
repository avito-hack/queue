FROM golang:1.24-alpine AS builder

WORKDIR /src/services/tickets
COPY services/tickets/go.mod services/tickets/go.sum ./
RUN go mod download
COPY services/tickets/ ./
COPY schemas/services/avito-adapter/openapi.yaml /src/schemas/services/avito-adapter/openapi.yaml
RUN go generate ./...
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/tickets ./cmd/app

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /out/tickets /usr/local/bin/tickets
USER app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["tickets"]
