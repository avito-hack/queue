# Frontend (Avito Queue)

Vite + React + TypeScript + Tailwind + Redux Toolkit.

## Scripts

```bash
npm ci
npm run dev      # локальная разработка (обычно :5173)
npm run build    # tsc + vite production build
npm run lint     # ESLint (тот же шаг, что CI Frontend / lint)
npm run test     # Vitest
```

Из корня репозитория:

```bash
make lint-frontend
make build-frontend
```

## ESLint

Конфиг: [`eslint.config.js`](./eslint.config.js) (flat config).

Что включено:

- `@eslint/js` recommended
- `typescript-eslint` recommended
- `eslint-plugin-react-hooks` (в т.ч. правила React 19 про refs / setState в effects)
- `eslint-plugin-react-refresh` для Vite HMR

CI workflow [`.github/workflows/frontend.yml`](../../.github/workflows/frontend.yml) на `push` в `dev`/`main` гоняет `make lint-frontend`, затем `build`. Если lint красный — build в Actions не стартует.

Перед пушем фронта локально:

```bash
cd services/frontend && npm run lint && npm run test && npm run build
```
