# gRPC Chat

Учебный личный мессенджер на Go: модульный монолит с gRPC, PostgreSQL,
GORM, JWT и server streaming.

## Запуск и Web API Console

```sh
cp .env.example .env
# Замените JWT_SECRET и пароль PostgreSQL, затем:
docker compose --env-file .env up --build
```

После запуска откройте **<http://localhost:8080/docs>**. Это встроенная
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
