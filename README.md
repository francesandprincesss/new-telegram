# gRPC Chat

Учебный личный мессенджер на Go: модульный монолит с gRPC, PostgreSQL,
GORM, JWT и server streaming.

## Запуск

```sh
cp .env.example .env
# Замените JWT_SECRET и пароль PostgreSQL, затем:
docker compose --env-file .env up --build
```

Миграции применяются отдельным контейнером до старта gRPC-сервера на `:50051`.
`Register` и `Login` публичны; остальные RPC требуют metadata
`authorization: Bearer <JWT>`.

`GetUser` принимает UUID, `FindUser` выполняет точный поиск username без учёта
регистра. `FindByUsername` оставлен как совместимый alias. Статусы `ONLINE`,
`IDLE`, `DND` сохраняются как пользовательская настройка, а `OFFLINE`
вычисляется автоматически при отсутствии активного `SubscribeEvents`.
