# Architecture

This document describes the KAWAN Threads automation platform at the
subsystem level: the content pipeline, the AMAB scheduler, the publisher,
the analytics feedback loop, the datastore abstraction, and how the two
processes (API + worker) divide responsibility.

```
  ┌─────────────┐   ┌──────────────┐   ┌────────────┐   ┌──────────────┐
  │  Admin/Web  │──▶│ Content API  │──▶│ GEMINI AI  │   │              │
  │ (React app) │   │ (cmd/api)    │   │  adapter   │   │   Threads    │
  └─────────────┘   └──────────────┘   └────────────┘   │    API       │
         ▲                 │                                └─────▲──────┘
         │                 ▼                                      │
         │        ┌────────────────────┐      ┌─────────────────┐│
         │        │  Shared datastore   │      │  Publishing     ││
         │        │  (Firebase OR file) │◀────▶│   worker        ││
         │        └──────────▲─────────┘      │  (cmd/worker)   │┘
         │                   │                 └─────────────────┘
         │                   │ AMAB scheduler │ analytics worker │
         │                   └────────────────┴──────────────────┘
         ▼
   /api/dashboard, /api/analytics, …
```

## Domain model

The core lifecycle of a content item is governed by a strictly ordered state
machine:

```
DRAFT ──approve──▶ QUEUED ──AMAB──▶ SCHEDULED ──publish──▶ PUBLISHING ──▶ PUBLISHED
  │                                     ▲                              │
  └──reject──▶ REJECTED                 └──── cancel ◀──────────────────┘
                                                               (failed → FAILED)
```

Every transition is recorded in the `History` collection (audit trail with
actor, old/new status, and note).

### Key rules enforced in application code

1. **Generated content is always `DRAFT`.** Generation or regeneration never
   auto-publishes — even with `AUTO_APPROVAL=true` the content simply enters
   the queue; it is never pushed to Threads without passing through the queue.
2. **Only `DRAFT` content can be approved** (`approval_usecase.Approve`).
   The second approve of the same item fails with `ErrNotDraft` because the
   status check runs before the idempotency check; the queue always holds
   exactly one active item per content.
3. **Only `QUEUED` content is scheduled** — by AMAB automatically or by an
   operator manually.
4. **Only `SCHEDULED` content is published** — the publisher selects due
   schedules and marks `PUBLISHING` first to prevent duplicate sends.
5. **Rejection** returns a `DRAFT` item back to `DRAFT` only when the current
   state is not `REJECTED` already (idempotent).

## Processes

### 1. `cmd/api` — HTTP API server

- Loads config, logger, selects the datastore, then wires adapters/usecases/
  handlers into the router.
- Middleware stack (outermost → innermost):
  `Recovery → RateLimit(120 burst, 2/s) → CORS → RequestID`.
- Routes use the Go 1.22 `ServeMux` method+pattern matching (see
  `internal/interfaces/http/router/router.go`).
- Reads most entities directly; all **state-changing** content and queue
  operations go through `ContentUseCase` / `ApprovalUseCase` so business
  rules and history writes are never bypassed.

### 2. `cmd/worker` — background workers

Started as three goroutines sharing one root context:

| Worker | Input | Output |
|--------|-------|--------|
| AMAB scheduler (`scheduler.Run`) | Queue items, schedules, performance | `Schedule` + content `QUEUED→SCHEDULED`, one item per tick |
| Publisher (`publishing.Run`) | Due `SCHEDULED` items | `PublishedPost`, content `→PUBLISHED`/`FAILED` |
| Analytics (`analytics.Run`) | Published posts lacking insights | `PostPerformance` records |

All three workers start unconditionally but **self-gate** on the runtime
Threads/Gemini configuration — the publisher and analytics idles until
credentials are present in settings, so credentials can be added later from the
UI without restarting the process.

## Runtime configuration

Config splits into two classes:

- **Environment-locked** (only `cmd` processes read these): `DATA_STORE`,
  `DATA_FILE`, `FIREBASE_DATABASE_URL`, `FIREBASE_SERVICE_ACCOUNT_BASE64`,
  `PORT`, `APP_ENV`.
