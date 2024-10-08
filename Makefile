# Variables
DOCKER_COMPOSE = docker compose

# Targets
.PHONY: up down logs

# Start containers
up:
	$(DOCKER_COMPOSE) up --build -d

# Stop and remove containers
down:
	$(DOCKER_COMPOSE) down

# Show logs
logs:
	$(DOCKER_COMPOSE) logs -f
