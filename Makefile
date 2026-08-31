.PHONY: up down build logs ps backend-up backend-down backend-test

up:
	docker compose up -d --build

down:
	docker compose down

build:
	docker compose build

logs:
	docker compose logs -f

ps:
	docker compose ps

backend-up:
	$(MAKE) -C backend docker-up

backend-down:
	$(MAKE) -C backend docker-down

backend-test:
	$(MAKE) -C backend test
