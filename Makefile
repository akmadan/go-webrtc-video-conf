COMPOSE := docker compose

.PHONY: up upd down logs ps build up-turn up-turn-d restart

up:
	$(COMPOSE) up --build

upd:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

build:
	$(COMPOSE) build

up-turn:
	$(COMPOSE) --profile turn up --build

up-turn-d:
	$(COMPOSE) --profile turn up -d --build

restart:
	$(COMPOSE) down
	$(COMPOSE) up -d --build

