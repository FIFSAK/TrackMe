# Сборка образа Docker
build:
	docker-compose build

# Запуск контейнеров
up:
	docker-compose up -d

# Остановка и удаление контейнеров
down:
	docker-compose down

# Перезапуск контейнеров
restart: down up

.PHONY: build up down restart
