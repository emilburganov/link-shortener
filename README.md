# link-shortener

Сервис сокращения ссылок на Go, gin и PostgreSQL.

## Возможности

- короткий код из 7 символов base62
- ограничение числа переходов по ссылке
- лимит переходов проверяется атомарно, гонки между запросами не превышают его

## Требования

- Docker с Compose v2
- свободный порт 8080

## Запуск

```bash
cp .env.example .env
```

Задайте `POSTGRES_PASSWORD` в `.env`, затем:

```bash
docker compose up -d --build --wait
```

Остановка:

```bash
docker compose down
```

Флаг `-v` дополнительно удалит данные базы.

## API

### POST /links

```bash
curl -X POST localhost:8080/links \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com/page", "max_clicks": 5}'
```

Поле `max_clicks` необязательное.

Ответ `201`:

```json
{"code": "5aVcEpZ"}
```

Ответ `400` при неверном теле или адресе:

```json
{"error": "url must be a valid http or https address"}
```

### GET /{code}

Редирект `302` на исходный адрес. Ответ `404`, если ссылки нет или лимит переходов исчерпан.

## Конфигурация

| Переменная | Назначение | По умолчанию |
|---|---|---|
| `APP_PORT` | порт сервиса | обязательна |
| `POSTGRES_USER` | пользователь БД | обязательна |
| `POSTGRES_PASSWORD` | пароль БД | обязательна |
| `POSTGRES_DB` | имя БД | обязательна |
| `POSTGRES_SSLMODE` | режим SSL | `disable` в Docker Compose |
| `POSTGRES_HOST` | хост БД | `postgres` |
| `POSTGRES_PORT` | порт БД | `5432` |
| `GIN_MODE` | режим gin | `release` в Docker Compose |

## Структура

```
cmd/app            точка входа
internal/config    настройки из окружения
internal/handler   HTTP-обработчики
internal/link      логика ссылок и работа с БД
migrations         SQL-миграции
```

## Лицензия

MIT
