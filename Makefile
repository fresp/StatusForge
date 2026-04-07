# ============================================================
#  Statora — Docker Compose Makefile
# ============================================================

COMPOSE       := docker compose
COMPOSE_FILE  := docker-compose.yml
PROJECT_NAME  := Statora

# Default target
.DEFAULT_GOAL := help

# -------------------- Lifecycle --------------------

.PHONY: up
up: ## Start all services in detached mode
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d

.PHONY: up-build
up-build: ## Build images and start all services
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d --build

.PHONY: down
down: ## Stop and remove containers, networks
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down

.PHONY: down-v
down-v: ## Stop and remove containers, networks, AND volumes (⚠️  data loss)
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down -v

.PHONY: restart
restart: ## Restart all services
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) restart

.PHONY: stop
stop: ## Stop all services (without removing)
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) stop

.PHONY: start
start: ## Start previously stopped services
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) start

# -------------------- Build --------------------

.PHONY: build
build: ## Build all images
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) build

.PHONY: build-no-cache
build-no-cache: ## Build all images without cache
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) build --no-cache

# -------------------- Logs --------------------

.PHONY: logs
logs: ## Follow logs for all services
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f

.PHONY: logs-server
logs-server: ## Follow logs for server service
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f server

.PHONY: logs-mongo
logs-mongo: ## Follow logs for MongoDB
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f mongo

.PHONY: logs-redis
logs-redis: ## Follow logs for Redis
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f redis

# -------------------- Status --------------------

.PHONY: ps
ps: ## Show running containers
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) ps

.PHONY: ps-all
ps-all: ## Show all containers (including stopped)
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) ps -a

.PHONY: top
top: ## Display running processes
	$(COMPOSE) -f $(COMPOSE_FILE) -p $(PROJECT_NAME) top

# -------------------- Shell Access --------------------

.PHONY: shell-server
shell-server: ## Open a shell in the server container
	docker exec -it status-forge-unified /bin/sh

.PHONY: shell-mongo
shell-mongo: ## Open mongosh in the MongoDB container
	docker exec -it status-forge-mongodb mongosh

.PHONY: shell-redis
shell-redis: ## Open redis-cli in the Redis container
	docker exec -it status-forge-redis redis-cli

# -------------------- Cleanup --------------------

.PHONY: clean
clean: down ## Stop services and prune dangling images
	docker image prune -f

.PHONY: clean-all
clean-all: down-v ## Stop services, remove volumes, and prune images
	docker image prune -f

# -------------------- Help --------------------

.PHONY: help
help: ## Show this help message
	@echo ""
	@echo "  Statora — Available Commands"
	@echo "  ================================"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
