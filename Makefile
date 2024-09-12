# Variables
DOCKER_COMPOSE = docker-compose
COMMON_INFRA_DIR = ../nd-common-infra

# Targets
.PHONY: init up down logs

# Start containers
up: check_common_infra
	$(DOCKER_COMPOSE) up --build -d

# Stop and remove containers
down:
	$(DOCKER_COMPOSE) down

# Show logs
logs:
	$(DOCKER_COMPOSE) logs -f

# Check and start nd-common-infra if not running
check_common_infra:
	@if ! docker-compose -f $(COMMON_INFRA_DIR)/docker-compose.yml ps -q | grep -q .; then \
		echo "Starting common-infra..."; \
		cd $(COMMON_INFRA_DIR) && $(DOCKER_COMPOSE) up -d; \
	else \
		echo "common-infra is already running"; \
	fi