# Мониторинг (Prometheus + Grafana)

## Запуск

Вместе со стеком:

```bash
docker compose up -d prometheus grafana api-service user-service poll-service vote-service feed-service analytics-service
```

## URL

| Сервис | Адрес | Логин |
|--------|--------|-------|
| Grafana | http://localhost:3001 | admin / admin |
| Prometheus | http://localhost:9090 | — |
| Метрики api-service | http://localhost:9101/metrics | — |

## Дашборды

1. **PSP — обзор для защиты** (`psp-platform-overview`) — статус сервисов, RPS API, gRPC, задержки. Открывается по умолчанию.
2. **Backend Technical Overview** — детальная техническая панель (память, CPU, goroutines, ошибки).

## Что показать на защите (1–2 мин)

1. Открыть Grafana → дашборд «PSP — обзор для защиты».
2. Убедиться, что все 6 сервисов **UP**.
3. В другой вкладке пройтись по сайту (лента, голос, регистрация).
4. Вернуться в Grafana — вырастут графики **HTTP RPS** (api-service) и **gRPC RPS** (микросервисы).

Метрики: `http_requests_*` на api-gateway, `grpc_server_*` на внутренних сервисах (см. `back/pkg/metrics`).
