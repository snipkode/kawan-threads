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

`cmd/worker` starts the scheduler unconditionally; the publisher and analytics
workers only start when Threads credentials are configured.

## AMAB scheduler

`internal/infrastructure/scheduler` implements the **Autonomous Multi-Arm
Bandit** posting algorithm.

### Strategy recommendation

`AMABScorer.GetStrategyRecommendation` categorises each pillar based on its
engagement rate:

| Rate < past-avg | Category | Action |
|-----------------|----------|--------|
| fd < 0.7        | Struggling | Add new pillars/topics (novelty) |
| 0.7 ≤ fd ≤ 1.0  | Stable    | Continue with slight variation |
| fd > 1.0        | Strong    | Increase posting frequency (+10%) and assignment weight |

`RecommendStrategy` combines the categorical index with the exploration rate
(`EXPLORATION_RATE`, default 0.20) into the final top-2 strategy list.

### Posting-window selection

- Scores candidate slots using historical engagement per weekday/hour
  (`performance_score`).
- Additionally applies diversity (recent pillars/topics), duplicate-topic
  exclusion, and the min-interval guard before committing to a window.
- Runs at most once per tick (`break` after the first accepted item) for a
  predictable cadence.

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