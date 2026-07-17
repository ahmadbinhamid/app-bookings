# Build stage: compile the API server and migration CLI.
FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migration

# Runtime stage: minimal image.
FROM alpine:3.20

RUN adduser -D -u 10001 app
COPY --from=build /out/server /usr/local/bin/
COPY --from=build /out/migrate /usr/local/bin/

# cmd/migration always os.MkdirAll()s "database/migrations" relative to cwd
# (even for `run`/`fresh`, not just `create`), so the working directory must
# exist and be writable by the non-root user.
WORKDIR /app
RUN chown app:app /app

USER app
EXPOSE 8080

# The server never auto-migrates — docker-compose runs a one-off `migrate`
# service before starting this one.
ENTRYPOINT ["server"]
