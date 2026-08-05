FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/avito-adapter ./cmd/app

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /out/avito-adapter /usr/local/bin/avito-adapter
USER app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["avito-adapter"]
