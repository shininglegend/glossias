# Glossias — Developer Quality Review

A full-stack review of the Go backend and React frontend, organized from **quick wins** to **large refactors**.

## Summary — Attack Plan

Tackle these in roughly this order (numbers may be incorrect)

1. **Next up** (#1 CORS, #10 graceful shutdown, #13 fmt.Println, #13 validation bug, #7 dependency injection)
2. **And then** (#12 error standardization, #6 rate limiter cleanup, #10 modal a11y)
3. **This might take awhile** (#16 timing-safe compare, #9 type unification, #19 dead code, #16 CSS consolidation)
4. **This. could take even longer** (#15 code splitting, #21 proper migrations, #8 component decomposition,)
5. **Later, maybe** (#3 dev bypass)

---

## 🔴 Critical — Fix Immediately

### 1. CORS Wildcard Overrides Real CORS Policy

[auth.go](file:///Users/jvcte/code/logos-stories/src/apis/auth.go) sets `Access-Control-Allow-Origin: *` on every response, completely negating the proper allowlist in [middleware.go](file:///Users/jvcte/code/logos-stories/src/apis/middleware.go) (which is defined but **never registered** in `main.go`).

**Fix**: Remove the CORS header from `auth.go`, register `CORSMiddleware()` in [main.go](file:///Users/jvcte/code/logos-stories/main.go).

---

## 🟠 High — Address Soon

### 2. Rate Limiter Memory Leak

[ratelimit.go](file:///Users/jvcte/code/logos-stories/src/apis/ratelimit.go) stores a `rate.Limiter` per IP in a `map` that **never gets cleaned up**. Same pattern in `dbhealth.go`. Under sustained traffic this grows unboundedly — a DoS vector.

**Fix**: Use a TTL cache (e.g., `bigcache` which you already depend on, or `sync.Map` with periodic cleanup).

---

### 3. Hardcoded Dev Auth Bypass

[auth.go](file:///Users/jvcte/code/logos-stories/src/apis/auth.go) has a dev bypass with password `"12345678"`. If `DEV_USER` is accidentally set in production, all auth is bypassable.

**Fix**: Gate behind a `GO_ENV=development` check, or remove entirely and use Clerk test tokens instead.

---

### 4. God Components (400–640 Lines Each)

Several frontend components mix data fetching, audio state, business logic, and rendering in a single file:

| Component                                                                                               | Lines | `useState` Calls |
| ------------------------------------------------------------------------------------------------------- | ----- | ---------------- |
| [admin.users.tsx](file:///Users/jvcte/code/logos-stories/frontend/app/routes/admin.users.tsx)           | ~643  | —                |
| [StoriesScore](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoriesScore.tsx)         | ~471  | —                |
| [StoriesTranslate](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoriesTranslate.tsx) | ~437  | **15**           |
| [StoriesVocab](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoriesVocab.tsx)         | ~419  | —                |
| [StoriesGrammar](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoriesGrammar.tsx)     | ~415  | —                |

**Fix**: Extract custom hooks (`useStoryAudio`, `useStoryProgress`, `useTranslationState`) and decompose into sub-components. Consider `useReducer` for components with 10+ state variables.

---

### 5. Duplicate Type Definitions

`Story`, `StoryMetadata`, `VocabularyItem`, `GrammarItem`, etc. are defined in **both** [types/api.ts](file:///Users/jvcte/code/logos-stories/frontend/app/types/api.ts) and [types/admin.ts](file:///Users/jvcte/code/logos-stories/frontend/app/types/admin.ts) with subtly different shapes (e.g., `title?: string | { [key: string]: string }` vs `title: Record<string, string>`).

**Fix**: Single source of truth in `types/api.ts`; have admin types extend/pick from base types.

---

### 6. Modal Accessibility Failures

Custom modals in admin pages and `ConfirmDialog` lack:

- `role="dialog"` / `aria-modal="true"`
- Focus trapping
- Escape key handling
- `aria-label` on icon-only buttons (play/pause)

**Fix**: Use the native `<dialog>` element (supported in all modern browsers) or add ARIA attributes + focus trap to existing modals.

---

## 🟡 Medium — Improve When Convenient

### 7. Package-Level Global State (Backend)

[story.go](file:///Users/jvcte/code/logos-stories/src/apis/stories/story.go) uses package globals for `queries`, `rawConn`, `storageClient`. The `SetDB` function accepts `any` and manually type-asserts. This prevents testing and caused the transaction race condition (#1).

**Fix**: Wrap in a `StoryService` struct, inject dependencies via constructor.

---

### 8. Inconsistent Error Handling

| Layer            | Pattern                                                              | Problem                                |
| ---------------- | -------------------------------------------------------------------- | -------------------------------------- |
| Go handlers      | Mix of `http.Error()` (plain text) and `json.Encode()` (JSON)        | Clients can't reliably parse errors    |
| Go handlers      | Some return 200 with error JSON body                                 | Breaks HTTP semantics                  |
| React components | Mix of bare `<p>Error</p>`, styled alerts, and silent `console.warn` | Inconsistent UX                        |
| React            | No shared `ErrorBanner` component                                    | Each component reinvents error display |

**Fix (backend)**: Standardize on a `writeError(w, status, message)` helper. **Fix (frontend)**: Create a shared `ErrorAlert` component.

---

### 9. Debug Print Statements in Production

`fmt.Println` calls leak data to stdout in production paths:

- [timetracking.go](file:///Users/jvcte/code/logos-stories/src/apis/stories/timetracking.go) — leaks session data
- [reconnect.go](file:///Users/jvcte/code/logos-stories/src/pkg/database/reconnect.go), [cache_invalidation.go](file:///Users/jvcte/code/logos-stories/src/pkg/cache/cache_invalidation.go), [get.go](file:///Users/jvcte/code/logos-stories/src/apis/stories/get.go)

**Fix**: Replace with `slog` calls (you already have a good `slog` setup).

---

### 10. Missing Graceful Shutdown

[main.go](file:///Users/jvcte/code/logos-stories/main.go) calls `srv.ListenAndServe()` without signal handling. Active requests are killed on deploy.

**Fix**: Add `signal.Notify(stop, os.Interrupt, syscall.SIGTERM)` + `srv.Shutdown(ctx)`.

---

### 11. No Code Splitting / Lazy Routes

All routes are eagerly imported. `canvas-confetti` (15KB) is bundled in the main chunk despite only being used on the score page.

**Fix**: Use React Router 7's `lazy()` for heavy routes (admin pages, score page).

---

### 12. Mixed CSS Paradigms

Tailwind v4 utility classes coexist with raw CSS files ([StoryList.css](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoryList.css), [StoriesVocab.css](file:///Users/jvcte/code/logos-stories/frontend/app/components/StoriesVocab.css)) that use hardcoded hex colors and custom class names. Some CSS references `var(--primary)` while Tailwind uses `primary-500` theme tokens.

**Fix**: Migrate component CSS files to Tailwind utilities, or at minimum use theme tokens (`theme()` function) in the CSS files.

---

### 13. Story Validation Logic Bug

[story.go](file:///Users/jvcte/code/logos-stories/src/apis/stories/story.go) line ~303:

```go
if len(s.Metadata.Title) > minTitleLength {  // checks MAP LENGTH, not string length
    return ErrTitleTooShort
```

This checks the **number of translation keys**, not the title text length. The comparison direction is also inverted.

---

### 14. `cn()` Utility Doesn't Handle Tailwind Conflicts

The custom [cn()](file:///Users/jvcte/code/logos-stories/frontend/app/lib/cn.ts) is just `filter(Boolean).join(" ")`. It can't resolve conflicting Tailwind classes (e.g., `bg-red-500` vs `bg-primary-500` both apply).

**Fix**: Use `clsx` + `tailwind-merge` (standard pattern).

---

## 🟢 Low — Nice to Have

### 15. Dead Code Cleanup

| File                                                                                                    | Issue                                    |
| ------------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| [src/pkg/utils/utils.go](file:///Users/jvcte/code/logos-stories/src/pkg/utils/utils.go)                 | Empty package                            |
| [src/apis/users/course_users.go](file:///Users/jvcte/code/logos-stories/src/apis/users/course_users.go) | Empty package                            |
| [src/apis/middleware.go](file:///Users/jvcte/code/logos-stories/src/apis/middleware.go)                 | Defined but never registered             |
| [src/pkg/templates/templates.go](file:///Users/jvcte/code/logos-stories/src/pkg/templates/templates.go) | Template engine, unused (SPA frontend)   |
| `github.com/lib/pq` in go.mod                                                                           | Likely vestigial (project uses `pgx/v5`) |

---

### 16. Timing-Safe API Key Comparison

[banner.go](file:///Users/jvcte/code/logos-stories/src/apis/admin/banner.go) compares API keys with `==`. Use `crypto/subtle.ConstantTimeCompare` for defense-in-depth.

---

### 17. Migrations Gitignored

[.gitignore](file:///Users/jvcte/code/logos-stories/.gitignore) contains `migrations/*.sql`, so migration files aren't version controlled. Schema is applied via embedded `schema.sql` on every startup (`CREATE TABLE IF NOT EXISTS`) — this isn't a proper migration system and will break when you need to ALTER tables.

---

### 18. React 19 RC in Production

[package.json](file:///Users/jvcte/code/logos-stories/frontend/package.json) uses `react@^19.0.0-rc`. Consider pinning to a stable release.

---

### 19. Admin Route Flashing

Admin pages check `userInfo.is_super_admin` inside the component render, causing a brief flash of the admin UI before showing "Access Denied." Authorization should be checked at the route/layout level.

---

### 20. Beacon Auth Gap

[timeTracking.ts](file:///Users/jvcte/code/logos-stories/frontend/app/services/timeTracking.ts) uses `navigator.sendBeacon` on page leave, but beacons can't carry auth headers. The endpoint apparently accepts unauthenticated `FormData` — worth verifying this is intentional and rate-limited.
