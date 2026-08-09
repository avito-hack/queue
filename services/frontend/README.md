# Frontend — Авито Очередь

Клиентская часть MVP: каталог лимитированных товаров, постановка в очередь, экран «Мои очереди», тикет (право на покупку) и переход к демо-чекауту.

Стек: **React 19**, **TypeScript**, **Vite**, **Redux Toolkit**, **React Router**, **Axios**, **Tailwind CSS 4**, **Vitest**, **ESLint**.

## Запуск

### Вместе со стеком (рекомендуется для проверки сценария)

Из корня репозитория:

```bash
docker compose up -d --build
```

UI: [http://localhost](http://localhost) (через nginx; API на том же origin).

Пересборка только фронта после изменений:

```bash
docker compose up -d --build frontend
```

### Локальная разработка

Нужен поднятый бэкенд/прокси (обычно тот же `docker compose up`), иначе запросы к `/v1/...` не дойдут.

```bash
npm ci
npm run dev      # Vite, обычно http://localhost:5173
```

Переменная `VITE_API_URL` — baseURL для Axios. В Docker для frontend она задаётся пустой строкой (same-origin через nginx). Для `npm run dev` укажите origin API, например `http://localhost`.

```bash
npm run build    # tsc -b && vite build
npm run preview  # превью production-сборки
npm run lint
npm run test
```

Из корня репозитория:

```bash
make lint-frontend
make build-frontend
```

## Сценарий проверки UI

Демо-пользователь поднимается автоматически (`AuthGate` + opaque seed-токен Demo Buyer).

1. `/catalog` — список объявлений с avito-adapter.
2. `/product/:id` — остаток, кнопка встать в очередь / sold-out.
3. После постановки — модалка и переход в `/queue` («Мои очереди»).
4. На плитке: позиция, выход из очереди, при тикете — активация или отказ.
5. Активация ведёт на `/checkout?ticket=<uuid>` (демо-stub оформления; оплата вне MVP).

## Экраны и состояния

| Маршрут | Назначение |
| --- | --- |
| `/catalog` | Каталог лимитированных товаров |
| `/product/:id` | Карточка товара, enqueue, sold-out |
| `/queue` | Плитки очередей и тикетов |
| `/checkout` | Демо-чекаут после активации тикета |

| Состояние | UI | Следующий шаг |
| --- | --- | --- |
| Не в очереди | Кнопка забронировать | Встать в очередь |
| В очереди | Позиция на плитке | Ждать тикет / выйти |
| Есть тикет | Таймер, купить / отказаться | Активировать или decline |
| Sold out | Модалка + «уведомить» | Каталог / подписка (localStorage) |

## Продуктовая логика на клиенте

- Покупка лимитированного товара всегда идёт через очередь/тикет: кнопка на карточке не открывает «голый» чекаут.
- «Мои очереди» — единый экран ожидания: статус, следующий шаг, действия (выйти / активировать / отказаться).
- Тикет персональный: в запросы уходит Bearer; активация с чужим пользователем отклоняется бэкендом.
- После постановки в очередь показываем понятный следующий шаг (модалка → экран очередей), без «серой зоны».
- Sold-out и «уведомить о поступлении» удерживают пользователя в сценарии (уведомление в MVP — localStorage).

## Архитектура фронта

Разделение по зонам ответственности (feature-oriented):

```
src/
  app/                 # store, hooks
  pages/               # экраны (catalog, product, queue, checkout)
  features/
    auth/              # createUser API
    session/           # AuthGate, bootstrap состояния
    product/           # каталог / карточка API + slice
    queue/             # enqueue/dequeue, polling, slice
    ticket/            # list/activate/decline, polling, slice
  shared/
    api/               # axios client, ошибки
    auth/              # demo opaque token
    toast/             # уведомления
  components/          # layout, модалки, ToastHost
```

Состояние: Redux Toolkit. Сеть: Axios + interceptor с Bearer. Синхронизация: polling очередей и тикетов после логина.

## API, которые дергает UI

Через nginx (same-origin) или `VITE_API_URL`:

| Метод | Путь | Зачем |
| --- | --- | --- |
| `POST` | `/v1/avito/users` | демо-регистрация токена |
| `GET` | `/v1/avito/listings` | каталог |
| `GET` | `/v1/avito/listings/{id}` | карточка |
| `POST` | `/v1/queue/{id}/enqueue` | встать в очередь |
| `DELETE` | `/v1/queue/{id}/dequeue` | выйти |
| `GET` | `/v1/user/queues` | мои очереди / позиции |
| `GET` | `/v1/ticket/list` | мои тикеты |
| `POST` | `/v1/ticket/{id}/activate` | право → чекаут |
| `POST` | `/v1/ticket/{id}/decline` | отказаться от тикета |

Контракты бэкенда: `schemas/services/*/openapi.yaml` в корне репозитория.

## Авторизация (демо)

Полноценный логин Avito вне скоупа кейса. Клиент:

1. Берёт opaque seed-токен Demo Buyer (`shared/auth/demoAuth.ts`).
2. Регистрирует его через `POST /v1/avito/users`.
3. Кладёт `authToken` / `userId` в `localStorage` и подставляет `Authorization: Bearer …` во все запросы (кроме `createUser`).

Seed совпадает с миграцией avito-adapter: `00000000-0000-4000-8000-000000000001`.

## ESLint

Конфиг: [`eslint.config.js`](./eslint.config.js) (flat config).

Включено и зачем:

- `@eslint/js` + `typescript-eslint` recommended — базовая гигиена TS/JS;
- `eslint-plugin-react-hooks` — корректность hooks (в т.ч. React 19);
- `eslint-plugin-react-refresh` — совместимость с Vite HMR;
- `@typescript-eslint/no-unused-vars` с игнором `_prefix` — меньше мёртвого кода без шума на намеренно неиспользуемых аргументах.

CI: [`.github/workflows/frontend.yml`](../../.github/workflows/frontend.yml) на `push` в `dev`/`main` гоняет `make lint-frontend`, затем build.

Перед пушем:

```bash
npm run lint && npm run test && npm run build
```

## Тесты

Vitest + Testing Library. Важные зоны:

- demo auth / bootstrap;
- маппинг queue & ticket API;
- интеграционные сценарии страниц Catalog / Product / Queue / Checkout;
- slices и обработка ошибок API.

```bash
npm test
npm run test:watch
```

## Ограничения MVP (фронт)

- `/checkout` — демонстрационный stub после активации тикета (оплата и списание стоков — зона мока Avito / вне UI).
- «Уведомить о поступлении» — только `localStorage`, без push-сервиса.
- Демо-авторизация упрощённая (opaque seed), не полноценный SSO Avito.
- Картинки товаров в каталоге — демо-заглушки, не поля API.

## Использование ИИ

ИИ использовался как помощник: черновики компонентов и тестов, разбор ошибок TypeScript/ESLint, формулировки документации. Продуктовые состояния экранов и привязка к OpenAPI-контрактам согласованы с командой; финальные решения по UX сценария очереди/тикета принимались разработчиками.
