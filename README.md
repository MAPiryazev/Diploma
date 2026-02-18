# Сервис потоковой обработки финансовых транзакций

## Описание
Сервис предоставляет HTTP API для работы с транзакциями пользователей и получения аналитики за определенный период.
В качестве хранилища используется PostgreSQL, конфигурация читается из `environment/.env`.

## Установка и запуск
Поднимите PostgreSQL через Docker Compose из корня проекта.
Контейнер Postgres публикует порт `28436` на хост-машину (внутри контейнера 5432).

```bash
docker compose up -d
```

Проверьте файл `environment/.env`.
Для локального запуска по умолчанию используются `POSTGRES_PORT=28436` и `SERVER_PORT=8080`.

Запустите сервер:

```bash
go mod tidy
go run ./cmd/server/main.go
```

Откройте в браузере:  
Web UI: http://localhost:8080/  
Healthcheck: http://localhost:8080/health  

## Правила валидации транзакций

### Допустимые значения
- type: `income`, `expense`, `transfer`  
- status: `pending`, `done`, `failed`   

### Правила валидации
- Для `income` требуется `to_account_id`. 
- Для `expense` требуется `from_account_id`.  
- Для `transfer` требуются оба поля, при этом `from_account_id` и `to_account_id` должны отличаться