- **Runtime** (`internal/application/runtimeconfig`): Gemini key/model, Threads
  OAuth credentials & token, scheduler timezone/interval, auto-approval/
  auto-publish, posting caps, exploration rate, retry. Seeded from the
  environment at boot (`Store.Load`), then overlaid with values persisted in the
  datastore via the `SettingsRepository` (filestore node `settings` /
  firebase path `app/settings`).

Flow:

1. API boots → `runtimeconfig.New(settingsRepo, envCfg)` + `Load()` merges
   persisted overrides on top of the env seed.
2. User edits a section in the UI → `PUT /api/settings` → `Update()` normalises
   types (e.g. string `"12"` → int, clamps exploration to `[0,1]`), persists the
   full snapshot, and the in-memory overlay updates immediately.
3. `cmd/worker` builds its own store; every worker tick calls `Store.Reload()`
   so API-side edits apply without a worker restart.
4. Gemini adapter reads the key/model per request; the Threads adapter reads all
   credentials per call. `GET /api/settings/status` reports live
   configured/connected state for the UI.
5. Locked keys (`data_store`, `app_env`, `db_url`, `sa_configured`) are ignored
   in `Update()` — they can never be overridden from the UI.

## AMAB algorithm

`internal/infrastructure/scheduler` implements **AMAB — the Adaptive
Metrics-Based Algorithm** (not to be confused with a classic multi-armed
bandit; it is a bandit-*style* exploration/exploitation scheduler tuned for
social posting). It decides **when** to post by balancing *exploitation*
(repeating proven high-engagement time slots) with *exploration* (trying
under-tested slots to gather fresh signal). The loop is closed by the
analytics worker, so every published post makes future decisions smarter.

### Candidate windows

`defaultPostingWindows` defines the slots evaluated in the configured timezone:
`07:00`, `12:00`, `16:30`, `19:30`, `21:00`. Selection considers **today and
tomorrow**, keeping only windows in the future.

### 1. Slot scoring (`AMABScorer.Score`)

For each candidate hour, the scorer loads every `PostPerformance` record for
the matching pillar published within **±1 hour** of that slot, then:

- **Exploration mode (cold start):** if fewer than **3 samples** exist, the
  slot returns a random score in **[0.5, 1.0]** so under-tested slots still
  get a fair chance.
- **Exploitation mode (warm):** each sample is normalised against the dataset
  maximum and weighted:
  `score = views*0.30 + replies*0.30 + reposts*0.20 + quotes*0.20`
  Recency weight favours fresh data (≤7d → ×1.5, ≤30d → ×1.2, older → ×1.0),
  and the average is scaled by sample confidence `min(1, n/10)` so small
  samples are never fully trusted.

### 2. Window selection (`AMABScorer.SelectPostingWindow`)

- With probability **`EXPLORATION_RATE`** (default **0.20**) a random future
  window is chosen.
- Otherwise the **highest-scored** window wins (greedy exploitation).
- If no candidate remains today or tomorrow, it falls back to the first slot
  two days out — guaranteeing the queue always make progress.

### 3. Strategy recommendation (`GetStrategyRecommendation`)

Aggregates all performance data by **pillar**, **hook type**, and **hour** to
produce a monthly/periodic report used by admins and the analytics worker:
top pillar, top hook type, top-3 best hours, and an automatic suggested action
based on the dominant engagement signal (high reply rate → more conversational
content; high repost rate → more shareable/utility content; low views →
experiment with hooks; otherwise keep the current strategy).

### 4. Guard rails in the scheduling loop (`Scheduler.processQueue`)

Whether a queued item actually gets scheduled depends on more than the score:

- Queue is processed in **priority DESC** order (events=100, membership=80,
  default=50).
- **Max posts/day** (`MAX_POSTS_PER_DAY`) — the daily cap is checked before
  scheduling.
- **Min interval** (`MIN_POST_INTERVAL_MINUTES`) — every existing schedule must
  be farther away than the configured minimum from the submission time.
- **One schedule per tick** — the loop schedules at most a single item, then
  breaks for a predictable cadence.
