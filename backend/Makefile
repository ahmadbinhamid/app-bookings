.PHONY: run build migrate fresh create tidy test

# Start the HTTP API server (live reload via `air` if installed).
run:
	go run ./cmd/server

# Compile everything (syntax/typecheck across the module).
build:
	go build ./...

# Apply any migrations that haven't run yet.
migrate:
	go run ./cmd/migration run

# Drop all tables and re-run every migration from scratch.
fresh:
	go run ./cmd/migration fresh

# Generate a new migration file. Usage: make create services  (or: make create name=services)
create:
	@go run ./cmd/migration create $(or $(name),$(filter-out create,$(MAKECMDGOALS)))

# Tidy module dependencies.
tidy:
	go mod tidy

# Run the test suite.
test:
	go test ./...

# Catch-all so a positional migration name (`make create services`) isn't
# treated as its own target. Explicit targets above still take precedence.
%:
	@:
