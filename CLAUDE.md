# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CloudDisk — a full-stack file sharing application. Go backend (Gin) serves an embedded React frontend as a single binary. Files are uploaded by an authenticated admin and shared publicly via unique codes with optional password protection and expiration.

## Build & Run Commands

### Docker (production)
```bash
docker compose up -d --build    # Build and start app + PostgreSQL
docker compose down             # Stop services
```

### Backend (Go 1.22, from `backend/`)
```bash
cd backend
go build -o server .            # Build binary
go run .                        # Run dev server (port 8080)
go mod tidy                     # Sync dependencies
```

### Frontend (React 19 + Vite, from `frontend/`)
```bash
cd frontend
npm install                     # Install dependencies
npm run dev                     # Dev server with hot reload (proxies API to :8080)
npm run build                   # Production build → dist/
npx tsc --noEmit                # Type check
npm run lint                    # ESLint
```

### shadcn/ui components
```bash
cd frontend
npx shadcn@latest add <component>   # Add a new UI component
```

## Architecture

**Embedded SPA pattern**: Frontend builds to `frontend/dist/`, which is embedded into the Go binary via `go:embed` in `backend/frontend.go`. The Gin server serves these static assets with SPA fallback routing (non-file-extension routes serve `index.html`).

**Key data flow**: Admin authenticates → uploads files → creates shares (8-char hex code) → public users access `/s/:code` → verify password if needed → download files.

### Backend (`backend/`)
- `main.go` — Gin router setup, route definitions, CORS config
- `frontend.go` — `go:embed` of frontend dist assets
- `config/config.go` — Environment variable loading with defaults
- `database/db.go` — PostgreSQL/GORM connection, auto-migration
- `models/` — GORM models: `File` (UUID, name, stored name, size, MIME) and `Share` (UUID, code, title, password hash, expiration, many-to-many files)
- `handlers/` — `auth.go` (JWT login), `file.go` (upload/list/delete), `share.go` (CRUD + public endpoints)
- `middleware/auth.go` — JWT Bearer token validation

### Frontend (`frontend/src/`)
- `pages/Login.tsx` — Admin login
- `pages/Admin.tsx` — Tabbed dashboard (files + shares management)
- `pages/ShareView.tsx` — Public share view with password verification
- `lib/api.ts` — Axios client with JWT interceptor, auto-logout on 401
- `components/` — FileUpload, FileList, ShareDialog, plus shadcn/ui primitives in `ui/`

### API Routes
- `POST /api/auth/login` — Login → JWT token
- `GET|POST /api/s/:code[/verify|/download/:fileId]` — Public share access
- `POST /api/files/upload`, `GET /api/files`, `DELETE /api/files/:id` — Protected file ops
- `POST /api/shares`, `GET /api/shares`, `DELETE /api/shares/:id` — Protected share ops

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_USER` | clouddisk | Database user |
| `DB_PASSWORD` | *(required)* | Database password |
| `DB_NAME` | clouddisk | Database name |
| `JWT_SECRET` | *(required)* | JWT signing key |
| `ADMIN_USER` | admin | Login username |
| `ADMIN_PASS` | admin123 | Login password |
| `UPLOAD_DIR` | ./uploads | File storage path |
| `PORT` | 8090 | Docker host port binding |

## Key Conventions

- **UI language is Chinese** — all user-facing strings are in Chinese
- Frontend path alias: `@/*` maps to `frontend/src/*`
- UUIDs for all entity primary keys
- Passwords hashed with bcrypt; JWT tokens expire in 24 hours
- File upload limit: 100MB per file
- Docker multi-stage build: Node → Go → Alpine minimal image
- App binds to `127.0.0.1:8090` (designed for nginx reverse proxy)
