# API и интерактивная консоль

Сервер поднимает два транспорта над одной бизнес-логикой:

| Интерфейс | Адрес по умолчанию | Назначение |
|---|---|---|
| gRPC `chat.v1` | `localhost:50051` | Основной программный API |
| HTTP JSON/SSE | `http://localhost:8080` | Браузерная консоль и ручное тестирование |

Интерактивная документация доступна на <http://localhost:8080/docs>.
OpenAPI 3.1: <http://localhost:8080/openapi.json>.

## Запуск

```bash
cp .env.example .env
# Задайте POSTGRES_PASSWORD, DATABASE_URL, JWT_SECRET и MESSAGE_ENCRYPTION_KEY (openssl rand -base64 32).
docker compose --env-file .env up --build
```

Контейнер `migrate` дождётся PostgreSQL, применит SQL-миграции и только после
этого запустит сервер. Проверка процесса: `GET http://localhost:8080/healthz`.

## Авторизация

Публичны только `Register` и `Login`. Для остальных вызовов передавайте JWT:

```text
authorization: Bearer <access_token>
```

В браузерной консоли токен после Register/Login сохраняется в `sessionStorage`
текущей вкладки и автоматически добавляется к запросам. В URL токен никогда не
передаётся. Время жизни задаётся `JWT_TTL`.

### Register

- gRPC: `chat.v1.AuthService/Register`
- HTTP: `POST /api/v1/auth/register`
- username: `^[A-Za-z0-9_]{3,32}$`, уникален без учёта регистра;
- password: от 8 до 72 кодовых точек Unicode при регистрации. Новые пароли хешируются с помощью Argon2id.

```bash
curl -s http://localhost:8080/api/v1/auth/register \
  -H 'content-type: application/json' \
  -d '{"username":"alice","password":"password123"}'
```

```bash
grpcurl -plaintext -import-path proto -proto chat/v1/auth.proto \
  -d '{"username":"alice","password":"password123"}' \
  localhost:50051 chat.v1.AuthService/Register
```

### Login

- gRPC: `chat.v1.AuthService/Login`
- HTTP: `POST /api/v1/auth/login`

```bash
curl -s http://localhost:8080/api/v1/auth/login \
  -H 'content-type: application/json' \
  -d '{"username":"alice","password":"password123"}'
```

Неизвестное имя пользователя и неверный пароль возвращают одинаковую ошибку.
Старые аккаунты по-прежнему могут войти с паролями, зарегистрированными по правилам bcrypt.

## Пользователи

Все методы требуют JWT.

### GetMe

- gRPC: `chat.v1.UserService/GetMe`
- HTTP: `GET /api/v1/users/me`

```bash
curl -s http://localhost:8080/api/v1/users/me \
  -H "authorization: Bearer $TOKEN"

grpcurl -plaintext -import-path proto -proto chat/v1/user.proto \
  -H "authorization: Bearer $TOKEN" -d '{}' \
  localhost:50051 chat.v1.UserService/GetMe
```

Идентификатор текущего пользователя берётся только из проверенного JWT.

### GetUser

- gRPC: `chat.v1.UserService/GetUser`
- HTTP: `GET /api/v1/users/{user_id}`

```bash
curl -s "http://localhost:8080/api/v1/users/$USER_ID" \
  -H "authorization: Bearer $TOKEN"
```

### FindUser

- gRPC: `chat.v1.UserService/FindUser`
- HTTP: `GET /api/v1/users/find?username=alice`

Поиск точный и не зависит от регистра. Старый gRPC-метод `FindByUsername`
сохранён как совместимый alias.

## Статусы

### SetStatus

- gRPC: `chat.v1.PresenceService/SetStatus`
- HTTP: `PUT /api/v1/presence/status`
- допустимые значения: `ONLINE`, `IDLE`, `DND`.

```bash
curl -s -X PUT http://localhost:8080/api/v1/presence/status \
  -H "authorization: Bearer $TOKEN" \
  -H 'content-type: application/json' \
  -d '{"status":"DND"}'
```

`OFFLINE` нельзя установить вручную. Это вычисляемое состояние: пользователь
считается offline, когда у него нет активных `SubscribeEvents`. Несколько вкладок
или клиентов учитываются отдельно; закрытие одного из нескольких stream не
переводит пользователя в offline. При закрытии последнего обновляется
`last_seen_at`.

## События

Основной gRPC stream:

```bash
grpcurl -plaintext -import-path proto -proto chat/v1/event.proto \
  -H "authorization: Bearer $TOKEN" -d '{}' \
  localhost:50051 chat.v1.EventService/SubscribeEvents
```

Для браузера тот же Event Hub доступен как Server-Sent Events:

