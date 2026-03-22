# Сервис потоковой обработки финансовых транзакций

## Обзор проекта
Проект представляет собой учебный сервис для работы с финансовыми транзакциями пользователей. Базовые данные и основная бизнес-логика находятся в PostgreSQL, а Kafka используется как шина событий для асинхронной обработки факта создания транзакции.

Сервис поддерживает:

- HTTP API для CRUD-операций по транзакциям.
- Web UI из каталога `web/`.
- Аналитику по транзакциям за период.
- Идемпотентность для `POST /items` через заголовок `Idempotency-Key`.
- Kafka producer на стороне API.
- Kafka consumer с дедупликацией, retry, DLQ и replay.
- Метрики Prometheus и дашборды Grafana.
- Легкую нагрузку для демонстрации графиков и активности системы.

## Архитектура
Проект состоит из нескольких компонентов:

- `app` — основной HTTP-сервис на Go.
- `consumer` — отдельный воркер, читающий события из Kafka.
- `PostgreSQL` — основное хранилище данных.
- `Kafka` — брокер событий.
- `kafka-init` — одноразовый контейнер, создающий Kafka topics.
- `Prometheus` — сбор метрик.
- `Grafana` — визуализация метрик и lag.
- `kafka-exporter` — экспорт lag и offsets из Kafka.

Поток данных выглядит так:

1. Клиент создает транзакцию через `POST /items`.
2. API валидирует и записывает транзакцию в PostgreSQL.
3. После успешной записи API публикует событие `transaction.created` в Kafka.
4. Consumer читает событие, валидирует envelope, проверяет дедупликацию и фиксирует обработку.
5. При ошибках сообщение может попасть в DLQ.
6. Метрики producer, consumer и lag доступны в Prometheus и Grafana.

## Структура проекта

- `cmd/server` — точка входа HTTP API.
- `cmd/consumer` — Kafka consumer.
- `cmd/dlqreplay` — replay сообщений из DLQ обратно в основной topic.
- `cmd/load` — генератор легкой нагрузки на API.
- `internal/config` — загрузка конфигурации.
- `internal/db` — подключение к БД и миграции.
- `internal/events` — producer, DLQ, replay, контракт события.
- `internal/handlers` — HTTP-обработчики.
- `internal/service` — бизнес-логика.
- `internal/repository` — доступ к PostgreSQL.
- `internal/observability` — Prometheus-метрики.
- `migrations` — SQL-миграции и тестовые данные.
- `grafana` — provisioning и дашборды Grafana.
- `web` — статический клиент.

## Основные возможности

### HTTP API
Сервис поднимает следующие маршруты:

- `GET /` — статический Web UI.
- `GET /health` — liveness check.
- `GET /ready` — readiness check с проверкой подключения к БД.
- `GET /metrics` — метрики Prometheus для API.
- `POST /items` — создание транзакции.
- `GET /items` — список транзакций пользователя.
- `GET /items/{id}` — получение транзакции по id.
- `PUT /items/{id}` — обновление транзакции.
- `DELETE /items/{id}` — мягкое удаление транзакции.
- `GET /analytics` — аналитика за период.

### Идемпотентность `POST /items`
Если клиент передает заголовок `Idempotency-Key`, сервис:

- вычисляет хеш тела запроса;
- проверяет, был ли уже такой ключ для данного пользователя;
- при совпадении тела возвращает сохраненный ответ повторно;
- при конфликте тела возвращает `409 Conflict`.

Идемпотентность хранится в таблице `idempotency_keys`.

### Kafka
Kafka в проекте используется как событийная шина вокруг операции создания транзакции.

Что уже реализовано:

- API публикует событие `transaction.created`.
- Consumer читает события из topic `transactions.events`.
- Consumer валидирует JSON-envelope.
- Дедупликация идет по `event_id` через таблицу `processed_events`.
- Для ошибок обработки используется topic `transactions.events.dlq`.
- Есть утилита replay из DLQ в основной topic.
- Добавлены retry перед отправкой в DLQ.

## Контракт события Kafka
Событие `transaction.created` имеет JSON-envelope:

```json
{
  "event_id": "uuid",
  "event_type": "transaction.created",
  "event_time": "2026-03-21T16:00:00Z",
  "correlation_id": "uuid",
  "schema_version": 1,
  "source": "diploma-app",
  "transaction": {
    "id": "uuid",
    "user_id": "uuid",
    "amount": "1500.00",
    "currency": "RUB",
    "from_account_id": "uuid",
    "to_account_id": null,
    "provider_id": "uuid",
    "category_id": "uuid",
    "type": "expense",
    "status": "done",
    "description": "Lunch",
    "external_id": "ext-123",
    "occurred_at": "2026-03-21T16:00:00Z",
    "created_at": "2026-03-21T16:00:01Z",
    "updated_at": "2026-03-21T16:00:01Z",
    "deleted_at": null
  }
}
```

