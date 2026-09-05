# Glossias — Developer Quality Review

Open quality issues. Numbers are stable so wave agents can cite them. Completed or dropped items (rate-limiter TTL, modal a11y, banner API-key compare, goose migrations, React 19 stable, DEV_USER extra gate) are gone. `DEV_USER` itself is the non-prod flag.

Summer 2026 (T1–T16) shipped the five-phase student flow and three authoring editors. Items **#21–#24** are leftovers from that work, not pre-S26 debt.

## Attack plan

One agent owns a file. `#1` / `#10` / `#20` share `main.go` and `src/auth/auth.go`, so they are one agent.

### Wave 1 — five agents, no shared files

| Agent | Items | Owns |
| ----- | ----- | ---- |
| A | #1 CORS, #10 shutdown, #20 beacon | `main.go`, `src/auth/auth.go`, `src/apis/middleware.go`, `src/apis/timetracking.go` |
| B | #9 println, #13 validation, #15 minus `lib/pq` and middleware | `src/pkg/models/story.go`, `story_validation_test.go`, `reconnect.go`, `cache_invalidation.go`, `get.go`, empty/unused packages |
| C | #14 `cn()` | `frontend/app/lib/cn.ts`, frontend lockfile |
| D | #8 frontend, create-only | new `ErrorAlert.tsx` + test; do not rewire pages |
| E | #12 StoryList only | `StoryList.tsx`, `StoryList.css`, `StoryList-sections.css` |

### Wave 2 — after Wave 1 merge

| Agent | Items | Owns |
| ----- | ----- | ---- |
| F | #11 lazy routes, #19 admin flash, **#21 admin chrome** | `frontend/app/routes.ts`, admin routes, `AdminStoryPage`, `AdminStoryNavigation`, `admin.index.tsx` |
| G | #5 types | `types/api.ts`, `types/admin.ts`, `services/api.ts`, forced import sites |
| H | #8 backend | `src/apis/handlers/`, `src/admin/` (not files A owns) |

### Wave 3 — sequential, hard file locks

1. Rest of #15: remove `lib/pq` (touches `main.go` / models; after A and B).
2. **#22** delete retired Vocab/Grammar student stack (routes, components, CSS, handlers). Parallel-safe with #23.
3. **#23** drop unused `story_images` path (migration + models + `/image/*` + student `/images/signed`). Parallel-safe with #22.
4. #4 god components, one file per agent: Produce → Recall → Score → Translate → Identify → `admin.users` → `CourseStudentPerformance` → `ProduceEditor`. After G.
5. #7 DI last. Touches most of `models/` and `main.go`.
6. **#24** only if product wants Identify/Recall to skip when unauthored (today they do not; Produce does).

Do not start two agents on `main.go` or `auth.go`. Do not run G in parallel with anyone editing files that import those types. Do not split god components before G. Do not start DI before A/B/H. Do not run F in parallel with anyone editing admin route files.

---

## 🔴 Critical

### 1. CORS Wildcard Overrides Real CORS Policy