- Diversity checks (recent pillars/topics) exist at the API boundary of the
  scorer; today `CheckContentDiversity` and `hasDuplicateTopic` are
  conservative stubs, so the practical constraints are the priority, daily
  cap, min-interval, and one-per-tick rules above.

### Configuration knobs

| Knob | Env | Effect |
|------|-----|--------|
| `EXPLORATION_RATE` | default `0.20` | % of decisions that explore instead of exploit |
| `MAX_POSTS_PER_DAY` | default `5` | Hard daily posting cap |
| `MIN_POST_INTERVAL_MINUTES` | default `90` | Min gap between any two posts |
| `SCHEDULER_TIMEZONE` | default `Asia/Jakarta` | Posting window timezone |

See `internal/infrastructure/scheduler/amab.go` for the full implementation.

## Publisher

`internal/worker/publishing/publisher.go`

- Picks due schedules (`FindScheduled(before=now)`), sorts by window.
- **Idempotency:** publishes a container once per `IdempotencyKey`; a second
  attempt for an already-sent key is skipped (fresh content fetches,
  container/publish steps require dedicated Threads containers).
- **Transitions:** `SCHEDULED → PUBLISHING → PUBLISHED` (or `FAILED` after
  `MaxRetry`).
- **Backoff:** `[5s, 30s, 2m]` by default; configurable through the
  `Backoff` field (used to keep tests fast).
- Handles HTTP 429 while retaining a pending server-side publish.

## Analytics feedback loop

`internal/worker/analytics/analytics.go`

- Polls published posts that have no `PostPerformance` row yet.
- Fetches Threads insights (views/likes/replies/reposts/quotes) and persists a
  `PostPerformance` record with pillar/topic/hook metadata and timestamps.
- Refreshes `AMABScorer` strategy recommendation so future windows benefit
  from measured engagement.

## Datastore abstraction

All persistence lives behind the interfaces in
`internal/domain/repository/interfaces.go`. Two implementations exist:

### Firebase (`DATA_STORE=firebase`) — production
- Lightweight RTDB REST client (`internal/infrastructure/firebase`) that uses
  a service-account OAuth2 token source — no heavy Admin SDK.
- Nodes: `threads/content`, `threads/queue`, `threads/schedules`,
  `threads/posts`, `threads/performance`, `threads/topics`,
  `threads/history`, `threads/experiments`.

### JSON file (`DATA_STORE=file`) — dev/demo
- `internal/infrastructure/filestore` keeps the whole dataset in memory and
  persists atomically (temp file + rename) on every mutation.
- **Multi-process safe:** every operation reloads the latest database from
  disk *before* acting, so the API and worker (which run as separate
  processes over a shared volume in docker-compose) observe each other's
  writes. Updates follow last-writer-wins per operation.
- Returns deep-copied entities so callers can never corrupt the store.

## Frontend

`web/` is a mobile-first React SPA built with Vite + Tailwind.

- API layer: `src/services/api.js` (axios) decodes the JSON envelope down to
  `data`.
- Data fetching: `src/hooks/useApi.js` wraps every endpoint in TanStack Query
  mutations/queries and invalidates dependent keys on success.
- Routing: bottom-navigation shell (`AppLayout` + `BottomNav`) with pages for
  Dashboard, Content (list / create / preview), Queue, Schedule, Analytics,
  Topics, and Settings. The preview page preserves the DRAFT-review UX
  (variant hook picker, edit, regenerate, reject, approve & queue).
- The Vite dev server proxies `/api` to `http://localhost:8080`; the
  production nginx image does the same to the `api` service.

## Configuration flow

1. `config.LoadConfig()` loads `.env` (without overriding already-set
   environment variables) then reads the environment.
2. Env vars are validated/used at startup; missing optional integrations
   (Gemini, Threads) degrade gracefully with clear logs.

## Deployment

`Dockerfile` is multi-stage: one Go builder emits both `api` and `worker`
binaries into separate Alpine runtime targets. `web/Dockerfile` builds the
React app and ships it in nginx. `docker-compose.yml` wires the three
services together with a shared named volume for the file datastore.