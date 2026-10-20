# link-shortener

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)

Сервис коротких ссылок на Go.

Стек: Go, gin, PostgreSQL, pgx, Docker.

## Возможности

- [x] Короткая ссылка для адреса со схемой http:// или https://
- [x] Ограничение числа переходов: после лимита ссылка не открывается
- [x] Лимит соблюдается точно, даже когда по ссылке переходят одновременно

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

Описание в формате OpenAPI: [api/openapi.yaml](api/openapi.yaml).

### POST /links

> Создать короткую ссылку

#### Request

```bash
curl -X POST localhost:8080/links \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com/page", "max_clicks": 5}'
```

Поле `max_clicks` необязательное.

#### Response 201

```json
{"code": "5aVcEpZ"}
```

#### Response 400

```json
{"error": "url must be a valid http or https address"}
```

### GET /{code}

> Перейти по короткой ссылке

#### Request

```bash
curl -i localhost:8080/5aVcEpZ
```

#### Response 302

```
HTTP/1.1 302 Found
Location: https://example.com/page
```

#### Response 404

Ссылки нет или лимит переходов исчерпан.

```json
{"error": "link not found"}
```

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
api/openapi.yaml             Описание API
cmd/app                      Точка входа
internal/config              Настройки из окружения
internal/handler             HTTP-обработчики
internal/link                Логика ссылок
internal/repository/postgres Работа с PostgreSQL
migrations                   SQL-миграции
```

## Команды

| Команда | Действие |
|---|---|
| `make help` | Показать список команд |
| `make up` | Собрать и запустить сервис в Docker |
| `make down` | Остановить сервис |
| `make logs` | Смотреть логи приложения |
| `make build` | Собрать бинарник в `bin/app` |
| `make run` | Запустить приложение локально, база в Docker |
| `make fmt` | Отформатировать код |
| `make lint` | Запустить линтеры |
| `make test` | Юнит-тесты |
| `make test-integration` | Тесты с настоящим Postgres в Docker |

Для `fmt` и `lint` нужен установленный golangci-lint версии 2.

## Лицензия

MIT
