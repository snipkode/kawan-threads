# KAWAN AI — Threads Automation Platform

Automatic AI-powered content engine for the **KAWAN** community on Threads
(@kawan). Generates, reviews, schedules, and publishes advocacy content on a
smart cadence — and learns from engagement to keep improving.

> **Workflow at a glance**
> `Gemini AI → DRAFT → human review → APPROVE → QUEUE → AMAB scheduler → SCHEDULED → publisher → Threads API → PUBLISHED → analytics → AMAB feedback loop`

---

## Feature highlights

- **AI copywriting (Gemini)** — pillars, audiences, tones, single posts /
  thread series, hook variants, built-in quality validation.
- **Human-in-the-loop approval** — generated content is *always* a DRAFT;
  nothing touches the queue until an admin approves it. Optional auto-approval.
- **AMAB scheduler** — the Adaptive Metrics-Based Algorithm picks the best
  posting window from historical engagement (views/likes/replies/reposts),
  balancing exploration vs exploitation, and enforces max posts/day,
  min-interval, and priority rules. See `docs/architecture.md`.
- **Reliable publishing** — idempotent Threads publishing with token
  persistence, retry with backoff, and rate-limit handling. Setup guide:
  `docs/threads-api.md`.
- **Analytics feedback loop** — a worker collects post insights and feeds them
  back into AMAB strategy recommendations.
- **Full web dashboard** — mobile-first React app: dashboard, content
  review/edit/preview, queue management, schedules, analytics, topics,
  settings.
- **Zero-credential dev mode** — a JSON-file datastore lets the whole stack
  run locally without Firebase; swap one env var for production.

---

## Tech stack

| Layer     | Technology |
|-----------|-----------|
| Backend   | Go 1.23, stdlib `net/http` (Go 1.22 method+pattern mux) |
| Storage   | Firebase Realtime Database (REST) **or** local JSON file (dev) |
| AI        | Google Gemini (custom provider adapter) |
| Publishing| Meta Threads API (custom adapter, OAuth + long-lived token) |
| Frontend  | React 18 · Vite 5 · Tailwind CSS · React Router 6 · TanStack Query 5 · axios |
| Infra     | Docker multi-stage images · docker-compose · nginx (SPA + API proxy) |

---

## Repository layout

```
cmd/
  api/       HTTP API server (entry point)
  worker/    Background workers (scheduler, publisher, analytics)
internal/
  application/
    runtimeconfig/          Runtime settings store (seed env → overlay UI edits)
    usecase/                Business logic (Content, Approval)
  config/                .env + environment configuration
  domain/
    entity/              Core domain entities & enums
    port/                Ports (interfaces) for AI provider & Threads API
    repository/          Persistence interfaces
  infrastructure/
    filestore/           JSON-file datastore (dev/demo, thread-safe, multi-process)
    firebase/            Firebase RTDB client + repositories
    gemini/              Gemini adapter + content validator + tests
    scheduler/           AMAB scorer + scheduler service + tests
    threads/             Threads API adapter
  interfaces/http/
    handler/             HTTP handlers + JSON envelope helpers
    middleware/          Rate limit, CORS, request ID, recovery
    router/              Route registration
  logger/                Structured logging helpers
  worker/
    publishing/          Idempotent publisher + tests
    analytics/           Insights collector (AMAB feedback)
web/                     Vite + React + Tailwind frontend
Dockerfile               Multi-stage Go build (api + worker targets)
web/Dockerfile           Node build → nginx runtime
nginx.conf               SPA fallback + /api proxy
docker-compose.yml       api + worker + web with shared data volume
```

---

## Getting started (local, no credentials)

The default `DATA_STORE=file` uses a local JSON file, so no Firebase account is
required to try the platform.

```bash
# 1. Configure (defaults already point at the file datastore)
cp .env.example .env

# 2a. Run backend (two processes share ./data/db.json)
go run ./cmd/api                # http://localhost:8080
go run ./cmd/worker             # scheduling + publishing loop

# 2b. Run the web dashboard
cd web
npm install
npm run dev                     # http://localhost:5173 (proxies /api → :8080)
```

Open http://localhost:5173. Without `GEMINI_API_KEY` the *create* page reports
AI generation unavailable — everything else (review, queue, schedules,
analytics, topics, settings) works.

### Docker (same zero-credential experience)

```bash
docker compose up -d --build
# Web: http://localhost:3000   API: http://localhost:8080
```

`docker-compose.yml` defaults to `DATA_STORE=file` and shares one volume
(`data/db.json`) between api and worker.

---

## Production configuration

Only the **datastore** and the **Firebase service account** (plus app infra:
`PORT`, `APP_ENV`) are environment-locked. Everything else — Gemini API key &
model, Threads credentials, scheduler timezone/interval, auto-approval,
posting limits, exploration rate, retries — is seeded from the environment **and
then editable at run time via the UI** (`GET/PUT /api/settings`), persisted in
the datastore and shared with the worker process. Values edited in the UI take
effect without a restart; the worker re-reads them on every tick.

