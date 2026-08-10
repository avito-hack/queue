# Tickets

`tickets` — сервис прав на покупку лимитированных товаров.

Сервис предоставляет strict Gin-server, сгенерированный из `api/openapi.yaml`, проверяет HTTP-запросы по OpenAPI-контракту, хранит тикеты в PostgreSQL, поддерживает healthcheck и graceful shutdown.

Пользовательский Bearer-токен валидируется через `POST /v1/users/validate` сервиса avito-adapter. Tickets передаёт токен в теле запроса и использует полученный `user_id`; локальной валидации токена в сервисе нет.

Tickets ожидает `200` с телом `{"user_id":"<uuid>"}` для валидного токена и `401` для невалидного. Остальные статусы и некорректный ответ считаются ошибкой зависимости.

Клиент avito-adapter генерируется из `../../schemas/services/avito-adapter/openapi.yaml` командой `make generate`.

Активация тикета напрямую вызывает `CreateOrder` с идентификаторами тикета, объявления, SKU и пользователя. Ручки резервации в этом сценарии не вызываются.

Перед выдачей тикета сервис получает `quantity` объявления через `GetListing`. Выдача сериализуется транзакционным advisory lock по объявлению и разрешается, только если количество действующих `issued`-тикетов не превышает физический остаток.

## Запуск

```bash
DATABASE_URL='postgres://tickets:password@localhost:5432/tickets?sslmode=disable' \
AVITO_ADAPTER_URL='http://localhost:8081' \
RABBITMQ_URL='amqp://tickets:password@localhost:5672/' \
go run ./cmd/app
```

По умолчанию сервер доступен по адресу `http://0.0.0.0:8080`.

```bash
make generate
make test
make run
```

Для контейнерной сборки из корня репозитория:

```bash
docker build -f services/tickets/build/Dockerfile -t tickets .
docker run --rm \
  -e DATABASE_URL="$DATABASE_URL" \
  -e AVITO_ADAPTER_URL="$AVITO_ADAPTER_URL" \
  -e RABBITMQ_URL="$RABBITMQ_URL" \
  -p 8080:8080 tickets
```

## Конфигурация

| Переменная | Значение по умолчанию |
| --- | --- |
| `HTTP_HOST` | `0.0.0.0` |
| `HTTP_PORT` | `8080` |
| `HTTP_READ_TIMEOUT` | `5s` |
| `HTTP_WRITE_TIMEOUT` | `10s` |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` |
| `DATABASE_URL` | обязательная строка подключения к PostgreSQL |
| `DATABASE_CONNECT_TIMEOUT` | `5s` |
| `AVITO_ADAPTER_URL` | обязательный URL avito-adapter |
| `AVITO_ADAPTER_TIMEOUT` | `3s` |
| `RABBITMQ_URL` | обязательный URL RabbitMQ |
| `RABBITMQ_EXCHANGE` | `domain.events` |
| `RABBITMQ_LISTING_EVENTS_QUEUE` | `tickets.listing-events` |
| `TICKET_ACTIVATION_TTL` | `15m` |
| `TICKET_MAINTENANCE_INTERVAL` | `1s` |
| `ACTIVATION_RECOVERY_TIMEOUT` | `1m` |
| `WORKER_BATCH_SIZE` | `100` |
| `OUTBOX_POLL_INTERVAL` | `500ms` |
| `OUTBOX_LEASE` | `30s` |
| `OUTBOX_RETRY_DELAY` | `5s` |
| `OUTBOX_CONCURRENCY` | `4` |

## API

Контракт находится в `api/openapi.yaml`. Реализованы `GET /healthz`, `GET /v1/ticket/list`, `GET /v1/ticket/{ticket_id}`, `POST /v1/ticket/{ticket_id}/activate`, `POST /v1/ticket/{ticket_id}/decline` и `POST /internal/v1/ticket/issue`.

После изменения OpenAPI-схемы необходимо выполнить `make generate`.

## События

Контракт исходящих событий находится в `docs/events/events.yaml`. Tickets публикует `ticket.closed` после отказа, истечения срока активации или отзыва права из-за изменения объявления и `ticket.redeemed` после успешного создания заказа. Сервис получает `listing.quantity.changed` и `listing.status.changed` из durable-очереди RabbitMQ и идемпотентно обрабатывает их через inbox. Оплата заказа и жизненный цикл резервации остаются внутри стандартных сервисов Авито и не обрабатываются tickets.