Consumer проверяет:

- `event_id` не пустой;
- `event_type == transaction.created`;
- `schema_version == 1`;
- `transaction` присутствует;
- `transaction.id == event_id`.

## Правила валидации транзакций

### Допустимые значения

- `type`: `income`, `expense`, `transfer`
- `status`: `pending`, `done`, `failed`

### Базовые правила

- Для `income` должен быть указан `to_account_id`.
- Для `expense` должен быть указан `from_account_id`.
- Для `transfer` должны быть указаны оба счета.
- Для `transfer` `from_account_id` и `to_account_id` должны отличаться.
- `amount` должен быть положительным числом.
- `currency` должна быть валидной валютой.
- `occurred_at` должен быть корректным RFC3339 timestamp.
- `category_id`, `provider_id`, `user_id`, `account_id` должны быть валидными UUID.

## Аналитика
Маршрут `GET /analytics` принимает query-параметры:

- `user_id`
- `from`
- `to`

В ответе возвращаются:

- `sum`
- `avg`
- `count`
- `median`
- `percentile_90`

Пример запроса:

```bash
curl "http://localhost:8081/analytics?user_id=11111111-1111-1111-1111-111111111111&from=2026-03-01T00:00:00Z&to=2026-03-31T23:59:59Z"
```

## Тестовые данные
В `migrations/002_init_data.sql` есть сиды для пользователей, счетов, категорий и провайдеров.

Самый удобный набор для ручных запросов:

- `user_id`: `11111111-1111-1111-1111-111111111111`
- `from_account_id`: `aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa`
- `category_id`: `44444444-4444-4444-4444-444444444444`
- `provider_id`: `77777777-7777-7777-7777-777777777777`

Именно эти значения использует утилита нагрузки `cmd/load`.

## Конфигурация
Конфигурация загружается из `environment/.env` и может быть переопределена через переменные окружения.

Ключевые параметры:

- `POSTGRES_HOST`
- `POSTGRES_PORT`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_NAME`
- `SERVER_PORT`
- `SERVER_READ_TIMEOUT`
- `SERVER_WRITE_TIMEOUT`
- `APP_ENV`
- `APP_LOG_LEVEL`
- `KAFKA_BROKERS`
- `KAFKA_TOPIC`
- `KAFKA_DLQ_TOPIC`
- `KAFKA_CONSUMER_GROUP_ID`
- `CONSUMER_METRICS_PORT`
- `APP_HTTP_PORT`
- `PROMETHEUS_PORT`
- `GRAFANA_PORT`
- `KAFKA_EXTERNAL_PORT`
- `KAFKA_EXPORTER_PORT`

## Миграции
Миграции лежат в `migrations/` и выполняются автоматически при старте API.

Текущие миграции:

- `000_extensions.sql` — включает `pgcrypto`.
- `001_init_schema.sql` — базовая схема.
- `002_init_data.sql` — сиды.
- `003_idempotency.sql` — таблица идемпотентности.
- `004_consumer_step4.sql` — таблица `processed_events`.

Важно:

- миграции выполняет только `app`;
- `consumer` миграции не запускает;
- в `RunMigrations` используется advisory lock, чтобы исключить гонку между процессами.

## Запуск

### Рекомендуемый запуск: полный стек через Docker Compose
Из корня проекта:

```bash
make docker-up
```

Эта команда поднимает:

- PostgreSQL
- Kafka
- kafka-init
- app
- consumer
- Prometheus
- Grafana
- kafka-exporter

После старта сервисы обычно доступны по следующим адресам:

- API / Web UI: `http://localhost:8081`
- Health: `http://localhost:8081/health`
- Ready: `http://localhost:8081/ready`
- API metrics: `http://localhost:8081/metrics`
- Consumer metrics: `http://localhost:2112/metrics`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`
- Kafka exporter: `http://localhost:9308/metrics`
- Kafka с хоста: `localhost:9094`
- PostgreSQL с хоста: `localhost:28436`

### Полный сброс и чистый старт
Если нужно начать с нуля и удалить volumes:

```bash
make docker-down-v
make docker-up
```

### Локальный запуск без Docker Compose
Можно запускать отдельные процессы локально:

```bash
make run
make run-consumer
```

Но это не основной сценарий. Для локального запуска зависимости должны быть доступны отдельно по значениям из `.env`.

