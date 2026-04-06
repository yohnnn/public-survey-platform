# Polling Platform Architecture

Платформа для создания и голосования в опросах с бесконечной лентой, фильтрацией по тегам, **расширенной аналитикой** и realtime обновлением голосов.

---

# Технологии

* Go
* gRPC
* gRPC Gateway (HTTP → gRPC)
* Apache Kafka
* PostgreSQL
* Redis
* WebSocket / SSE
* Docker

---

# Общая архитектура

```
Client (Web / Mobile)
        │
        ▼
HTTP API (gRPC Gateway)
        │
        ▼
Microservices
 ├── Auth Service
 ├── Poll Service
 ├── Vote Service
 ├── Analytics Service
 ├── Feed Service
 └── Realtime Service
        │
        ▼
Infrastructure
 ├── Kafka
 ├── PostgreSQL (5 databases)
 └── Redis
```

---

# Сервисы

## 1. Auth Service

Отвечает за пользователей и авторизацию.

### База данных

`auth_db`

### Таблицы

```sql
users
-----
id (uuid)
email
password_hash
created_at
country
gender
birth_year

refresh_sessions
----------------
id (uuid)
user_id
refresh_token_hash
expires_at
created_at
revoked_at
```

### gRPC API

```
Register
Login
RefreshToken
Logout
LogoutAll
ValidateToken
GetUser
```

### HTTP API

```
POST /v1/auth/register
POST /v1/auth/login
POST /v1/auth/refresh
POST /v1/auth/logout
POST /v1/auth/logout-all
GET /v1/auth/me
```

### Результат

Пара токенов для авторизации:

* `access_token` (короткоживущий)
* `refresh_token` (долгоживущий, с хранением сессии)

---

# 2. Poll Service

Отвечает за создание и управление опросами **и тегами**.

### База данных

`poll_db`

### Таблицы

```sql
polls
-----
id
creator_id
question
type
is_anonymous
ends_at
created_at
total_votes

poll_options
-----
id
poll_id
text
votes_count

tags
-----
id
name
created_at

poll_tags
---------
poll_id
tag_id
```

### Индексы

```sql
CREATE INDEX idx_poll_tags_poll ON poll_tags(poll_id);
CREATE INDEX idx_poll_tags_tag ON poll_tags(tag_id);
```

### Типы опросов

```
SINGLE_CHOICE
MULTIPLE_CHOICE
```

### gRPC API

```
CreatePoll
GetPoll
ListPolls
UpdatePoll
DeletePoll
CreateTag
ListTags
```

### HTTP API

```
POST /v1/polls
GET /v1/polls
GET /v1/polls/{id}
PATCH /v1/polls/{id}
DELETE /v1/polls/{id}

POST /v1/tags
GET /v1/tags
```

### Kafka события

```
poll.created
poll.deleted
```

### Kafka подписки

Poll Service подписывается на:

```text
vote.cast
vote.removed
```

и обновляет `polls.total_votes` как быстрый счетчик для ленты и карточки опроса.

Также обновляет `poll_options.votes_count` для быстрого отображения голосов по каждому варианту.

`GET /v1/polls/{id}` должен возвращать варианты с текущими счетчиками:

```json
{
  "id": "poll_123",
  "question": "Какой язык вы используете чаще всего?",
  "total_votes": 1520,
  "options": [
    {"id": "1", "text": "Go", "votes_count": 700},
    {"id": "2", "text": "Python", "votes_count": 520},
    {"id": "3", "text": "Java", "votes_count": 300}
  ]
}
```

---

# 3. Vote Service

Отвечает за голосование пользователей.

### База данных

`vote_db`

### Таблицы

```sql
votes
-----
user_id
poll_id
option_id
created_at

UNIQUE(user_id, poll_id)
```

### gRPC API

```
Vote
RemoveVote
GetUserVote
```

### HTTP API

```
POST /v1/polls/{id}/vote
DELETE /v1/polls/{id}/vote
GET /v1/polls/{id}/vote
```

`GET /v1/polls/{id}/vote` возвращает текущий выбор авторизованного пользователя в конкретном опросе.

Если пользователь еще не голосовал, endpoint возвращает `200 OK`:

```json
{
  "poll_id": "123",
  "has_voted": false,
  "option_ids": [],
  "voted_at": null
}
```

Пример:

```json
{
  "poll_id": "123",
  "has_voted": true,
  "option_ids": ["2"],
  "voted_at": "2026-03-22T12:30:00Z"
}
```

### Kafka события

```
vote.cast
vote.removed
```

---

# 4. Analytics Service

Сервис аналитики голосования.

Он позволяет получать **интересную статистику по каждому опросу**.

Примеры аналитики:

* распределение голосов
* голоса по странам
* голоса по полу
* голоса по возрасту
* динамика голосования

### База данных

`analytics_db`

---

### Таблицы

#### Голоса по вариантам

```sql
poll_option_votes
-----------------
poll_id
option_id
votes_count
```

