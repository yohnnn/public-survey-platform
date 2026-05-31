# Public Survey Platform

Веб-платформа публичных опросов: создание и прохождение опросов, лента публикаций, подписки и аналитика результатов. Серверная часть — микросервисы на Go, клиент — React и TypeScript.

## Стек

- **Backend:** Go, gRPC, HTTP API (grpc-gateway), Apache Kafka, PostgreSQL, Redis, MinIO
- **Frontend:** React, TypeScript, Vite
- **Инфраструктура:** Docker Compose, Prometheus, Grafana

## Сервисы

| Сервис | Назначение |
|--------|------------|
| `user-service` | Пользователи, JWT, подписки |
| `poll-service` | Опросы, теги, изображения |
| `vote-service` | Голосование |
| `feed-service` | Лента, тренды, подписки |
| `analytics-service` | Агрегированная статистика |
| `api-service` | HTTP API-шлюз для клиента |

## Быстрый старт

Требования: Docker и Docker Compose.

```bash
cd back
make compose-up
```

После запуска:

| Сервис | URL |
|--------|-----|
| Веб-интерфейс | http://localhost:3000 |
| HTTP API | http://localhost:8080 |
| Swagger UI | http://localhost:8082 |
| Grafana | http://localhost:3001 (admin / admin) |
| Prometheus | http://localhost:9090 |

Остановка:

```bash
cd back
make compose-down
```

## Структура репозитория

```
back/          — микросервисы, общие пакеты, protobuf, e2e-тесты
front/         — клиентское приложение
docker-compose.yml
```

## Разработка

Команды Makefile выполняются из каталога `back/`:

```bash
make help           # список целей
make proto-generate # генерация gRPC/OpenAPI из proto
make test-unit      # модульные тесты
make test-e2e       # сквозные тесты (нужен поднятый compose)
make compose-logs   # логи контейнеров
```

Нагрузочное тестирование: `back/k6/loadtest.js` (k6, API на `localhost:8080`).

