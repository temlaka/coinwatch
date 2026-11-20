# coinwatch

[![CI](https://github.com/temlaka/coinwatch/actions/workflows/ci.yml/badge.svg)](https://github.com/temlaka/coinwatch/actions/workflows/ci.yml)

REST API для отслеживания цен криптовалют. Добавьте монету по тикеру, и coinwatch
будет хранить историю её цены из CoinGecko, обновлять её в фоне по настраиваемому
расписанию и считать статистику.

## Возможности

- Аутентификация по JWT, пароли хешируются bcrypt
- Отслеживание монет по тикеру (`BTC` → `bitcoin` через CoinGecko)
- История цен (последние 100 точек) и статистика: min / max / avg / изменение в %
- Фоновое обновление цен с настраиваемым интервалом и ручным запуском
- PostgreSQL, миграции применяются при старте
- Graceful shutdown, Docker-образ, CI
- Интерактивная документация API (Swagger UI)

## Стек

Go 1.27 · `net/http` · PostgreSQL 16 · pgx v5 · goose · JWT · Docker Compose ·
GitHub Actions · golangci-lint · OpenAPI 3

## Быстрый старт

Понадобятся Docker и бесплатный [Demo API-ключ CoinGecko](https://www.coingecko.com/en/api).

```bash
git clone https://github.com/temlaka/coinwatch.git
cd coinwatch
cp .env.example .env
```

Замените заглушки в `.env`:

- `COINGECKO_API_KEY` — ваш Demo-ключ CoinGecko
- `JWT_SECRET` — любая длинная случайная строка, например `openssl rand -hex 32`

Запустите приложение и базу:

```bash
docker compose up -d --build
```

Документация API: **http://localhost:8080/docs**

### Запуск без Docker

```bash
docker compose up -d db
go run .
```

Приложение читает `.env` и подключается к базе на `localhost:5434`.
Внутри Compose оно использует `db:5432`.

## API

| Метод | Путь | Описание |
|---|---|---|
| POST | `/auth/register` | Регистрация, возвращает токен |
| POST | `/auth/login` | Вход, возвращает токен |
| GET | `/crypto` | Список отслеживаемых монет |
| POST | `/crypto` | Начать отслеживать монету `{"symbol": "BTC"}` |
| GET | `/crypto/{symbol}` | Информация о монете |
| PUT | `/crypto/{symbol}/refresh` | Обновить цену сейчас |
| GET | `/crypto/{symbol}/history` | История цен |
| GET | `/crypto/{symbol}/stats` | Статистика цен |
| DELETE | `/crypto/{symbol}` | Перестать отслеживать монету |
| GET | `/schedule` | Настройки автообновления |
| PUT | `/schedule` | Изменить настройки `{"enabled": true, "interval_seconds": 60}` |
| POST | `/schedule/trigger` | Обновить все цены сейчас |

Все эндпоинты, кроме `/auth/*`, требуют заголовок `Authorization: Bearer <token>`.
Полная спецификация: [`docs/openapi.yaml`](docs/openapi.yaml).

```bash
TOKEN=$(curl -s -X POST localhost:8080/auth/register \
  -d '{"username":"alice","password":"secret"}' | jq -r .token)

curl -X POST localhost:8080/crypto -H "Authorization: Bearer $TOKEN" -d '{"symbol":"BTC"}'
curl localhost:8080/crypto/BTC/stats -H "Authorization: Bearer $TOKEN"
```

## Структура проекта

```
main.go                  сборка приложения: конфиг, БД, миграции, HTTP-сервер, shutdown
internal/
  domain/                сущности и доменные ошибки
  usecase/               бизнес-логика: auth, crypto, schedule
  repository/postgres/   хранилище в PostgreSQL (pgx)
  repository/memory/     хранилище в памяти, используется в тестах
  handler/               HTTP-хендлеры, роутинг, JWT middleware
  client/coingecko/      клиент CoinGecko API
  scheduler/             фоновое обновление цен
migrations/              SQL-миграции, встроены в бинарник
docs/                    спецификация OpenAPI
```

Слой usecase зависит только от интерфейсов, поэтому хранилище и источник цен
можно заменить. Unit-тесты работают на in-memory репозиториях и фейковом
источнике цен, без базы и сети.

## Разработка

```bash
go test -race ./...
golangci-lint run ./...
```