---

#### Голоса по странам

```sql
poll_country_stats
------------------
poll_id
country
votes_count
```

---

#### Голоса по полу

```sql
poll_gender_stats
-----------------
poll_id
gender
votes_count
```

---

#### Голоса по возрасту

```sql
poll_age_stats
--------------
poll_id
age_range
votes_count
```

Пример возрастных групп:

```
18-24
25-34
35-44
45+
```

---

### Источник данных

Analytics Service подписывается на события Kafka:

```
vote.cast
vote.removed
```

---

### Поток данных

```
Vote Service
      │
      ▼
Kafka
      │
      ▼
Analytics Service
      │
      ▼
analytics_db
```

---

### gRPC API

```
GetPollAnalytics
GetCountryStats
GetGenderStats
GetAgeStats
```

---

### HTTP API

```
GET /v1/polls/{id}/analytics
GET /v1/polls/{id}/analytics?from=2026-03-01T00:00:00Z&to=2026-03-22T00:00:00Z&interval=day
GET /v1/polls/{id}/analytics/countries
GET /v1/polls/{id}/analytics/gender
GET /v1/polls/{id}/analytics/age
```

---

### Пример ответа

```json
{
  "poll_id": "123",
  "total_votes": 1520,
  "options": [
    {"option_id": "1", "votes": 700},
    {"option_id": "2", "votes": 520},
    {"option_id": "3", "votes": 300}
  ],
  "countries": [
    {"country": "US", "votes": 620},
    {"country": "DE", "votes": 240},
    {"country": "FR", "votes": 180}
  ],
  "gender": [
    {"gender": "male", "votes": 800},
    {"gender": "female", "votes": 720}
  ]
}
```

---

# 5. Feed Service

Отвечает за бесконечную ленту опросов и фильтрацию по тегам.

### База данных

`feed_db`

### Таблицы

```sql
feed_cache
-------------
poll_id          TEXT PRIMARY KEY
question         TEXT NOT NULL
creator_id       TEXT NOT NULL
type             TEXT NOT NULL
is_anonymous     BOOLEAN NOT NULL
ends_at          TIMESTAMPTZ
created_at       TIMESTAMPTZ NOT NULL
total_votes      INTEGER NOT NULL DEFAULT 0
updated_at       TIMESTAMPTZ NOT NULL
```

```sql
feed_options
-------------
poll_id          TEXT NOT NULL
option_id        TEXT NOT NULL
text             TEXT NOT NULL
votes_count      INTEGER NOT NULL DEFAULT 0
PRIMARY KEY (poll_id, option_id)
```

```sql
feed_tags
----------
poll_id          TEXT NOT NULL
tag_name         TEXT NOT NULL
PRIMARY KEY (poll_id, tag_name)
```

### Индексы

```sql
CREATE INDEX idx_feed_cache_created ON feed_cache(created_at DESC);
CREATE INDEX idx_feed_cache_tags ON feed_tags(tag_name);
```

### Kafka подписки

Feed Service подписывается на события:

```text
poll.created
poll.deleted
poll.updated
vote.cast
vote.removed
```

При получении событий обновляет таблицы:
- `poll.created` → `INSERT` в feed_cache, feed_options, feed_tags
- `poll.updated` → `UPDATE` feed_cache, feed_options
- `poll.deleted` → `DELETE` из всех таблиц
- `vote.cast/vote.removed` → `UPDATE total_votes` и `votes_count`

### Поток данных

```
Poll Service
      │
      ▼
Kafka (poll.created, poll.deleted, poll.updated, vote.cast, vote.removed)
      │
      ▼
Feed Service
      │
      ▼
feed_db (feed_cache, feed_options, feed_tags)
```

### Основные функции

* генерация ленты из feed_db
* сортировка
* пагинация
* фильтрация по тегам
* синхронизация данных через Kafka события (poll.*, vote.*)
* объединение с Analytics Service только для детальной аналитики/виджетов

### gRPC API

```
GetFeed
GetTrending
GetUserPolls
```

### HTTP API

```
GET /v1/feed?cursor=<cursor>&limit=20&sort=new
GET /v1/feed/trending
GET /v1/feed?tag=technology
GET /v1/feed?tags=technology,ai
GET /v1/feed/user/{user_id}?cursor=<cursor>&limit=20
```

Ответ ленты должен содержать `next_cursor`.

Минимальный элемент ленты:

```json
{
  "id": "poll_123",
  "question": "Какой язык вы используете чаще всего?",
  "total_votes": 1520,
  "created_at": "2026-03-22T12:00:00Z"
}
```

### Pagination

Используется **cursor pagination** вместо OFFSET.

```sql
SELECT *
FROM polls
WHERE created_at < cursor
ORDER BY created_at DESC
LIMIT 20
```

---

# 6. Realtime Service

Сервис realtime обновлений голосов.

### Технология

* WebSocket
* Server-Sent Events (SSE)

