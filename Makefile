# Сборка образа Docker
build:
	docker build -t TrackMe .

# Запуск контейнеров
up:
	docker-compose up -d

# Остановка и удаление контейнеров
down:
	docker-compose down

# Перезапуск контейнеров
restart: down up

.PHONY: build up down restart
