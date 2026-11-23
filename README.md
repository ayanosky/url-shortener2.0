# URL Shortener

Сервис для создания коротких ссылок на Go: принимает длинный URL, возвращает короткий код, делает редирект и считает переходы.

## Стек

- Go 1.27.1
- PostgreSQL 18
- Docker и Docker Compose

## Запуск

### Через Docker Compose

```bash
docker compose up -d --build
```

Поднимутся приложение (порт `8080`) и PostgreSQL (порт `5434` на хосте). Таблица создаётся автоматически из `migrations/` при первом старте базы.

Остановить: `docker compose down` (добавьте `-v`, чтобы удалить и данные).

### Локально

Сначала запустите только базу:

```bash
docker compose up -d db
```

Затем задайте переменные окружения (образец в `.env.example`) и запустите сервер.

Bash:

```bash
export DATABASE_URL="postgres://shortener:shortener@localhost:5434/shortener?sslmode=disable"
go run ./cmd/server
```

PowerShell:

```powershell
$env:DATABASE_URL = "postgres://shortener:shortener@localhost:5434/shortener?sslmode=disable"
go run ./cmd/server
```

Приложение не читает файл `.env` само, переменные нужно задавать в окружении.

## Конфигурация

| Переменная     | Обязательная | По умолчанию | Описание                         |
|----------------|--------------|--------------|----------------------------------|
| `DATABASE_URL` | да           | —            | Строка подключения к PostgreSQL  |
| `PORT`         | нет          | `8080`       | Порт HTTP-сервера                |
| `LOG_LEVEL`    | нет          | `info`       | `debug`, `info`, `warn`, `error` |

## API

### `POST /shorten` — создать короткую ссылку

```bash
curl -i -X POST localhost:8080/shorten -d '{"url": "https://jut.su"}'
```

```
HTTP/1.1 201 Created
Content-Type: application/json

{"code":"idJb95"}
```

Ошибки: `400`, если JSON некорректен или URL невалидный (допустимы только `http` и `https`).

### `GET /{code}` — редирект

```bash
curl -i localhost:8080/idJb95
```

```
HTTP/1.1 302 Found
Location: https://jut.su
```

Ошибки: `404`, если код не найден.

### `GET /stats/{code}` — статистика

```bash
curl -i localhost:8080/stats/idJb95
```

```
HTTP/1.1 200 OK
Content-Type: application/json

{"code":"idJb95","url":"https://jut.su","clicks":1}
```

Ошибки: `404`, если код не найден.

### `GET /health` — проверка работоспособности

Возвращает `200 OK`.

## Структура проекта

```
.
├── cmd/server/          # точка входа, сборка зависимостей, middleware логирования
├── internal/
│   ├── config/          # чтение настроек из переменных окружения
│   ├── handler/         # HTTP-слой: разбор запросов, статус-коды
│   ├── service/         # бизнес-логика: валидация, генерация кодов, повтор при коллизиях
│   └── storage/         # интерфейс хранилища, реализации Postgres и Memory
├── migrations/          # SQL-миграции
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

Зависимости идут в одну сторону: `handler → service → storage`. Хэндлеры ничего не знают о базе, а сервис работает через интерфейс `storage.Storage`, поэтому в тестах вместо PostgreSQL подставляется `Memory`.

## Тесты

```bash
go test ./...
go test -race ./...   # с детектором гонок
go test -cover ./...  # с покрытием
```

Тесты не требуют запущенной базы.

## Заметки

- Пароль в `.env.example` и `docker-compose.yml` предназначен только для локальной разработки. Для реального окружения задайте свой.
- Коды состоят из 6 символов (`a-z`, `A-Z`, `0-9`), при коллизии сервис пробует до 5 раз.
- Сервер корректно завершается по `SIGINT` и `SIGTERM`: дожидается текущих запросов до 10 секунд.
