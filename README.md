# link-shortener

Сервис коротких ссылок на Go.

Стек: Go, gin, PostgreSQL, pgx, Docker.

## Возможности

- Короткая ссылка для адреса со схемой http:// или https://
- Ограничение числа переходов: после лимита ссылка не открывается
- Лимит соблюдается точно, даже когда по ссылке переходят одновременно

## Требования

- Docker 27.4

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
| `APP_PORT` | Порт сервиса | Обязательна |
| `POSTGRES_USER` | Пользователь БД | Обязательна |
| `POSTGRES_PASSWORD` | Пароль БД | Обязательна |
| `POSTGRES_DB` | Имя БД | Обязательна |
| `POSTGRES_SSLMODE` | Режим SSL | В Docker Compose `disable` |
| `POSTGRES_HOST` | Хост БД | `postgres` |
| `POSTGRES_PORT` | Порт БД | `5432` |
| `GIN_MODE` | Режим gin | В Docker Compose `release` |

## Структура

```
cmd/app            Точка входа
internal/config    Настройки из окружения
internal/handler   HTTP-обработчики
internal/link      Логика ссылок и работа с БД
migrations         SQL-миграции
```

## Команды

| Команда | Действие |
|---|---|
| `make up` | Собрать и запустить сервис |
| `make down` | Остановить сервис |
| `make logs` | Смотреть логи приложения |
| `make fmt` | Отформатировать код |
| `make lint` | Запустить линтеры |
| `make test` | Юнит-тесты |
| `make test-integration` | Тесты с настоящим Postgres в Docker |

Для `fmt` и `lint` нужен установленный golangci-lint версии 2.

## Лицензия

MIT
