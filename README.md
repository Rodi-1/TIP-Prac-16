# ПЗ №16 — Интеграционные тесты API

Сортов Семён Александрович, ЭФМО-01-26. Дата: 25.10.2026.

Интеграционно проверяется цепочка Gin → service → repository → PostgreSQL. Выбран вариант A из задания: Docker Compose. Схема создаётся миграцией internal/db/schema.sql: таблица notes, CHECK заголовка, индекс title и триггер updated_at. Требуется Go 1.25+ и Docker Desktop; проверено с Go 1.27.2, Gin 1.11.0, PostgreSQL 16.15 и pgx/v5 5.11.0.

## БД и тесты

```powershell
docker compose up -d --wait
docker compose ps
$env:DB_DSN='postgres://test:test@localhost:54321/notes_test?sslmode=disable'
go test -v -shuffle=on -count=2 ./integration/...
```

Эквивалент: tasks.ps1 up, затем tasks.ps1 test. Без DB_DSN интеграционные тесты явно пропускаются. Для действительной проверки задайте переменную, как выше. В сохранённом прогоне пропусков нет.

Каждый верхнеуровневый тест создаёт случайную отдельную схему pz16_..., задаёт search_path и поднимает httptest.Server на свободном порту. t.Cleanup закрывает HTTP/SQL и удаляет только эту схему. Благодаря этому t.Parallel не конфликтует с очисткой чужих данных. Фикстура создания — integration/testdata/create.json.

Проверяются CRUD, JSON и Content-Type, точные поля и эффекты INSERT/PATCH/DELETE в БД, даты и триггер, 400/404, пагинация/фильтр и X-Total-Count. Дополнительно — десять конкурентных созданий с уникальными ID, rollback при CHECK, успешный commit и повторное применение миграции. В Git добавлен .github/workflows/integration.yml с PostgreSQL service и перемешанным повторным запуском тестов. Конфигурация CI подготовлена; сохранённые результаты получены локально.

## Запуск API отдельно

```powershell
$env:DB_DSN='postgres://test:test@localhost:54321/notes_test?sslmode=disable'
go run ./cmd/api
```

Маршруты: POST/GET /notes, GET/PATCH/DELETE /notes/{id}. GET списка принимает page, limit (1–100) и q. JSON-модель содержит id/title/content/created_at/updated_at. Остановить контейнер после проверки: docker compose stop (или tasks.ps1 down).

Connection refused: дождаться healthcheck. Ошибка доступа к Docker: проверить, что запущен Docker Desktop. Ошибка SQL выводится сервером в журнал, HTTP-клиент получает общий JSON database_error. Для чистоты тестов не используются общие TRUNCATE при параллельных сценариях.
