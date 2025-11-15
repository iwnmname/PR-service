# Reviewer Assignment Service

Микросервис для автоматического назначения ревьюеров на Pull Request'ы.

## Описание

Сервис автоматически назначает до двух активных ревьюеров из команды автора при создании PR, позволяет переназначать ревьюеров и управлять командами.

## Стек технологий

- **Go 1.22** - язык разработки
- **PostgreSQL 15** - база данных
- **Docker & Docker Compose** - контейнеризация
- **golang-migrate** - миграции БД

## Быстрый старт

```bash
# Клонируем репозиторий
git clone https://github.com/iwnmname/PR-service.git

# Запускаем сервис (БД + миграции + приложение)
docker-compose up --build
```

Сервис будет доступен на `http://localhost:8080`

### Остановка

```bash
# Остановить
docker-compose down

# Остановить и удалить данные
docker-compose down -v
```

## Примеры использования

### 1. Создать команду с участниками

```bash
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alice", "is_active": true},
      {"user_id": "u2", "username": "Bob", "is_active": true},
      {"user_id": "u3", "username": "Charlie", "is_active": true}
    ]
  }'
```

### 2. Создать PR (автоматически назначит ревьюеров)

```bash
curl -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add search feature",
    "author_id": "u1"
  }'
```

### 3. Изменить активность пользователя

```bash
curl -X POST http://localhost:8080/users/setIsActive \
  -H "Content-Type: application/json" \
  -d '{"user_id": "u2", "is_active": false}'
```

### 4. Смерджить PR

```bash
curl -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{"pull_request_id": "pr-1001"}'
```

### 5. Получить статистику сервиса

```bash
curl http://localhost:8080/statistics
```

### 6. E2E тесты

Автоматические end-to-end тесты:

```bash
# Запустить E2E тесты
make test-e2e

# Или вручную
docker-compose -f docker-compose.e2e.yml up --build -d
docker logs -f reviewer_e2e_tests
docker-compose -f docker-compose.e2e.yml down -v
```