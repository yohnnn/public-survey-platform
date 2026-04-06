# Event-driven контур (Outbox Pattern)

## Что реализовано

- `poll-service` пишет событие `poll.created` в таблицу `outbox_events` в **той же транзакции**, что и создание опроса.
- `vote-service` пишет события `vote.cast` и `vote.removed` в `outbox_events` в **той же транзакции**, что и изменение голосов.
- В обоих сервисах запущен relay-воркер, который:
  - читает неопубликованные события из `outbox_events`;
  - публикует их через `events.Publisher`;
  - помечает событие как опубликованное (`published_at`) или увеличивает `attempts` с `last_error`.

## Структура

- Общие события и контракты:
  - `api/events/topics.go`
  - `api/events/contracts.go`
  - `api/events/log_publisher.go`
  - `api/events/kafka.go`
  - `api/events/consumer.go` (подготовка consumer для feed/analytics/realtime)
- Общий relay:
  - `pkg/outbox/relay.go`
  - `pkg/outbox/clock.go`

## Outbox таблицы

Добавлены в миграции `poll-service` и `vote-service`:

```sql
outbox_events (
  id TEXT PRIMARY KEY,
  topic TEXT NOT NULL,
  event_key TEXT NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  published_at TIMESTAMPTZ,
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT
)
```

## Publisher

Поддерживаются два режима публикации:

- `EVENT_PUBLISHER=log` → `LogPublisher` (локальная отладка)
- `EVENT_PUBLISHER=kafka` → `KafkaPublisher` (реальный брокер)

Kafka настраивается переменными:

- `KAFKA_BROKERS` (CSV список брокеров)
- `KAFKA_TOPIC_PREFIX` (необязательный префикс топиков)
- `KAFKA_WRITE_TIMEOUT`

## Следующий шаг

1. Поднять Kafka в окружении и включить `EVENT_PUBLISHER=kafka`.
2. Реализовать прикладные consumer-хендлеры в `feed` / `analytics` / `realtime`
  на базе `events.Subscriber` + `events.Consumer`.
