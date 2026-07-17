.PHONY: up upd down build logs ps

# Bring up the full stack (mysql + migrate + backend + frontend + phpmyadmin).
up:
	docker compose up --build

# Same as `up`, but detached.
upd:
	docker compose up --build -d

# Build images without starting containers.
build:
	docker compose build

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps

# For Go-only workflows without Docker (run/build/migrate/fresh/create/tidy/test):
#   cd backend && make <target>
