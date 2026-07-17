# app-booking

Boilerplate for a FlowPOS mini-app: Go REST API (Gin + `database/sql`) backed
by MySQL, plus a React/Vite frontend using FlowPOS's shared `@flowposltd/ui`
component library. Copied from the same tooling as the `appointments` and
`quotes` apps so it installs into tenant-dashboard the same way.

This is intentionally just the skeleton — auth, the marketplace
install/uninstall lifecycle, migrations engine, docker-compose stack, and a
minimal "it's alive" page all work end to end. The actual domain (employees,
services, bookings — see the app-bookings design doc) is not built yet.

## Layout

```
app-booking/
├── docker-compose.yml            full stack: mysql + migrate + backend + frontend + phpmyadmin
├── .env.example                  docker-compose variables (DB creds, host port overrides)
├── Makefile                      thin wrapper around `docker compose`
│
├── backend/
│   ├── Dockerfile
│   ├── go.mod / go.sum
│   ├── Makefile                  run / build / migrate / fresh / create / tidy / test
│   ├── .air.toml                 live reload config for `air`
│   ├── .env.example              config for running the Go binary directly on the host
│   ├── cmd/
│   │   ├── server/               HTTP API server
│   │   └── migration/            migration CLI (create / run / fresh)
│   ├── internal/
│   │   ├── auth/                 minimal HS256 JWT (Generate/Parse)
│   │   ├── config/                loads env into a Config struct
│   │   ├── db/                   Connect() — pooled MySQL connection
│   │   ├── server/
│   │   │   ├── server.go         router setup + wiring — add new feature modules here
│   │   │   └── handlers/         HTTP handlers (auth, lifecycle, me, errors)
│   │   └── modules/
│   │       └── installation/     marketplace install state, one row per tenant
│   └── database/
│       ├── migrations/           timestamped migration files only
│       ├── migrator/             the migration engine (registry, run, fresh)
│       └── seeds/                seed scripts (optional)
│
└── frontend/
    ├── Dockerfile
    ├── nginx.conf                 proxies /api/ to the backend container
    ├── package.json
    └── src/
        ├── main.tsx               adopts ?token= / ?theme= from the embed URL
        ├── App.tsx                router + embed handshake
        ├── app/use-embed.ts       apps-sdk postMessage handshake (theme, resize)
        ├── components/
        │   ├── providers/         react-query + tooltip providers
        │   └── shell/             sidebar nav + layout shell
        ├── lib/
        │   ├── auth-token.ts      session JWT storage
        │   ├── theme.ts           ?theme= adoption
        │   ├── query-keys.ts      react-query key registry
        │   └── api/               axios client (with interceptors) + one file per domain
        ├── pages/                 HomePage.tsx placeholder — add real pages here
        ├── types/                 mirrors backend response shapes
        └── utils/                 cn() (clsx + tailwind-merge)
```

Dependencies flow inward: `cmd` → `server` / `config` / `db` → `modules/<feature>`
(`handlers` → `service` → `repository` → `model`). Only `cmd` wires everything
together.

## Running with Docker (recommended)

```bash
cp .env.example .env   # defaults work out of the box; edit if you like
make up                # docker compose up --build
```

This starts, in order: `mysql` → a one-off `migrate` job (applies pending
migrations, then exits) → `backend` → `frontend` → `phpmyadmin`.

| Service | URL |
|---|---|
| Frontend | http://localhost:3003 |
| Backend API | http://localhost:8086/api/v1 |
| phpMyAdmin | http://localhost:8087 |
| MySQL (external access) | localhost:3310 |

Other Make targets: `make upd` (detached), `make down`, `make logs`, `make ps`, `make build` (build images only).

Ports are overridable via `.env` — the defaults are offset from the sibling
`ai-builder`, `quotes`, and `appointments` stacks so all four can run at once.

## Running locally without Docker

**Backend:**
```bash
cd backend
cp .env.example .env   # then edit DB credentials
go mod tidy
make migrate           # apply pending migrations
make run                # or `air` for live reload
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev             # runs the Vite dev server + `go run ../backend/cmd/server` together
```

Getting a dev auth token (once the backend is running, with `JWT_DEV_TOKENS=true`):

```bash
curl -X POST http://localhost:8086/api/v1/dev/token \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": 1, "user_id": 1, "user_email": "dev@example.com"}'
```

Open `http://localhost:3003/?token=<the token>` — the frontend adopts it from
the query string (see `src/lib/auth-token.ts`) the same way the tenant
dashboard delivers it when embedding this app for real.

## Migrations

Migrations are plain Go files in `backend/database/migrations` — that folder
holds *only* the change files. Each one registers itself with the engine in
`backend/database/migrator` via `init()`, exposing `Up`/`Down` functions that
receive a `*sql.DB`. The engine tracks applied migrations in a `migrations`
table and applies them oldest-first by filename timestamp.

The server itself never auto-migrates — under Docker, the `migrate` compose
service runs the CLI once before `backend` starts.

```bash
cd backend
make create name=create_widgets   # scaffolds 2026..._create_widgets.go
# fill in the Up/Down bodies, then:
make migrate                      # or, from the repo root: make up (runs it in Docker)
```

## Adding a feature module

Follow the `installation` module as the template:
1. `backend/internal/modules/<feature>/{model,repository,service}.go`
2. A migration in `backend/database/migrations/` for its table(s)
3. A handler in `backend/internal/server/handlers/<feature>.go`, wired up in `internal/server/server.go`
4. `frontend/src/lib/api/<feature>.ts` + a type in `frontend/src/types/index.ts`
5. A page in `frontend/src/pages/`, added to `App.tsx`'s routes and `components/shell/nav.tsx`
