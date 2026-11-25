# gRPC Chat

Учебный личный мессенджер на Go: модульный монолит с gRPC, PostgreSQL,
GORM, JWT и server streaming.

## Запуск мессенджера

```sh
cp .env.example .env
# Замените JWT_SECRET и пароль PostgreSQL, затем:
docker compose --env-file .env up --build
```

После запуска откройте **<http://localhost:3000>**. Зарегистрируйтесь, нажмите
«Новый чат» и найдите собеседника по имени пользователя. Для проверки переписки
между двумя аккаунтами используйте два профиля браузера или обычное и приватное
окна.

При разработке фронтенда запустите Go-сервер с зависимостями через
`docker compose --env-file .env up --build server`, затем выполните
`npm ci` и `npm run dev` в каталоге `frontend`. Требуется Node.js 22.18 или новее.
`CHAT_API_ORIGIN` в `frontend/.env.local` задаёт адрес серверного API.
При работе через обратный прокси, который меняет хост или протокол, задайте
публичный адрес приложения через `CHAT_APP_ORIGIN`.

Проверки фронтенда в `frontend`: `npm run lint`, `npm run typecheck`,
`npm test` и `npm run build`.
`npm run test:integration` проверяет HTTP и SSE через запущенный Next.js.
Он создаёт три тестовых аккаунта, поэтому запускайте его с отдельной тестовой
базой. Другой адрес фронтенда можно задать через `CHAT_TEST_ORIGIN`.

## Web API Console

Консоль доступна на **<http://localhost:8080/docs>**. Это встроенная
интерактивная документация: в ней можно зарегистрироваться, войти, автоматически
подставить JWT, вызвать профильные RPC, изменить статус и смотреть Event stream.

Миграции применяются отдельным контейнером до старта gRPC-сервера на `:50051`;
web UI и JSON/SSE facade слушают `:8080`.
`Register` и `Login` публичны; остальные RPC требуют metadata
`authorization: Bearer <JWT>`.

`GetUser` принимает UUID, `FindUser` выполняет точный поиск username без учёта
регистра. `FindByUsername` оставлен как совместимый alias. Статусы `ONLINE`,
`IDLE`, `DND` сохраняются как пользовательская настройка, а `OFFLINE`
вычисляется автоматически при отсутствии активного `SubscribeEvents`.

Полная документация по gRPC, HTTP-маршрутам, curl/grpcurl, JWT, статусам,
событиям и переменным окружения: [docs/API.md](docs/API.md).
