# Авторизация mini-market

`auth-app` регистрирует пользователей, проверяет пароль и выдаёт две учётные
сущности:

- короткоживущий JWT access token для доступа к защищённым API;
- случайный refresh token в `HttpOnly` cookie для продления сессии.

Вход выполняется по `username`. Email также обязателен и уникален, но для входа
не используется.

## Структура и запуск

Go-модуль и все серверные компоненты находятся в `backend/`. У каждого
микросервиса будет свой Dockerfile; сейчас реализован
`backend/auth-app/Dockerfile`. Файл `backend/docker-compose.yml` запускает весь
backend, а корневой `docker-compose.yml` является общей точкой orchestration для
backend и будущего frontend. Пустая папка `frontend/` пока не создаёт контейнер.

Нужны Docker с Compose и свободные порты `5433` и `8080`. PostgreSQL внутри
контейнера слушает стандартный `5432`, а наружу опубликован `5433`. Порты можно
заменить через `POSTGRES_PORT` и `AUTH_PORT`.

Первый запуск всего проекта из корня репозитория:

```bash
cp backend/.env.example backend/.env
# Замените JWT_SECRET на случайную строку длиной не менее 32 символов.
make up
make ps
```

Первый запуск только backend:

```bash
cd backend
cp .env.example .env
make docker-up
```

В обоих вариантах Compose ждёт готовности PostgreSQL, отдельный контейнер
`migrate` применяет goose-миграции и только после его успешного завершения
запускается `auth-app`. Остановить контейнеры можно через `make down` в корне
или `make docker-down` в `backend/`.

Для запуска Go-процесса без контейнера:

```bash
cd backend
set -a; source .env; set +a
make db-up
make migrate-up
make run
```

Локальные миграции откатываются командой `make migrate-down`. `goose`
запускается через `go run`, поэтому глобальная установка не нужна.

## Переменные окружения

- `DATABASE_URL` — PostgreSQL DSN, обязательна.
- `JWT_SECRET` — ключ подписи HS256, минимум 32 символа, обязателен.
- `JWT_ISSUER` — issuer JWT, по умолчанию `mini-market-auth`.
- `ACCESS_TOKEN_TTL` — срок access token, по умолчанию `15m`.
- `REFRESH_TOKEN_TTL` — срок refresh-сессии, по умолчанию `720h`.
- `HTTP_PORT` — порт сервиса, по умолчанию `8080`.
- `AUTH_PORT` — host-порт auth-контейнера, по умолчанию `8080`.
- `POSTGRES_PORT` — host-порт PostgreSQL, по умолчанию `5433`.
- `COOKIE_SECURE` — отправлять refresh cookie только через HTTPS. В production
  должно быть `true`.
- `COOKIE_SAME_SITE` — `lax`, `strict` или `none`. Значение `none` разрешено
  только вместе с `COOKIE_SECURE=true`.

## Схема данных

`users` хранит UUID пользователя, username, email, bcrypt hash пароля, имя и
временные метки. Индексы обеспечивают регистронезависимую уникальность username
и email.

`refresh_sessions` хранит отдельную сессию для каждого входа:

- UUID сессии и пользователя;
- SHA-256 hash refresh token — исходный токен в БД не сохраняется;
- время истечения;
- `revoked_at` для отозванных или уже заменённых токенов.

Удаление пользователя каскадно удаляет его refresh-сессии.

## Как работает auth-flow

### Регистрация

`POST /auth/register` валидирует данные, создаёт bcrypt hash пароля и
пользователя. В ответе возвращаются публичные данные пользователя и JWT access
token. Refresh token устанавливается cookie `refresh_token` с атрибутами
`HttpOnly`, `SameSite`, `Path=/auth` и настроенным `Secure`.

Username содержит 3–32 латинские буквы, цифры или `_`. Пароль содержит 8–72
байта (72 — ограничение bcrypt).

### Логин

`POST /auth/login` находит пользователя без учёта регистра username и сравнивает
пароль с bcrypt hash. Ошибка одинакова для неизвестного username и неверного
пароля, чтобы не раскрывать наличие аккаунта. Каждый успешный вход создаёт
отдельную refresh-сессию.

### Доступ к защищённому API

Клиент передаёт access token:

```text
Authorization: Bearer <access_token>
```

Middleware проверяет подпись HS256, issuer и срок действия, затем помещает UUID
и username в контекст запроса. `GET /auth/me` — пример защищённого endpoint.

### Refresh и ротация

`POST /auth/refresh` читает refresh token только из `HttpOnly` cookie. В одной
транзакции PostgreSQL старая сессия блокируется и отзывается, а новая
создаётся. Клиент получает новый access token и новую cookie.

Повторное использование старого, истёкшего или отозванного refresh token
возвращает `401`. Одновременные refresh-запросы с одной cookie сериализуются
блокировкой строки: успешным будет только один.

### Logout

`POST /auth/logout` отзывает текущую refresh-сессию и удаляет cookie. Уже
выданный access token остаётся действительным до своего короткого срока
истечения; сервер не хранит blacklist access-токенов.

## API и примеры

Во всех примерах `cookies.txt` хранит refresh cookie, а `ACCESS_TOKEN` нужно
взять из JSON ответа регистрации, логина или refresh.

В Postman используйте base URL `http://localhost:8080`. Postman сохраняет
`HttpOnly` refresh cookie автоматически. Для `/auth/me` выберите Authorization
type `Bearer Token` и вставьте `access_token` из JSON. Для `/auth/refresh` и
`/auth/logout` тело запроса не требуется.

```bash
curl -i -c cookies.txt http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "username":"stepan_1",
    "email":"stepan@example.com",
    "password":"correct-horse-battery-staple",
    "first_name":"Stepan",
    "last_name":"Ivanov"
  }'
```

```bash
curl -i -c cookies.txt http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"stepan_1","password":"correct-horse-battery-staple"}'
```

```bash
curl -i http://localhost:8080/auth/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"

curl -i -b cookies.txt -c cookies.txt -X POST \
  http://localhost:8080/auth/refresh

curl -i -b cookies.txt -c cookies.txt -X POST \
  http://localhost:8080/auth/logout
```

Проверка состояния сервиса и БД:

```bash
curl -i http://localhost:8080/health
```

Основные коды ответа: `201` для регистрации, `200` для login/refresh/me, `204`
для logout, `400` для невалидного JSON или полей, `401` для credentials/token,
`409` для занятого username/email.

## Production

- Используйте длинный случайный `JWT_SECRET` из secret manager и план ротации
  ключей; текущая версия использует один симметричный HS256-ключ.
- Включите HTTPS и `COOKIE_SECURE=true`.
- Ограничьте CORS доверенными origin и добавьте rate limiting для register,
  login и refresh на уровне gateway.
- Периодически удаляйте истёкшие и давно отозванные строки
  `refresh_sessions`.
- Не записывайте пароли, JWT и cookie в логи.
