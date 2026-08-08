# Avito Adapter

`avito-adapter` — mock-сервис готовых механизмов Avito, с которыми взаимодействуют сервисы очереди и тикетов. Read-path API получает пользователей, объявления и заказы из PostgreSQL, а команды изменения состояния в текущем MVP хранятся в памяти процесса.

Сервис не реализует регистрацию, полноценную авторизацию, настоящий checkout или списание физического остатка после оплаты. В нём есть mock-валидация Bearer-токена и mock-оплата заказа для интеграционных сценариев.

## Реализованные возможности

Ниже перечислено текущее поведение сервиса. API-контракт находится в [api/openapi.yaml](api/openapi.yaml), HTTP-слой — в [infrastructure/transport/http](infrastructure/transport/http), а правила предметной области — в [internal/usecase](internal/usecase).

| Возможность | Что реализовано | API | Где находится логика |
| --- | --- | --- | --- |
| Пользователи | Создание mock-пользователя, получение по идентификатору и валидация Bearer-токена. | `POST /v1/users`, `POST /v1/users/validate`, `GET /v1/users/{userId}` | [users.go](internal/usecase/users.go) |
| Список объявлений | Получение списка активных товаров с пагинацией и фильтрами. | `GET /v1/listings` | [listings.go](internal/usecase/listings.go) |
| Конфигурация объявления | Создание объявления продавца, чтение, изменение названия и цены. | `POST /v1/listings`, `GET`, `PATCH /v1/listings/{listingId}` | [listings.go](internal/usecase/listings.go) |
| Очередь на объявлении | Поле `queueEnabled` включает или отключает возможность использовать очередь для объявления. Адаптер хранит настройку, но не ведёт FIFO-очередь: это задача queue-service. | `PUT /v1/listings/{listingId}/queue` | [listings.go](internal/usecase/listings.go) |
| Остатки | Изменение физического количества товара и вычисление доступного остатка с учётом резервов. | `PUT /v1/listings/{listingId}/quantity`, поле `availableQuantity` | [models.go](internal/usecase/models.go), [listings.go](internal/usecase/listings.go) |
| Жизненный цикл объявления | Приостановка, повторная активация и снятие объявления. Снятое объявление нельзя редактировать или активировать повторно. | `POST /v1/listings/{listingId}/pause`, `POST /v1/listings/{listingId}/activate`, `DELETE /v1/listings/{listingId}` | [listings.go](internal/usecase/listings.go) |
| Checkout-заказ | Идемпотентное создание заказа для конкретного SKU, выдача checkout URL, чтение и mock-оплата. | `POST /v1/orders`, `GET /v1/orders/{orderId}`, `POST /v1/orders/{orderId}/pay` | [orders.go](internal/usecase/orders.go) |
| HTTP-инфраструктура | Strict Gin-server, валидация запросов по OpenAPI, healthcheck, конфигурация из окружения и graceful shutdown. | `GET /health` | [router.go](infrastructure/transport/http/router.go), [handler.go](infrastructure/transport/http/handler.go), [main.go](cmd/app/main.go) |

## Состояния и инварианты

Состояние объявления хранится в поле `status`:

| Значение | Смысл | Новые блокировки и обычные резервы |
| --- | --- | --- |
| `active` | Объявление опубликовано и доступно. | Разрешены при достаточном остатке. |
| `paused` | Объявление временно приостановлено. | Запрещены. |
| `removed` | Объявление снято долгосрочно. | Запрещены. |

Состояния экрана пользователя, FIFO-очередь, тикеты, резервирование товара, уведомления и очистка очереди при долгосрочном снятии не реализуются этим сервисом. Эти сценарии координируют queue-service и tickets-service; адаптер предоставляет им конфигурацию объявления, физический остаток и checkout flow.

## Запуск

Требуется Go версии 1.23 или новее.

```bash
cd services/avito-adapter
DATABASE_URL='postgres://avito_adapter:password@localhost:5432/avito_adapter?sslmode=disable' go run ./cmd/app
```

