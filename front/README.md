# Front

Статический пользовательский frontend без сборщика и внешних зависимостей.

Запуск:

```bash
cd front
python3 server.py
```

После этого открыть:

```text
http://localhost:3000
```

По умолчанию фронт ходит в backend API:

```text
http://localhost:8080
```

`server.py` нужен не только для статики, но и для proxy загрузки картинок в MinIO через presigned URL.