### Источник данных

Подписка на Kafka события:

```
vote.cast
```

### Поток данных

```
Vote Service
     │
     ▼
Kafka
     │
     ▼
Realtime Service
     │
     ▼
WebSocket
     │
     ▼
Client UI Update
```

Обновление интерфейса происходит примерно за **50–150 ms**.

### HTTP API

```
GET /v1/realtime/polls/{id}/stream   (SSE)
GET /v1/ws                           (WebSocket handshake)
```

---

# Kafka события

Используется event-driven архитектура.

### Topics

```
poll.created
poll.deleted

vote.cast
vote.removed
```

### Пример события

```json
{
  "event": "vote.cast",
  "poll_id": "123",
  "option_id": "2",
  "user_id": "456",
  "timestamp": 171000000
}
```

---

# Базы данных

Каждый сервис владеет своей базой данных.

```
auth_db         (PostgreSQL)
poll_db         (PostgreSQL)
vote_db         (PostgreSQL)
analytics_db    (PostgreSQL)
feed_db         (PostgreSQL)
realtime_db     (Redis)
```

Основной принцип:

```
1 service = 1 database
```

Никакой сервис **не делает прямые SQL запросы к базе другого сервиса**.

---

# Структура проекта

```
backend/

proto/
 ├── auth.proto
 ├── poll.proto
 ├── vote.proto
 ├── analytics.proto
 └── feed.proto

gateway/

auth-service/
poll-service/
vote-service/
analytics-service/
feed-service/
realtime-service/

infra/
 ├── docker-compose
 ├── kafka
 └── migrations
```

---

# Поток голосования

```
User votes
    │
    ▼
Vote Service
    │
    ▼
vote_db
    │
    ▼
Kafka event (vote.cast)
    │
  ├── Poll Service
  │       │
  │       ▼
  │    poll_db: polls.total_votes + 1
  │    poll_db: poll_options.votes_count + 1 (для выбранного option_id)
  │
    ├── Analytics Service
    │       │
    │       ▼
    │    analytics_db update
    │
    └── Realtime Service
            │
            ▼
         WebSocket
            │
            ▼
         Client UI
```

---

# Поток ленты

```
Client
  │
  ▼
GET /feed
  │
  ▼
Feed Service
  │
  ├── Poll Service (опрос + total_votes)
  └── Analytics Service (опционально: расширенные виджеты)
  │
  ▼
Feed Response
```

---

# Архитектурные правила

## Нельзя

```
SELECT COUNT(*) FROM votes
OFFSET pagination
Cross-service SQL queries
LIKE '%tag%'
```

## Нужно

```
Kafka events
Cursor pagination
Aggregated analytics tables
Realtime WebSocket updates
Tag join tables
```

---

# Технические endpoint'ы

Для каждого сервиса:

```
GET /healthz
GET /readyz
GET /metrics
```

---

# Единый формат API-ответов

## Успешный ответ

```json
{
  "data": {},
  "meta": {
    "request_id": "8f5b4f2a-0f24-45da-9dcb-4f6e5c9f9d31",
    "timestamp": "2026-03-22T13:00:00Z"
  }
}
```

## Ответ с ошибкой

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "invalid poll_id",
    "details": {
      "field": "poll_id"
    }
  },
  "meta": {
    "request_id": "8f5b4f2a-0f24-45da-9dcb-4f6e5c9f9d31",
    "timestamp": "2026-03-22T13:00:00Z"
  }
}
```

## HTTP статусы

```text
200 OK
201 Created
204 No Content
400 Bad Request
401 Unauthorized
403 Forbidden
404 Not Found
409 Conflict
422 Unprocessable Entity
429 Too Many Requests
500 Internal Server Error
```

## Pagination (Feed)

```json
{
  "data": {
    "items": []
  },
  "meta": {
    "next_cursor": "2026-03-22T12:00:00Z_4f0a",
    "limit": 20,
    "has_more": true,
    "request_id": "8f5b4f2a-0f24-45da-9dcb-4f6e5c9f9d31"
  }
}
```

## Заголовки

```text
Authorization: Bearer <access_token>
X-Request-ID: <uuid> (опционально от клиента)
```

---

# Итог

Архитектура включает **6 микросервисов**:

```
Auth Service
Poll Service
Vote Service
Analytics Service
Feed Service
Realtime Service
```

Инфраструктура:

```
Kafka
PostgreSQL (5 баз данных)
Redis (realtime_db)
gRPC Gateway
Docker
```

**Базы данных:**

```
auth_db         (PostgreSQL)
poll_db         (PostgreSQL)
vote_db         (PostgreSQL)
analytics_db    (PostgreSQL)
feed_db         (PostgreSQL)
realtime_db     (Redis)
```

Система поддерживает:

* realtime голосование
* бесконечную ленту
* фильтрацию по тегам
* расширенную аналитику опросов
* аналитику по странам, полу и возрасту
* масштабирование до миллионов голосов