По умолчанию сервер доступен по адресу `http://0.0.0.0:8080`.

Также доступны команды:

```bash
make generate
make test
make run
```

Для контейнерной сборки:

```bash
docker build -f build/Dockerfile -t avito-adapter .
docker run --rm -p 8080:8080 avito-adapter
```

## Конфигурация

Конфигурация читается из переменных окружения. Реализация находится в [config/config.go](config/config.go).

| Переменная | Значение по умолчанию | Описание |
| --- | --- | --- |
| `HTTP_HOST` | `0.0.0.0` | Хост HTTP-сервера |
| `HTTP_PORT` | `8080` | Порт HTTP-сервера |
| `HTTP_READ_TIMEOUT` | `5s` | Таймаут чтения HTTP-запроса |
| `HTTP_WRITE_TIMEOUT` | `10s` | Таймаут записи HTTP-ответа |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` | Максимальное время graceful shutdown |
| `DATABASE_URL` | — | Обязательная строка подключения PostgreSQL |
| `POSTGRES_CONNECT_TIMEOUT` | `10s` | Таймаут подключения к PostgreSQL |

Например:

```bash
DATABASE_URL='postgres://avito_adapter:password@localhost:5432/avito_adapter?sslmode=disable' HTTP_PORT=8081 HTTP_SHUTDOWN_TIMEOUT=15s go run ./cmd/app
```

## API

Полный машиночитаемый контракт расположен в [api/openapi.yaml](api/openapi.yaml). Все бизнес-ручки используют префикс `/v1`, JSON-поля используют `camelCase`, идентификаторы ресурсов — UUID.

### Служебная ручка

| Метод | Путь | Назначение |
| --- | --- | --- |
| `GET` | `/health` | Проверка доступности сервиса. Возвращает `200` и `{ "status": "ok" }`. |

### Пользователи

Для mock-авторизации пользователь создаётся с токеном. В production-версии создание пользователей заменяется внешним провайдером идентификации.

| Метод | Путь | Назначение | Успешный ответ |
| --- | --- | --- | --- |
| `POST` | `/v1/users` | Создать пользователя. Тело: `name`, `token`. | `201`, объект пользователя |
| `POST` | `/v1/users/validate` | Валидировать raw-токен или `Bearer <token>`. | `200`, `user_id` |
| `GET` | `/v1/users/{userId}` | Получить пользователя. | `200`, объект пользователя |

Пользователь содержит `id`, `name` и `createdAt`.

### Объявления

| Метод | Путь | Назначение | Успешный ответ |
| --- | --- | --- | --- |
| `GET` | `/v1/listings` | Получить список объявлений. | `200`, страница объявлений |
| `POST` | `/v1/listings` | Создать объявление. | `201`, объект объявления |
| `GET` | `/v1/listings/{listingId}` | Получить объявление и текущие остатки. | `200`, объект объявления |
| `PATCH` | `/v1/listings/{listingId}` | Изменить `title` и/или `price`. | `200`, объект объявления |
| `PUT` | `/v1/listings/{listingId}/quantity` | Установить физическое количество `quantity`. | `200`, объект объявления |
| `PUT` | `/v1/listings/{listingId}/queue` | Включить или отключить очередь. Тело: `enabled`. | `200`, объект объявления |
| `POST` | `/v1/listings/{listingId}/pause` | Приостановить объявление. | `200`, объект объявления |
| `POST` | `/v1/listings/{listingId}/activate` | Активировать ранее приостановленное объявление. | `200`, объект объявления |
| `DELETE` | `/v1/listings/{listingId}` | Снять объявление. | `204` |

Тело `POST /v1/listings` содержит:

- `sellerId` — существующий пользователь-продавец;
- `title` — название;
- `price` — цена в минимальных денежных единицах;
- `quantity` — физический остаток;
- `queueEnabled` — необязательный флаг доступности очереди, по умолчанию `false`.

Объект объявления возвращает:

- `quantity` — физический остаток;
- `queueEnabled` — настройку очереди;
- `status` — `active`, `paused` или `removed`;
- конфигурацию объявления и временные метки.

`GET /v1/listings` по умолчанию возвращает только объявления со статусом `active`. Для списка доступны query-параметры:

- `sellerId` — UUID продавца;
- `status` — `active`, `paused` или `removed`; переопределяет статус по умолчанию;
- `limit` — размер страницы от `1` до `100`, по умолчанию `20`;
- `offset` — смещение, по умолчанию `0`.

Ответ содержит `items`, `total`, `limit` и `offset`. Объявления упорядочены от новых к старым.

### Заказы

| Метод | Путь | Назначение | Успешный ответ |
| --- | --- | --- | --- |
| `POST` | `/v1/orders` | Создать checkout-заказ для SKU. | `201`, объект заказа |
| `GET` | `/v1/orders/{orderId}` | Получить заказ. | `200`, объект заказа |
| `POST` | `/v1/orders/{orderId}/pay` | Выполнить mock-оплату. | `200`, оплаченный заказ |

Тело `POST /v1/orders/create` содержит `ticketId`, `listingId`, `skuId` и `userId`, а UUID-ключ идемпотентности передаётся в обязательном заголовке `Idempotency-Key`. Повторный запрос с тем же ключом и теми же параметрами возвращает исходный заказ. Повтор ключа с другим телом возвращает `409`. Создание резервации, заказа и `checkoutUrl` выполняется атомарно.

Новый заказ имеет статус `created`. `POST /v1/orders/{orderId}/pay` меняет его на `paid`; физический остаток при этом не списывается.

### Ошибки

Ошибочные JSON-ответы содержат поле `message`.

| Код | Значение |
| --- | --- |
| `400` | Невалидное тело запроса, недопустимое значение или недопустимое изменение конфигурации объявления. |
| `401` | Bearer-токен невалиден. |
| `404` | Пользователь, объявление или заказ не найден. |
| `409` | Конфликт идемпотентности, неактивное объявление или повторная оплата заказа. |

## Пример сценария

1. Создать продавца и покупателя через `POST /v1/users`.
2. Создать активное объявление продавца через `POST /v1/listings`.
3. При необходимости включить очередь через `PUT /v1/listings/{listingId}/queue`.
4. Валидировать токен покупателя через `POST /v1/users/validate`.
5. Создать checkout-заказ по SKU через `POST /v1/orders`.
6. Выполнить mock-оплату через `POST /v1/orders/{orderId}/pay`.

Пользователи, объявления и заказы читаются из PostgreSQL. Изменения, выполненные через command-endpoints, пока хранятся in-memory и пропадают после перезапуска процесса.

## Архитектура

| Расположение | Содержимое |
| --- | --- |
| [cmd/app/main.go](cmd/app/main.go) | Запуск HTTP-сервера и graceful shutdown. |
| [config/config.go](config/config.go) | Чтение HTTP-конфигурации из окружения. |
| [api/openapi.yaml](api/openapi.yaml) | Серверный OpenAPI-контракт. |
| [gen/server](gen/server) | Код strict Gin-сервера, генерируемый из OpenAPI. Не редактируется вручную. |
| [internal/usecase](internal/usecase) | In-memory use cases: модели, ошибки, пользователи, объявления, резервы и заказы. |
| [internal/usecase/health.go](internal/usecase/health.go) | Use case healthcheck. |
| [infrastructure/transport/http](infrastructure/transport/http) | HTTP-роутер, OpenAPI-валидация и обработчики. |
| [build/Dockerfile](build/Dockerfile) | Образ сервиса. |

## Генерация и тесты

После изменения [api/openapi.yaml](api/openapi.yaml) необходимо выполнить `make generate`. Конфигурация генератора — [api/oapi-codegen.yaml](api/oapi-codegen.yaml).

Тесты проверяют healthcheck, уменьшение доступного остатка при резервировании, восстановление остатка при отмене и запрет повторного заказа по одному резерву. Они запускаются командой:

```bash
make test
```