Если уже поднят стек через `make docker-up`, отдельно запускать `make run-consumer` обычно не нужно: порт `2112` уже занят контейнером `consumer`.

## Команды Makefile

- `make docker-up` — поднять весь стек.
- `make docker-stop` — остановить контейнеры.
- `make docker-down-v` — остановить контейнеры и удалить volumes.
- `make run` — локальный запуск API.
- `make run-consumer` — локальный запуск consumer.
- `make dlq-replay ARGS='-limit 5'` — replay сообщений из DLQ.
- `make load` — легкая нагрузка на API.

Примеры:

```bash
make load
make load LOAD_DURATION=45s LOAD_QPS=8 LOAD_WORKERS=4
make dlq-replay ARGS='-dry-run -limit 5'
make dlq-replay ARGS='-limit 5'
```

## Примеры запросов к API

### Health

```bash
curl -s http://localhost:8081/health
curl -s http://localhost:8081/ready
```

### Создание транзакции

```bash
curl -X POST http://localhost:8081/items \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "11111111-1111-1111-1111-111111111111",
    "amount": "199.99",
    "currency": "RUB",
    "from_account_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
    "provider_id": "77777777-7777-7777-7777-777777777777",
    "category_id": "44444444-4444-4444-4444-444444444444",
    "type": "expense",
    "status": "done",
    "description": "Manual test",
    "external_id": "manual-test-1",
    "occurred_at": "2026-03-21T16:00:00Z"
  }'
```

### Идемпотентное создание

```bash
curl -X POST http://localhost:8081/items \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: my-test-key-1" \
  -d '{
    "user_id": "11111111-1111-1111-1111-111111111111",
    "amount": "350.00",
    "currency": "RUB",
    "from_account_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
    "provider_id": "77777777-7777-7777-7777-777777777777",
    "category_id": "44444444-4444-4444-4444-444444444444",
    "type": "expense",
    "status": "done",
    "description": "Idempotent request",
    "external_id": "manual-test-2",
    "occurred_at": "2026-03-21T16:10:00Z"
  }'
```

Повтор с тем же телом и тем же `Idempotency-Key` должен вернуть сохраненный результат, а не создать новую транзакцию.

### Список транзакций

```bash
curl "http://localhost:8081/items?user_id=11111111-1111-1111-1111-111111111111"
```

### Получить транзакцию по id

```bash
curl "http://localhost:8081/items/<tx_id>?user_id=11111111-1111-1111-1111-111111111111"
```

## Нагрузка и демонстрация метрик

Для демонстрации Prometheus / Grafana:

```bash
make load
```

Или с параметрами:

```bash
make load LOAD_DURATION=60s LOAD_QPS=10 LOAD_WORKERS=6
```

Что делает `make load`:

- проверяет `GET /health`;
- шлет валидные `POST /items` на API;
- использует сиды из `002_init_data.sql`;
- генерирует уникальные `external_id`;
- печатает итог `ok` / `errors`.

Если после `make load` сервис не упал, а в конце видно `errors=0`, значит базовый smoke/load-сценарий прошел успешно.

## Kafka и consumer: как это работает

### Producer
После успешного создания транзакции API публикует событие в Kafka.

Это происходит после записи в PostgreSQL, поэтому Kafka здесь не заменяет БД, а фиксирует факт совершившейся операции.

### Consumer
Consumer:

- читает `transactions.events`;
- валидирует envelope;
- сохраняет `event_id` в `processed_events`, чтобы исключить повторную обработку;
- выполняет обработку события;
- делает retry с backoff при ошибке handler;
- при окончательной неудаче пишет в DLQ;
- коммитит offsets вручную.

### DLQ
DLQ содержит сообщения, которые:

- не прошли валидацию;
- не удалось обработать;
- не удалось сохранить как обработанные.

Формат DLQ-сообщения:

- `failed_at`
- `original_topic`
- `partition`
- `offset`
- `event_id`
- `event_type`
- `reason`
- `payload`

### Replay
Утилита `cmd/dlqreplay` читает сообщения из DLQ и заново публикует исходный payload в основной topic.

Нюанс:

- если `event_id` уже есть в `processed_events`, consumer увидит повтор как дубликат;
- чтобы действительно заново прогнать бизнес-обработку для того же `event_id`, нужно удалить запись из `processed_events`.

## Метрики и наблюдаемость

### Метрики API
API публикует:

- HTTP request count
- HTTP latency
- in-flight requests
- count создания транзакций
- count запросов аналитики
- метрики Kafka producer

### Метрики consumer
Consumer публикует:

- успешно обработанные сообщения
- invalid сообщения
- duplicate сообщения
- ошибки commit
- публикации в DLQ
- ошибки публикации в DLQ
- retry handler
- длительность обработки
- лаг «время события в envelope (`event_time`) → успешная обработка» (`kafka_consumer_event_processing_lag_seconds`)

### Метрики Kafka lag
`kafka-exporter` добавляет lag consumer group и offsets.

### Prometheus
Сейчас Prometheus скрейпит:

- `app:8080/metrics`
- `consumer:2112/metrics`
- `kafka-exporter:9308`

### Grafana
В проекте уже настроены:

- datasource Prometheus
- дашборд HTTP / бизнес-метрик
- дашборд Kafka pipeline (в т.ч. p50/p95 лага `event_time` → успешная обработка в consumer)

## Как тестировать сервис

Минимальный сценарий:

1. Поднять стек:

```bash
make docker-up
```

2. Проверить health:

```bash
curl -s http://localhost:8081/health
curl -s http://localhost:8081/ready
```

3. Создать транзакцию через `POST /items`.

4. Убедиться, что:

- API вернул `201 Created`;
- в логах `diploma_consumer` появилась обработка `transaction.created`;
- в Grafana начали двигаться графики;
- в Prometheus targets находятся в `UP`.

5. Запустить нагрузку:

```bash
make load
```

6. Проверить:

- рост `http_requests_total`
- рост `transactions_created_total`
- рост `kafka_producer_messages_sent_total`
- рост `kafka_consumer_messages_processed_total`
- lag в Kafka dashboard

## Типичные проблемы и диагностика

### 1. `make run-consumer` падает с `address already in use`
Обычно это значит, что consumer уже поднят через Docker Compose и порт `2112` занят контейнером `diploma_consumer`.

Что делать:

- не запускать `make run-consumer`, если уже выполнен `make docker-up`;
- или остановить контейнер consumer;
- или сменить порт метрик для локального запуска.

### 2. Приложение падает на миграциях
Раньше `app` и `consumer` могли параллельно стартовать и конкурировать за миграции. Сейчас это исправлено:

- миграции выполняет только `app`;
- в `RunMigrations` есть advisory lock;
- consumer зависит от старта `app`.

Если после неудачного старта БД осталась в промежуточном состоянии, проще всего:

```bash
make docker-down-v
make docker-up
```

### 3. `ready` возвращает ошибку
Сервис поднят, но БД недоступна. Нужно проверить:

- контейнер PostgreSQL;
- переменные окружения для БД;
- readiness route `GET /ready`.

### 4. `make load` показывает ошибки
Нужно проверить:

- действительно ли API слушает `http://localhost:8081`;
- применены ли сиды из `002_init_data.sql`;
- не изменились ли тестовые UUID;
- свободен ли API от 4xx / 5xx в логах.

### 5. DLQ replay не дает повторной обработки
Это может быть нормой, если `event_id` уже есть в `processed_events`.

## Безопасность и ограничения текущей реализации

- Проект учебный, поэтому admin credentials Grafana заданы явно (`admin/admin`).
- Нет полноценной аутентификации пользователей.
- Consumer пока не выполняет сложную доменную обработку, а служит каркасом для событийного пайплайна.
- Replay не удаляет автоматически записи из `processed_events`.
- DLQ и retry реализованы в минимально полезном виде для учебного сценария.

## Что уже реализовано по шагам Kafka

1. Kafka-инфраструктура в Docker Compose.
2. Producer в API после создания транзакции.
3. Отдельный consumer-сервис.
4. Дедупликация, DLQ, ручной commit.
5. Метрики, Prometheus, Grafana, lag через kafka-exporter.
6. Retry и replay.
7. Основа для демонстрации и описания в дипломе.

## Что можно развивать дальше

- расширить payload и версионирование event contract;
- добавить более сложную бизнес-обработку в consumer;
- реализовать отдельный replay workflow или admin endpoint;
- добавить алерты в Prometheus / Grafana;
- добавить интеграционные тесты поверх Docker Compose;
- добавить OpenTelemetry и трассировку;
- вынести Kafka/consumer в отдельный сервисный модуль.

## Краткий сценарий для защиты

1. Показать `make docker-up`.
2. Открыть `http://localhost:8081/health` и `http://localhost:8081/ready`.
3. Создать транзакцию через UI или `curl`.
4. Показать, что транзакция записалась в БД и событие ушло в Kafka.
5. Показать логи consumer.
6. Запустить `make load`.
7. Открыть Grafana и показать метрики HTTP, producer, consumer и lag.
8. При необходимости рассказать про DLQ и replay.
