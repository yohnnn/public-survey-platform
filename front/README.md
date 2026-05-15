# Frontend

React + TypeScript приложение на Vite.

## Запуск

```bash
npm install
npm run dev
```

Приложение будет доступно на:

```text
http://localhost:3000
```

Backend API ожидается на `http://localhost:8080`. В dev-режиме Vite проксирует `/v1/*` и `/healthz` на backend, а `/__upload_proxy` используется для загрузки изображений в MinIO по presigned URL.

## Production

```bash
npm run build
npm run start
```

`server.mjs` отдает собранный `dist`, проксирует `/v1/*` и `/healthz` на backend, а также поддерживает `/__upload_proxy`.

Настройки:

```text
PORT=3000
API_PROXY_TARGET=http://localhost:8080
```

## Команды

```bash
npm run typecheck
npm run build
npm run start
npm run preview
```
