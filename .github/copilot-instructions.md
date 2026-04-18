# Project Guidelines

## Code Style
- Предпочитай минимальные изменения и сохраняй существующий стиль Go в `back/`.
- Основной код находится в `back/`, это единый Go-модуль. Общие утилиты живут в `back/pkg/`, а код сервисов - в `back/services/<service>/internal/`.
- Сгенерированный код в `back/api/gen/` не редактируй вручную. При изменении protobuf или OpenAPI сначала обновляй генерацию.
- Для HTTP-слоя учитывай, что `api-service` - это gateway поверх gRPC (`grpc-gateway`), а сервисы общаются между собой по gRPC.

## Architecture
- Доменные границы уже разделены на отдельные сервисы: `auth`, `poll`, `vote`, `feed`, `analytics`.
- Локальный стек поднимается через `docker-compose.yml`: там описаны Kafka/Zookeeper, PostgreSQL, миграции и сервисы.
- E2E-проверки ожидают, что API доступно на `http://localhost:8080` и что `/healthz` отвечает `200 OK`.

## Build and Test
- В `back/Makefile` есть команды `proto-lint`, `proto-breaking`, `proto-deps-update` и `proto-generate`.
- После изменений в protobuf запускай `proto-generate`, чтобы синхронизировать `back/api/gen/` и `back/services/api-service/api/openapi.yml`.
- Для проверки backend используй `go test ./tests/e2e/` из `back/`; тест пропускается, если сервисы не подняты.

## Conventions
- Не дублируй документацию из README или спецификаций, если достаточно ссылки на существующий файл.
- При изменениях HTTP API держи синхронными protobuf, gateway-обработчики и `back/services/api-service/api/openapi.yml`.
- Учитывай заголовки `Authorization` и `X-Request-Id`, а также CORS-логику gateway при изменениях request/response поведения.