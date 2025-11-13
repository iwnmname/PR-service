# Reviewer Assignment Service

Микросервис для автоматического назначения ревьюеров на Pull Request'ы.

## Стек технологий

- Go 1.22
- PostgreSQL 15
- Docker & Docker Compose

## Быстрый старт

```bash
# Копируем переменные окружения
cp .env.example .env

# Запускаем сервис
docker-compose up --build