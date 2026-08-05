# Tickets

`tickets` — каркас сервиса прав на покупку лимитированных товаров.

Сервис предоставляет strict Gin-server, сгенерированный из `api/openapi.yaml`, проверяет HTTP-запросы по OpenAPI-контракту, поддерживает healthcheck и graceful shutdown. Бизнес-операции пока возвращают `500` с кодом `not_implemented`.

Заглушка проверяет наличие непустого Bearer-токена. Валидация токена и получение `user_id` намеренно не реализованы.

## Запуск

```bash
go run ./cmd/app
```

По умолчанию сервер доступен по адресу `http://0.0.0.0:8080`.

```bash
make generate
make test
make run
```

Для контейнерной сборки:

```bash
docker build -f build/Dockerfile -t tickets .
docker run --rm -p 8080:8080 tickets
```

## Конфигурация

| Переменная | Значение по умолчанию |
| --- | --- |
| `HTTP_HOST` | `0.0.0.0` |
| `HTTP_PORT` | `8080` |
| `HTTP_READ_TIMEOUT` | `5s` |
| `HTTP_WRITE_TIMEOUT` | `10s` |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` |

## API

Контракт находится в `api/openapi.yaml`. Рабочая служебная ручка — `GET /healthz`. Все остальные операции являются заглушками.

После изменения OpenAPI-схемы необходимо выполнить `make generate`.