| Variable | Locked | Description |
|---------|--------|-------------|
| `DATA_STORE` | ✔ env | `file` (dev) or `firebase` (prod). Default `firebase` via Go flag, `file` in compose. |
| `DATA_FILE` | ✔ env | Path to the JSON datastore. Default `data/db.json`. |
| `FIREBASE_DATABASE_URL` | ✔ env | e.g. `https://my-project.firebaseio.com` |
| `FIREBASE_SERVICE_ACCOUNT_BASE64` | ✔ env | Base64-encoded service-account JSON |
| `GEMINI_API_KEY` | UI | Runtime — enables generation (key `gemini_api_key`) |
| `GEMINI_MODEL` | UI | Runtime — default `gemini-3.6-flash` |
| `THREADS_CLIENT_ID` / `CLIENT_SECRET` / `REDIRECT_URI` | UI | Runtime — Threads OAuth app |
| `THREADS_ACCESS_TOKEN` / `USER_ID` | UI | Runtime — long-lived token + account (token also saved by OAuth callback) |
| `SCHEDULER_TIMEZONE` | UI | Runtime — default `Asia/Jakarta` |
| `SCHEDULER_INTERVAL_MINUTES` | UI | Runtime — scheduler tick, default `5` |
| `AUTO_APPROVAL` | UI | Runtime — skip the manual approval step. Default `false` |
| `AUTO_PUBLISH` | UI | Runtime — publish automatically. Default `false` |
| `MAX_POSTS_PER_DAY` | UI | Runtime — daily posting cap. Default `5` |
| `MIN_POST_INTERVAL_MINUTES` | UI | Runtime — min spacing. Default `90` |
| `EXPLORATION_RATE` | UI | Runtime — AMAB exploration vs exploitation. Default `0.20` |
| `MAX_RETRY` | UI | Runtime — max publish attempts. Default `3` |
| `PORT` / `APP_ENV` | ✔ env | API port / environment. Defaults `8080` / `development` |
| `VITE_API_BASE_URL` | — | Frontend API origin. Default `http://localhost:8080` |

> Secrets are loaded by the **backend only**. Never prefix a secret with
> `VITE_` — Vite inlines `VITE_*` variables into the public bundle.

---

## API reference

All responses use a consistent envelope: `{"success": true, "data": ...}` or
`{"success": false, "message": "...", "code": "..."}`.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Liveness check |
| GET | `/api/dashboard` | Post/queue/schedule counts + engagement totals |
| GET | `/api/content` | List content (query: `status`, `pillar`, `limit`, `offset`) |
| POST | `/api/content/generate` | Generate content → always saved as DRAFT |
| GET | `/api/content/:id` | Single content |
| GET | `/api/content/:id/preview` | Preview bundle (content, variants, approval, quality) |
| PUT | `/api/content/:id` | Edit content / pick hook variant |
| POST | `/api/content/:id/regenerate` | Regenerate with Gemini |
| POST | `/api/content/:id/approve` | Approve → enqueue (only from DRAFT) |
| POST | `/api/content/:id/reject` | Reject with a reason |
| GET | `/api/queue` | Queued items (enriched with content) |
| POST | `/api/queue/:id/remove` | Remove item from queue |
| POST | `/api/queue/:id/priority` | Adjust priority |
| POST | `/api/content/:id/schedule` | Manual schedule (content must be QUEUED) |
| DELETE | `/api/content/:id/schedule` | Cancel a manual schedule |
| GET | `/api/schedules` | All schedules (enriched with content) |
| GET | `/api/analytics` | Engagement totals |
| GET | `/api/analytics/hour` | Best publishing hours |
| GET | `/api/analytics/pillar` | Performance per pillar |
| GET | `/api/topics` · POST | Topic seeds |
| GET | `/api/settings` · PUT | Platform settings (runtime-editable, persisted) |
| GET | `/api/settings/status` | Datastore + Gemini/Threads configuration state |
| GET | `/api/auth/threads` | Redirect to Threads OAuth |
| GET | `/api/auth/threads/callback` | OAuth callback |

The API enforces a per-IP token-bucket rate limit (120 burst, ~2 req/s).

---

## Testing

```bash
go build ./... && go vet ./... && go test ./...
cd web && npm run build
```

Covered areas: Gemini content validator, AMAB scoring, approval use case
flow, publishing with retry/backoff/idempotency, rate limiter, and the
file datastore (CRUD, filtering, ordering, multi-instance persistence,
concurrency).

---

## Live end-to-end flow (dev mode)

1. **Create** — generate a post (needs `GEMINI_API_KEY`) → stored as DRAFT.
2. **Review** — pick a hook variant, edit, or reject in the web UI.
3. **Approve & Queue** — `POST /content/:id/approve` moves it to the queue.
4. **AMAB scheduling** — the worker schedules queued posts into the best
   posting windows (`SCHEDULER_INTERVAL_MINUTES`).
5. **Publish** — at the scheduled time the publisher pushes to Threads
   (needs `THREADS_*`); publishes idempotently with retry/backoff.
6. **Learn** — the analytics worker pulls insights; AMAB recommendations
   improve future windows.

---

## Roadmap

- [x] Content generation + review pipeline
- [x] Approval, queue, priority management
- [x] AMAB scheduler + diversity/duplicate/min-interval guards
- [x] Idempotent Threads publisher with retry & backoff
- [x] Analytics collection + AMAB feedback
- [x] Web dashboard (mobile-first)
- [x] Dev datastore + Dockerized zero-credential run
- [ ] Firebase RTDB security rules
- [ ] A/B hook-variant experiments
- [ ] Webhook / cron-based re-engagement