[src/auth/auth.go](file:///Users/jvcte/code/logos-stories/src/auth/auth.go) sets `Access-Control-Allow-Origin: *` on every response, completely negating the allowlist in [src/apis/middleware.go](file:///Users/jvcte/code/logos-stories/src/apis/middleware.go) (defined, **never registered** in `main.go`).

**Fix**: Remove the CORS header from `auth.go`, register `CORSMiddleware()` in [main.go](file:///Users/jvcte/code/logos-stories/main.go).

---

## 🟠 High

### 4. God Components

Data fetching, audio state, business logic, and rendering still live in single files. S26 extracted reducers (`identifyMachine.ts`, `translateMachine.ts`, `produceMachine.ts`) but the shells stayed large. Recall has no machine.

Student (lines as of Wave 0):

| Component | Lines | Notes |
| --------- | ----- | ----- |
| [StoriesProduce.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoriesProduce.tsx) | 871 | First decompose candidate |
| [StoriesRecall.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoriesRecall.tsx) | 755 | No `*Machine.ts` |
| [StoriesScore.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoriesScore.tsx) | 644 | |
| [StoriesTranslate.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoriesTranslate.tsx) | 578 | Machine extracted; shell still a pile |
| [StoriesIdentify.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoriesIdentify.tsx) | 499 | Same |

Admin, grown by S26 authoring and reporting:

| Component | Lines | Notes |
| --------- | ----- | ----- |
| [CourseStudentPerformance.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/components/CourseStudentPerformance.tsx) | 684 | T14/T15 table + reset |
| [admin.users.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/routes/admin.users.tsx) | 661 | Pre-S26 |
| [ProduceEditor.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/components/Admin/ProduceEditor.tsx) | 536 | T7 |
| [StudentStoryDrilldown.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/components/StudentStoryDrilldown.tsx) | 413 | T16 |
| [RecallEditor.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/components/Admin/RecallEditor.tsx) | 360 | T7 |
| [TargetVocabEditor.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/components/Admin/TargetVocabEditor.tsx) | 301 | T7 |

`StoriesVocab` (327) and `StoriesGrammar` (397) are off `defaultPageOrder` but still routed. Do not decompose them — **#22**.

**Fix**: Extract `useStoryAudio` / `useStoryProgress` / per-phase machines. `useReducer` for 10+ state variables. After #5 so types are stable. One file per agent. Admin editor internals after **#21** chrome lands.

---

### 5. Duplicate Type Definitions

`Story`, `StoryMetadata`, `VocabularyItem`, `GrammarItem`, etc. exist in three places with different shapes:

- [types/api.ts](file:///Users/jvcte/code/logos-stories/frontend/app/types/api.ts) — `title?: string | { [key: string]: string }`
- [types/admin.ts](file:///Users/jvcte/code/logos-stories/frontend/app/types/admin.ts) — `title: Record<string, string>`; also T7 types (`TargetVocabulary`, `ProducePage`, `RecallPage`, `StoryContentReadiness`)
- [services/api.ts](file:///Users/jvcte/code/logos-stories/frontend/app/services/api.ts) — student-facing `Line`, `VocabLine`, `TranslateData`, plus Identify/Produce/Recall payloads; this is what student components import

S26 added a third generation of phase types in `services/api.ts` instead of `types/api.ts`.

**Fix**: Single source in `types/api.ts`. Admin types `Pick`/`extend` from it. Phase types stay in `types/api.ts` only.

---

### 21. Unify Admin Interface (S26 leftover)

Story editors already share [AdminStoryPage](file:///Users/jvcte/code/logos-stories/frontend/app/components/Admin/AdminStoryPage.tsx) chrome. Everything around that is still three copies.

What diverged:

- [admin.index.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/routes/admin.index.tsx) keeps its own `EDITOR_BUTTONS` / `PHASE_EDITOR` that must stay in sync with [AdminStoryNavigation](file:///Users/jvcte/code/logos-stories/frontend/app/components/Admin/AdminStoryNavigation.tsx) (`EDITORS`). Order already differs (dashboard: metadata first; nav: annotate first).
- [AdminStoryPage](file:///Users/jvcte/code/logos-stories/frontend/app/components/Admin/AdminStoryPage.tsx) refetches metadata + `content-readiness` on every editor route. The rate limiter exempts those two paths because of this (`src/auth/ratelimit.go`).
- Target Vocab / Produce / Recall each reinvent load / error / busy and [ReadinessPanel](file:///Users/jvcte/code/logos-stories/frontend/app/components/Admin/ReadinessPanel.tsx). Deletes use `window.confirm`; metadata/annotate use `ConfirmDialog`. Unsaved-changes guard exists only on metadata and translate.
- Non-story admin (`admin.users`, `admin.courses`, `admin.performance`, `admin.system`, [CourseStudentPerformance](file:///Users/jvcte/code/logos-stories/frontend/app/components/CourseStudentPerformance.tsx), [StudentStoryDrilldown](file:///Users/jvcte/code/logos-stories/frontend/app/components/StudentStoryDrilldown.tsx)) each own header, width, and auth flash. No admin layout route.
- Raw JSON ([admin.stories.$id.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/routes/admin.stories.$id.tsx)) is still a first-class story URL with no nav tab.

**In scope (a bit, not a rewrite):**

1. One `EDITORS` constant used by the dashboard cards and the story nav.
2. Admin layout: wait for `userInfo`, allow or deny once (**#19**), fetch story title + readiness once for `/admin/stories/:id/*`.
3. Shared load/error chrome for the three phase editors. Destructive actions go through `ConfirmDialog`.
4. Performance, drill-down, users, courses, system use the same page shell (width, header, back link) as story editors.

**Out of scope:** merging the three editors into one page; rewriting the Annotator; decomposing `ProduceEditor` internals (that is #4).

**Fix**: Agent F. Layout is the unification. Child pages drop their own flash gate and readiness fetch.

---

## 🟡 Medium

### 7. Package-Level Global State (Backend)

[src/pkg/models/story.go](file:///Users/jvcte/code/logos-stories/src/pkg/models/story.go) keeps `queries`, `rawConn`, `storageClient` as package globals. `SetDB` takes `any` and type-asserts. This blocks testing and is the transaction race on the shared `queries` var.

`ProduceGradingService` is already injectable. Everything else is not.

**Fix**: `StoryService` struct, dependencies via constructor. Last — touches most of `models/` and `main.go`.

---

### 8. Inconsistent Error Handling

| Layer | Pattern | Problem |
| ----- | ------- | ------- |
| Admin stories | `writeJSONError` in [annotations.go](file:///Users/jvcte/code/logos-stories/src/admin/stories/annotations.go) | Helper is package-local, not shared |
| Other Go handlers | Mix of `http.Error()` (plain text) and `json.Encode()` | Clients can't reliably parse errors |
| `admin/system.go`, `audio.go`, `src/admin/users` | Still `http.Error` | Same |
| Student handlers | Mostly `types.APIResponse` | Closer; not universal |
| S26 admin editors | Local `error` string + ad-hoc `<p>` | Same UX three times |
| React | Bare `<p>Error</p>`, styled alerts, `console.warn` | Inconsistent UX |
| React | No shared `ErrorAlert` | Each page reinvents it |

**Fix (backend)**: Lift `writeJSONError` to a shared helper; convert remaining JSON-route `http.Error`. **Fix (frontend)**: Wave 1 creates `ErrorAlert` only. Wire pages in #21 so this does not collide with #5 / #11 / #19.

---

### 9. Debug Print Statements in Production

Live `fmt.Println` in production paths:

- [reconnect.go](file:///Users/jvcte/code/logos-stories/src/pkg/database/reconnect.go)
- [cache_invalidation.go](file:///Users/jvcte/code/logos-stories/src/pkg/models/cache_invalidation.go)
- [get.go](file:///Users/jvcte/code/logos-stories/src/pkg/models/get.go)
- [story.go](file:///Users/jvcte/code/logos-stories/src/pkg/models/story.go)

`src/apis/timetracking.go` already uses `slog.Debug`.

**Fix**: Replace with `slog`.

---

### 10. Missing Graceful Shutdown

[main.go](file:///Users/jvcte/code/logos-stories/main.go) calls `srv.ListenAndServe()` without signal handling. Active requests are killed on deploy.

**Fix**: `signal.Notify` for `os.Interrupt` and `syscall.SIGTERM` + `srv.Shutdown(ctx)`.

---

### 11. No Code Splitting / Lazy Routes

All routes in [routes.ts](file:///Users/jvcte/code/logos-stories/frontend/app/routes.ts) are eager. `canvas-confetti` is in the main chunk and only used on the score page. S26 added six admin editor routes and a drill-down on top of that.

**Fix**: React Router 7 `lazy()` for admin routes and the score page.

---

### 12. Mixed CSS Paradigms

Tailwind v4 utilities coexist with:

- [StoryList.css](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoryList.css)
- [StoryList-sections.css](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoryList-sections.css)
- [StoriesVocab.css](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoriesVocab.css) — retired page; delete with **#22**
- [CorrectFlash.css](file:///Users/jvcte/code/logos-stories/frontend/app/components/story-components/CorrectFlash.css)

Hardcoded hex colors and `var(--primary)` sit next to `primary-500` theme tokens.

**Fix (Wave 1)**: Migrate StoryList only. Leave Vocab CSS for #22.

---

### 13. Story Validation Logic Bug

[src/pkg/models/story.go](file:///Users/jvcte/code/logos-stories/src/pkg/models/story.go) `Validate()`:

```go
if len(s.Metadata.Title) > minTitleLength {  // checks MAP LENGTH, not string length
    return ErrTitleTooShort
```

This checks the number of translation keys, not title text length. The comparison is inverted. [story_validation_test.go](file:///Users/jvcte/code/logos-stories/src/pkg/models/story_validation_test.go) encodes the bug as “current behavior.”

S26 authoring validation (`ValidateTargetVocabulary` / `ValidateProduceContent` / `ValidateRecallSentences` in [content_readiness.go](file:///Users/jvcte/code/logos-stories/src/pkg/models/content_readiness.go)) is separate and correct. Do not conflate the two.

---

### 14. `cn()` Utility Doesn't Handle Tailwind Conflicts

[cn()](file:///Users/jvcte/code/logos-stories/frontend/app/lib/cn.ts) is `filter(Boolean).join(" ")`. Conflicting utilities (`bg-red-500` vs `bg-primary-500`) both apply.

**Fix**: `clsx` + `tailwind-merge`.

---

### 22. Retired Vocab / Grammar Student Stack (S26 leftover)

Identify replaced Vocab. Produce replaced Grammar. Both old pages stay in [routes.ts](file:///Users/jvcte/code/logos-stories/frontend/app/routes.ts), keep handlers (`stories-vocab.go`, `stories-grammar.go`), queries, and score-page fallback for mixed-generation data.

**Delete** (student surface): routes, `StoriesVocab.tsx` / `StoriesGrammar.tsx`, `StoriesVocab.css`, thin `stories-vocab.tsx` / `stories-grammar.tsx` routes.

**Keep** until no live rows remain: `vocab_*_answers` / `grammar_*_answers` tables, `ComputePhaseScores` legacy fallback, admin reset `phase=vocab|grammar`. Score cards already hide when there are no attempts.

`PageTypeVocab` / `PageTypeGrammar` in [stories-navigation.go](file:///Users/jvcte/code/logos-stories/src/apis/handlers/stories-navigation.go) can go once the routes go.

---

### 23. Unused `story_images` Second Path (S26 leftover)

T3 put `*_path` / `*_bucket` on `target_vocabulary` and `recall_sentences`. T4 added `story_images` mirroring `line_audio_files`. Authoring writes the T3 columns via `POST /phase-assets/upload` ([phase_assets.go](file:///Users/jvcte/code/logos-stories/src/admin/stories/phase_assets.go)). `story_images` is not written by any editor.

Still live and unused by the frontend:

- Table + [story_images.sql](file:///Users/jvcte/code/logos-stories/src/pkg/database/queries/story_images.sql) + [story_images.go](file:///Users/jvcte/code/logos-stories/src/pkg/models/story_images.go)
- Admin `POST /image/upload`, `/image/confirm`, `/image/delete`
- Student `GET /api/stories/:id/images/signed` ([stories-images.go](file:///Users/jvcte/code/logos-stories/src/apis/handlers/stories-images.go)) — no frontend caller

**Fix**: Goose down-migration dropping `story_images` if the table is empty in prod (or after a one-off check). Remove the T4 handlers and student signed-image route. Phase assets and `line_audio_files` stay.

---

### 24. Navigation Skip Rules Disagree (S26 leftover)

[content_readiness.go](file:///Users/jvcte/code/logos-stories/src/pkg/models/content_readiness.go) is the authoring source of truth (5 words + assets, 2 segments + explanation, 5 sentences). Student nav ([navigation.go](file:///Users/jvcte/code/logos-stories/src/pkg/models/navigation.go)) does not read it.

As built:

- Produce with **no** segments counts as complete → nav **skips**.
- Identify / Recall with **no** content never count as complete → student still visits an empty page (tested in `TestPageCompletionEmptyStory`).
- Translate with no request row is incomplete.

SUMMER_2026 hardest-part #4 asked nav to skip phases whose content is absent, using the `Ready` flags. That was not done. Empty Identify/Recall degrade (notice + Continue) instead of 500, so this is product, not a crash.

**Fix**: Only if product wants skip-when-unauthored. Then `IdentifyComplete` / `RecallComplete` should match Produce (`total == 0` → complete), or nav should consult `GetStoryContentReadiness`. Do not change this in Wave 1/2.

---

## 🟢 Low

### 15. Dead Code Cleanup

| File | Issue |
| ---- | ----- |
| [src/pkg/utils/utils.go](file:///Users/jvcte/code/logos-stories/src/pkg/utils/utils.go) | Empty package |
| [src/apis/users/course_users.go](file:///Users/jvcte/code/logos-stories/src/apis/users/course_users.go) | Empty file; [users.go](file:///Users/jvcte/code/logos-stories/src/apis/users/users.go) is live — delete the empty file only |
| [src/apis/middleware.go](file:///Users/jvcte/code/logos-stories/src/apis/middleware.go) | Same as #1. Drop this row when CORS is registered |
| [src/pkg/templates/templates.go](file:///Users/jvcte/code/logos-stories/src/pkg/templates/templates.go) | Unused template engine (SPA frontend) |
| `github.com/lib/pq` in go.mod | Blank-imported; [db.go](file:///Users/jvcte/code/logos-stories/src/pkg/database/db.go) uses pgx stdlib. Wave 3 — imports sit in files A/B own |
| `story_images` stack | **#23**, not this list |

Admin `GET /stories/hello` in [handler.go](file:///Users/jvcte/code/logos-stories/src/admin/stories/handler.go) is a leftover ping. Delete with #15 or #23.

---

### 19. Admin Route Flashing

Admin pages (`admin.system.tsx`, `admin.performance.tsx`, `admin.courses.tsx`, others) check `userInfo` during render. The admin UI flashes before “Access Denied.” No admin layout route exists.

**Fix**: This is the auth slice of **#21**. Gate in the admin layout; child pages drop their own flash gate. Do not do #19 as a separate pass.

---

### 20. Beacon Auth Gap

[frontend/app/lib/timeTracking.ts](file:///Users/jvcte/code/logos-stories/frontend/app/lib/timeTracking.ts) uses `navigator.sendBeacon` on page leave. Beacons cannot send auth headers.

`/api/time-tracking/record` is in `byPassURLS` in [src/auth/auth.go](file:///Users/jvcte/code/logos-stories/src/auth/auth.go). Record requires a cache session ID minted by authenticated `/api/time-tracking/start` ([src/apis/timetracking.go](file:///Users/jvcte/code/logos-stories/src/apis/timetracking.go)). Callers cannot invent time without a stolen ID. Still a session-theft / time-inflation hole. S26 made time gate score completion across five phases, so inflated time is more valuable than before.

**Fix**: Keep the beacon unauthenticated. Bind record to the cached session (already) and reject missing/expired IDs with a stable JSON error. Handler tests for the contract. No new auth scheme in Wave 1.
