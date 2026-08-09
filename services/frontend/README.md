# Frontend

`frontend` — веб-клиент сценария очереди к лимитированным товарам.

Приложение на React + TypeScript (Vite), стейт в Redux Toolkit, HTTP через Axios. В Docker собирается в статику и отдаётся nginx-контейнером сервиса; в полном стеке снаружи доступен через корневой `nginx` на порту `80`.

Полноценный логин Avito вне скоупа кейса. При старте `AuthGate` поднимает demo opaque Bearer (seed Demo Buyer), регистрирует его через `POST /v1/avito/users` и кладёт токен в `localStorage`. Axios interceptor добавляет `Authorization: Bearer <token>` ко всем запросам, кроме создания пользователя.

Клиент не валидирует токен сам: `user_id` и права проверяют backend-сервисы.

После авторизации клиент гидратирует состояние из `GET /v1/user/queues` и `GET /v1/ticket/list`, далее синхронизирует позиции и тикеты polling’ом. Покупка лимитированного товара в UI всегда идёт через enqueue → тикет → `activate`; страница `/checkout` — демо-stub после активации (оплата и стоки остаются в моке Avito).

## Запуск

Локальная разработка (при поднятом docker compose):

```bash
npm ci
npm run dev
```

Контейнерная сборка из корня (предпочтительный вариант):

```bash
docker compose up -d --build frontend
```

Либо:

```bash
docker build -f services/frontend/Dockerfile \
  --build-arg VITE_API_URL="" \
  -t frontend \
  services/frontend
docker run --rm -p 8088:80 frontend
```

В полном стеке UI: `http://localhost` (через корневой nginx).

## Конфигурация

| Переменная | Значение по умолчанию | Назначение |
| --- | --- | --- |
| `VITE_PUBLIC_BASE_URL` | `http://localhost:8080` (в коде Axios) | baseURL для API; вшивается на этапе `vite build` |
| `VITE_PUBLIC_BASE_URL` (compose) | `""` | same-origin запросы через корневой nginx |

## Экраны

| Маршрут | Назначение |
| --- | --- |
| `/catalog` | список объявлений (`GET /v1/avito/listings`) |
| `/product/:id` | карточка товара |
| `/queue` | «Мои очереди»: плитки очередей и тикетов |
| `/checkout` | демо-чекаут после `POST /v1/ticket/{id}/activate` |

## API

Клиент ходит в backend через префиксы корневого nginx (или `VITE_API_BASE_URL`):

| Метод | Путь | Назначение |
| --- | --- | --- |
| `POST` | `/v1/avito/users` | регистрация demo-токена |
| `GET` | `/v1/avito/listings` | каталог |
| `GET` | `/v1/avito/listings/{id}` | карточка |
| `POST` | `/v1/queue/{itemID}/enqueue` | встать в очередь |
| `DELETE` | `/v1/queue/{itemID}/dequeue` | выйти из очереди |
| `GET` | `/v1/user/queues` | очереди текущего пользователя |
| `GET` | `/v1/ticket/list` | тикеты текущего пользователя |
| `GET` | `/v1/ticket/{ticket_id}` | тикет по id |
| `POST` | `/v1/ticket/{ticket_id}/activate` | активировать право на покупку |
| `POST` | `/v1/ticket/{ticket_id}/decline` | отказаться от тикета |

Контракты: `../../schemas/services/*/openapi.yaml`.

## Структура

```
src/
  app/           # store, typed hooks
  pages/         # экраны
  features/      # auth, session, product, queue, ticket
  shared/        # api client, demo auth, toast, errors
  components/    # layout, модалки, ToastHost
```

## Линтер и тесты

ESLint flat config: `eslint.config.js` (`@eslint/js`, `typescript-eslint`, `react-hooks`, `react-refresh`). CI: `.github/workflows/frontend.yml` → `make lint-frontend`, затем build.

Тесты: Vitest + Testing Library (`npm test`) — auth bootstrap, API-маппинг, сценарии страниц Catalog / Product / Queue / Checkout.