```bash
curl -N http://localhost:8080/api/v1/events \
  -H "authorization: Bearer $TOKEN" \
  -H 'accept: text/event-stream'
```

Пример SSE:

```text
event: presence.changed
data: {"event_id":"...","type":"presence.changed","payload":{"user_id":"...","status":"ONLINE"}}
```

Используется `fetch()` streaming, а не нативный `EventSource`, потому что
`EventSource` не позволяет установить Bearer header. Очереди на подключение
ограничены; медленный consumer отключается, чтобы не задерживать остальных.

## Диалоги и сообщения

Фронтенд на Next.js передаёт запросы этому API. Браузер получает HttpOnly-cookie
сессии, а Next.js подставляет Bearer-токен при обращении к Go-серверу.
При прямых вызовах Go API по-прежнему передавайте Bearer-токен.

| HTTP | Тело запроса | Ответ |
|---|---|---|
| `GET /api/v1/dialogs` | — | `{dialogs, page_info}` |
| `POST /api/v1/dialogs` | `{other_user_id}` | Новый или существующий личный диалог |
| `GET /api/v1/dialogs/{dialog_id}/messages` | — | `{messages, page_info}` |
| `POST /api/v1/dialogs/{dialog_id}/messages` | `{body, reply_to_message_id?}` | Отправленное сообщение |
| `POST /api/v1/dialogs/{dialog_id}/typing` | `{}` | `204 No Content` |
| `POST /api/v1/dialogs/{dialog_id}/read` | `{up_to_message_id}` | `{marked_count, read_at}` |
| `PUT /api/v1/messages/{message_id}/reaction` | `{reaction}` | Реакция пользователя |
| `DELETE /api/v1/messages/{message_id}/reaction` | — | `{removed}` |

Списки возвращаются от новых записей к старым. Для следующей страницы передайте
`page_info.next_page_token` как `page_token`. Размер страницы можно задать
параметром `page_size`. Текст сообщения ограничен 4000 символами.
Допустимые реакции: `LIKE`, `LOVE`, `LAUGH`, `WOW`.

Чтобы ответить на сообщение, добавьте `reply_to_message_id` к JSON-запросу
`POST /api/v1/dialogs/{dialog_id}/messages`. UUID должен принадлежать сообщению
в том же диалоге. При успешной отправке и чтении истории у ответа есть
поле `reply_to` с `id`, `sender_id` и текстом превью в `excerpt`. Событие SSE
`message.created` для ответа также содержит `reply_to_message_id`.

POST на маршрут `typing` отправляет только собеседнику событие SSE
`dialog.typing` с `dialog_id` и `user_id`. Веб-клиент отправляет обновления
максимум раз в две секунды. Статус исчезает через пять секунд после последнего
события или при получении сообщения от собеседника.

Участники диалога получают события `message.created`, `messages.read` и
`reaction.changed`. После переподключения клиент запрашивает актуальные данные.

## HTTP-коды ошибок

| HTTP | gRPC | Значение |
|---|---|---|
| 400 | `InvalidArgument` | Некорректный JSON, UUID, username, пароль или статус |
| 401 | `Unauthenticated` | Отсутствующий, неверный или просроченный JWT |
| 403 | `PermissionDenied` | Операция запрещена |
| 404 | `NotFound` | Запись не найдена |
| 409 | `AlreadyExists` | Username уже занят |
| 500 | `Internal` | Внутренняя ошибка без деталей БД и секретов |

HTTP-ответ:

```json
{"error":"user not found"}
```

## Переменные окружения

| Переменная | Пример | Описание |
|---|---|---|
| `GRPC_ADDRESS` | `:50051` | Адрес gRPC listener |
| `HTTP_ADDRESS` | `:8080` | Адрес UI и JSON/SSE facade |
| `DATABASE_URL` | `postgres://...` | DSN PostgreSQL |
| `JWT_ISSUER` | `grpc-chat` | Проверяемый issuer JWT |
| `JWT_TTL` | `24h` | Положительное время жизни токена |
| `MESSAGE_ENCRYPTION_KEY` | `openssl rand -base64 32` | Ключ шифрования истории сообщений |
| `JWT_SECRET` | 32+ random bytes | HS256 secret; не коммитить в репозиторий |

## Архитектурные ограничения

Presence Manager и Event Hub находятся в памяти одного процесса. Это корректно
для текущего модульного монолита и локального запуска. Для нескольких экземпляров
сервера потребуются общий presence registry и pub/sub, например Redis или NATS.

HTTP facade обслуживает браузерный клиент и консоль. Контракты gRPC находятся
в `proto/chat/v1`. Схема базы изменяется SQL-миграциями из `migrations`.
