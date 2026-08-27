# Summer 2026 Mode — Story Flow Redesign Spec

Redesign of the student story experience into five sequential phases (~15 min total): **Watch (1:45) → Identify (3:30) → Translate (2:30) → Produce (5:00) → Recall (2:15)**. This replaces the current **Video → Vocab → Translate → Grammar → Score** flow.

This spec is grounded in the current codebase. Each section states what exists today (with file paths), what is net-new, and the concrete changes required. An implementer should be able to follow this without re-deriving the architecture.

## Task breakdown (async-friendly, with dependencies)

Tasks with no unmet dependencies can run in parallel as independent agent tasks. "Depends on" means the listed task must be merged first.

Complexity is 1–10 on the difficulty of getting it *right* (state, races, external systems), not on volume of typing — a small diff can rate high. File counts are new + modified files, excluding `sqlc`-generated output and lockfiles.

- [X] **T1 — Migration system.** Adopt `golang-migrate` or `goose`, un-gitignore `migrations/*.sql`, and convert `schema.sql` startup application into a baseline migration. *Depends on: nothing (hard prerequisite for T3).* *Complexity: 5/10 · ~8 files.*
- [X] **T2 — Navigation + placeholder phases (F1).** Extend `PageType` and `defaultPageOrder` to the new five-phase flow, add the three thin routes/components as placeholders, and add time-tracking route matching for the new phase names. *Depends on: nothing.* *Complexity: 3/10 · ~12 files.*
- [X] **T3 — Schema + SQLC for new content (F2/F3).** Add `target_vocabulary`, `produce_segments`, `produce_submissions`, `recall_sentences`, and the identify/recall answer-log tables, with queries, `sqlc generate`, and model functions. *Depends on: T1.* *Complexity: 6/10 · ~18 files — 5 new tables, ~5 query files, sqlc regen, model funcs.*
- [X] **T4 — Image storage.** Create the images bucket and mirror the audio signed-upload/signed-read flow in models and admin handlers. *Depends on: nothing.* *Complexity: 4/10 · ~6 files — mirrors an existing pattern end to end.*
- [X] **T5 — Modal accessibility fix.** Fix `ConfirmDialog` (role/aria, focus trap, Escape) before it gets cloned into the Identify and Produce popups (developer_review.md #6). *Depends on: nothing.* *Complexity: 3/10 · ~3 files.*
- [X] **T6 — `useAudioPlayer` extensions (F4).** Add pause-on-specific-lines, replay-single-line, skip-lines set, and timed resume behind new options with unchanged defaults, plus isolated hook tests. *Depends on: nothing.* *Complexity: 6/10 · ~3 files — few files, but dense concurrent-audio logic + tests.*
- [X] **T7 — Admin authoring editors.** Admin CRUD UI + endpoints for target vocab (with audio/image upload), produce segments/explanation, and recall sentences, including validation (5 words, ≥2 occurrences, 5 ordered sentences). *Depends on: T3, T4.* *Complexity: 8/10 · ~18 files — largest surface area: 3 editors x (admin handler + model + UI).*
- [X] **T8 — Watch phase.** Pre-video plot-summary screen and `videoWatched` gating in `StoriesVideo.tsx`. *Depends on: T2.* *Complexity: 2/10 · ~3 files.*
- [X] **T9 — Translate rework.** Rewrite `StoriesTranslate.tsx` as an explicit state machine (quota, consecutive cap, restart/fast-forward, auto-skip); backend unchanged. *Depends on: T2, T6.* *Complexity: 9/10 · ~5 files — single-file rewrite, but the hardest state machine in the app.*
- [ ] **T10 — Identify phase.** New `GET /identify` + `POST /check-identify` endpoints, target-word rendering, and the picture-quiz popup with line replay. *Depends on: T2, T3, T4, T5, T6.* *Complexity: 8/10 · ~14 files — spans schema, storage, audio, segment rendering, and a new popup.*
- [ ] **T11 — Recall phase.** Audio-only playback, `@dnd-kit` sequencing UI, and `GET /recall` + `POST /check-recall` endpoints with answer logging. *Depends on: T2, T3, T6.* *Complexity: 7/10 · ~11 files — includes adding and wiring @dnd-kit.*
- [ ] **T12 — Produce phase (frontend + submit endpoint).** Timed two-segment translation UI, reference reveal, explanation popup, and `POST /produce` storing submissions (ungraded). *Depends on: T2, T3, T5.* *Complexity: 6/10 · ~10 files.*
- [ ] **T13 — AI grading.** Anthropic SDK integration, grading prompt + iteration on sample answers, fail-open behavior, and rate limiting on the submit path. *Depends on: T12 (and developer_review.md #2 rate-limiter fix).* *Complexity: 7/10 · ~8 files — small diff, high risk: external API, cost, latency, prompt iteration.*
- [ ] **T14 — Score page rework.** New accuracy categories, five-phase time breakdown, and incomplete-detection for the new phases in `stories-score.go` + `StoriesScore.tsx`. *Depends on: T10, T11, T12 (T13 for graded produce scores).* *Complexity: 6/10 · ~9 files — must tolerate mixed-generation data.*

Parallel-start set: **T1, T2, T4, T5, T6** can all begin immediately.

## How the current flow works (read this first)

- **Phase ordering is server-driven.** Each phase component calls `POST /api/stories/:id/next` with `{ currentPage }`, handled by `Navigate` in `src/apis/handlers/stories-navigation.go` (see `defaultPageOrder` at line ~30). The frontend hook is `frontend/app/hooks/useNavigationGuidance.ts`, and the `PageType` union lives in `frontend/app/types/api.ts:105-111` (`"list" | "video" | "vocab" | "translate" | "grammar" | "score"`). **New phases plug in by extending both.**
- **Each phase = one thin route file + one big component.** Routes in `frontend/app/routes/stories-*.tsx` start time tracking (`frontend/app/lib/timeTracking.ts`) and render a component from `frontend/app/components/Stories*.tsx`. Follow the style conventions in `frontend/app/components/README.md`.
- **Audio is per-line, no timestamps.** `useAudioPlayer` in `frontend/app/components/story-components/AudioPlayer.tsx` (225 lines) prefetches per-line signed URLs from `GET /api/stories/:id/audio/signed?label=complete`, plays sequentially, supports pause-after-line, and reports `currentLineIndex` / `playedLines` via callbacks. Line-level highlight rendering is `story-components/StoryLine.tsx` (RTL-aware, supports inline translation box).
- **Schema lives in goose migrations:** `src/pkg/database/migrations/*.sql`, applied on startup and read directly by SQLC as its schema source (T1/T3; the old `schema.sql` is gone). SQLC queries in `src/pkg/database/queries/*.sql`, regenerated with `sqlc generate` into `src/pkg/generated/db/` (never hand-edit).
- **Storage:** Supabase Storage via `storage-go`; bucket `"audio-files"`, signed URLs generated on demand in `src/pkg/models/audio_files.go`. Images mirror this in the `"images"` bucket via `src/pkg/models/story_images.go` (T4). Assets owned 1:1 by a target word or recall sentence are addressed by the path stored on that row instead, signed via `src/pkg/models/asset_urls.go` (T7 — see F2 below).
- **Scoring:** append-only answer logs (`vocab_correct_answers` / `vocab_incorrect_answers`, `grammar_*_answers`), aggregated live by `GetScoresData` in `src/apis/handlers/stories-score.go` using `CalculateScoreWithRetriesAllowed` in `src/pkg/models/overview.go`. No persistent numeric score column exists.
- **No AI integration exists anywhere in the backend** (`go.mod` has no LLM SDK). Produce-phase grading is entirely net-new.
- **No images, no picture-choice UI, no Anki linkage, no drag-and-drop library** exist in the repo today.

## Cross-cutting foundations (build these first)

### F1. New page types and navigation order

- Backend: update `defaultPageOrder` in `src/apis/handlers/stories-navigation.go` to `video → identify → translate → produce → recall → score`. Decide whether old pages (`vocab`, `grammar`) stay reachable for old stories or are removed; recommend keeping handlers but removing them from the default order.
- Frontend: extend `PageType` in `frontend/app/types/api.ts`; add route entries in `frontend/app/routes.ts`; new thin route files `stories-identify.tsx`, `stories-produce.tsx`, `stories-recall.tsx` mirroring `frontend/app/routes/stories-vocab.tsx` (start tracking, render component).
- Time tracking route matching: `scores.sql` derives per-phase time via `route LIKE '%vocab%'` etc. — add matching clauses for `identify`, `produce`, `recall` in `src/pkg/database/queries/time_tracking.sql` / `scores.sql` and `GetUserStoryTimeTracking` in `src/pkg/models/get-scores.go`.

### F2. Target vocabulary (5 words per story)

The Identify and Recall phases center on exactly 5 "target" vocabulary words, each appearing ≥2 times in the story text.

- `vocabulary_items` (schema.sql L108) already has `word`, `lexical_form`, `line_number`, `position_start/end`. Net-new: a way to mark the 5 target lexical forms and attach per-word assets. Add a table:
  ```sql
  CREATE TABLE target_vocabulary (
    id SERIAL PRIMARY KEY,
    story_id INT REFERENCES stories(story_id) ON DELETE CASCADE,
    lexical_form TEXT NOT NULL,
    audio_path TEXT,          -- word pronunciation, audio-files bucket
    audio_bucket TEXT,
    correct_image_path TEXT,  -- the matching picture (images bucket)
    image_bucket TEXT,
    UNIQUE (story_id, lexical_form)
  );
  ```
  Occurrences in the text are found by joining on `vocabulary_items.lexical_form` — no per-occurrence duplication needed.
- **Picture options**: the spec says students "see the same five picture options previously introduced through Anki flashcards." There is no Anki integration in the repo; the images must be uploaded as assets. The 5 options shown in each pop-up are simply the 5 target words' images for that story (one correct, four distractors). This means one image per target word suffices — no separate distractor table.
- **Image storage**: create a Supabase bucket (e.g. `"images"`) and mirror the existing audio flow: signed upload URL + confirm (admin), signed read URLs (student). Reuse the pattern in `src/pkg/models/audio_files.go` and `src/admin/stories/audio.go` — add parallel functions/handlers for images rather than generalizing the audio code.
- Admin: extend the annotations/admin editor (`src/admin/stories/handler.go`, `frontend/app/routes/admin.stories.$id.*`) with target-vocab selection + audio/image upload per word. **As built (T7)**: a dedicated editor at `admin/stories/:id/target-vocab` rather than an extension of the annotate page, since target-word selection is per story rather than per line.

### F3. Story content additions (schema.sql + queries + models)

Add to the content model (new columns/tables, admin CRUD for each):

| Item | Suggested home | Used by |
|---|---|---|
| Plot summary (two sentences) | `story_descriptions` already exists per language — reuse it as the Watch summary, or add `plot_summary` column to `stories` if descriptions serve another purpose. Check current use before deciding. | Watch |
| Video asset | `stories.video_url` exists (plain URL, YouTube or direct file — `frontend/app/components/StoriesVideo.tsx` handles both). Sufficient; no schema change. | Watch |
| Per-line English translations | `line_translations` table exists. Sufficient. | Translate |
| Produce segments (2 per story) | New table `produce_segments(id, story_id, segment_order (1/2), english_text, reference_hebrew, grammar_point_id REFERENCES grammar_points)` plus a per-story `produce_explanation TEXT` (the contrastive grammar explanation popup) — either a column on `stories` or on a new `story_produce_config` table. | Produce |
| Recall sentences (5 per story) | New table `recall_sentences(id, story_id, sequence_order (1–5), hebrew_text, target_vocab_id REFERENCES target_vocabulary, image_path, image_bucket)`. | Recall |

After schema changes: add queries under `src/pkg/database/queries/`, run `sqlc generate`, add model functions in `src/pkg/models/`, wire student GET endpoints in `src/apis/handlers/` and admin CRUD in `src/admin/stories/`.

**As built (T3)** — migration `00003_summer_2026_content.sql`, queries in `queries/{target_vocabulary,produce,recall,identify}.sql`, models in `models/{target_vocabulary,produce,recall,identify}.go`. Decisions downstream tasks depend on:

- The Produce explanation is its own table, `story_produce_explanations(story_id PK, explanation_text)`, not a `stories` column — this keeps `Story`/`StoryMetadata` serialization untouched.
- `identify_incorrect_answers` stores `selected_target_vocab_id` (the target word whose picture was clicked) rather than free text, since the five options are always the story's own target words.
- `recall_correct_answers` / `recall_incorrect_answers` hold one row per sentence per attempt; `models.SaveRecallAttempt` takes the submitted ordering, validates it is a permutation of the story's sentences (`ErrInvalidRecallOrder`), logs each row, and returns per-position correctness. T11's `POST /check-recall` should just call it.
- `models.GetUserStoryProduceSummary` aggregates the **latest submission per segment**, returning `SegmentsSubmitted` / `SegmentsGraded` / `AverageScore`. T14's "both submissions present" completeness check is `SegmentsSubmitted == ProduceSegmentsPerStory`; ungraded segments are excluded from `AverageScore`, so use `SegmentsGraded` to tell "pending" from "scored 0".
- Identify and Recall summaries share `models.AnswerSummary{CorrectCount, IncorrectCount}`, ready for `CalculateScoreWithRetriesAllowed`. Totals come from `CountStoryTargetVocabulary` / `CountStoryRecallSentences`.
- Content-count constants live in the models package: `TargetWordsPerStory`, `ProduceSegmentsPerStory`, `RecallSentencesPerStory`. T7's validation should use them. The DB enforces `segment_order IN (1,2)` and `sequence_order BETWEEN 1 AND 5`, but the "exactly 5 target words, each appearing ≥2×" rule is not expressible as a constraint and remains T7's job.
- Image columns (`target_vocabulary.correct_image_path`, `recall_sentences.image_path`, both with a `*_bucket` sibling) are in place and nullable.

**~~Open question for T7 — two ways to attach an image.~~ Resolved by T7.** T3 and T4 landed independently and each followed the spec, but they overlapped: T4 added a `story_images(story_id, file_path, file_bucket, label)` table mirroring `line_audio_files`, while T3 added path/bucket columns directly on `target_vocabulary` and `recall_sentences` as F2/F3 specified. Both could name the same file.

**The T3 columns are the source of truth** for which asset belongs to a target word or recall sentence — the relationship is 1:1, so a column on the owning row expresses it better than a label string. `story_images` is not written by the authoring editors at all; it remains available for story-level images with no single owner, and if none materialize it should be dropped rather than left as a second, unused way to attach the same file.

Bytes still move through T4's mechanism — signed Supabase upload URL, direct browser PUT, then confirm — but the confirm step writes the path onto the owning row instead of inserting a `story_images` row. Consequences for downstream tasks:

- **`POST /api/admin/stories/{id}/phase-assets/upload`** (`src/admin/stories/phase_assets.go`) mints the upload URL. It takes `{kind, ownerId, fileName}` where kind is `target_vocab_image` / `target_vocab_audio` / `recall_image`, and derives both bucket and path from the kind and the owning row — neither is caller-supplied. The path prefix (`stories/{id}/image_target_vocab_{wordId}_`, `stories/{id}/word_audio_{wordId}_`, `stories/{id}/image_recall_{sentenceId}_`) is re-validated when the path is attached, so a path minted for one word or one asset kind cannot be attached to another. Word-pronunciation audio needed its own path shape because T4's `/audio/upload` requires a line number.
- **Signed reads** go through `models.GetSignedURLForPath(ctx, bucket, path, expires)`, with `SignTargetVocabularyURLs` / `SignRecallSentenceURLs` filling the non-persisted `AudioURL` / `ImageURL` fields on the T3 structs. T10 and T11 should reuse those helpers; authorization stays the caller's job (`CanUserEditStory` for admin, `CanUserAccessCourse` for students). `models.GetSignedImageURL` (T4, keyed by `story_images.image_id`) is not the path for phase assets.
- **Replaced and deleted assets are removed from storage** by the editors, so a superseded upload does not linger. Deleting a target word leaves recall sentences pointing at NULL (`ON DELETE SET NULL`), which the readiness report below flags.
- **T4's `/image/upload` and `/image/confirm` now sanitize the caller-supplied `label`** before it enters the path; previously only the filename was sanitized, so a label could escape the story prefix.

**As built (T7) — validation and readiness.** The authoring rules the schema cannot express live in `models.ValidateTargetVocabulary` / `ValidateProduceContent` / `ValidateRecallSentences` (`src/pkg/models/content_readiness.go`), each returning a `PhaseReadiness{Ready, Issues}`. `models.GetStoryContentReadiness` runs all three, and `GET /api/admin/stories/{id}/content-readiness` serves the report; the three editors show it as a checklist. Notes for downstream tasks:

- **Navigation (F1/T14) should read these `Ready` flags** rather than re-deriving completeness — this is the "stories missing the new content must degrade sanely" requirement in hardest-part #4. Identify needs 5 words each with audio, a picture, and ≥`MinTargetWordOccurrences` (2) annotated occurrences; Produce needs both ordered segments with a grammar point plus the explanation; Recall needs 5 sentences filling positions 1–5, each with a picture and its own distinct target word.
- **Target words can only be chosen from lexical forms already annotated in the story**, via the new `GetStoryLexicalFormCounts` query. The editor lists candidates with their counts and hides ineligible ones; the backend re-checks on create and on rename, and refuses a sixth word. So T15 authoring order is: annotate vocabulary first, then pick target words, then recall sentences (which link to those words).
- **Segments and recall sentences are addressed by position, not row ID** (`PUT /produce/segments/{order}`, `PUT /recall/sentences/{order}`), matching the fixed two- and five-slot shapes and the `UpsertProduceSegment` / `UpsertRecallSentence` models.
- `models.ErrDuplicate` now maps a Postgres unique violation into a sentinel, so handlers can answer 409 without inspecting error codes.

### F4. Reusable frontend pieces

- **Modal**: `frontend/app/components/ui/ConfirmDialog.tsx` is the only modal — use it as the base for the Identify picture-quiz popup and the Produce explanation popup (generalize or copy the `fixed inset-0 bg-black/50` pattern).
- **Continue button**: `story-components/CompletionMessage.tsx` — reuse at the end of every phase.
- **Audio player**: extend `useAudioPlayer` rather than rewriting: it already supports pause-after-line, played-line tracking, and current-line callbacks. Needed extensions: pause-on-specific-lines (Identify: pause only on target-word lines), replay-single-line (Identify), skip-lines set (Translate: auto-skip already-translated lines), programmatic timed resume (Translate's 2s/5s pauses).
- **Drag-and-drop** (Recall): no library installed. Add `@dnd-kit/core` + `@dnd-kit/sortable` (small, maintained, works with React 19) rather than hand-rolling HTML5 DnD.
- **Timers** (Produce 1.5-min limit, Translate 2s/5s pauses): plain `setTimeout`/`useEffect`; no library.

## Phase 1 — Watch (1:45)

Current: `frontend/app/components/StoriesVideo.tsx` plays `metadata.videoUrl` (YouTube embed or `<video>`), tracks `videoWatched` (currently unused for gating), shows Continue via `CompletionMessage`.

Changes:

1. Show the two-sentence plot summary **before** the video starts (fetch via `GET /api/stories/:id/metadata` — description is already on `StoryMetadata`; see F3 for where the summary lives). Simple pre-video screen with a "Start video" button.
2. Optionally gate the Continue button on `videoWatched` (the state already exists at `StoriesVideo.tsx` — it's set on `ended`/`timeupdate > 0.8` but never used). Recommend gating for direct video; YouTube embeds can't reliably report progress without the IFrame API, so leave those ungated.
3. Video production (Ken Burns pan/zoom over illustrations, synced narration, no on-screen text) is a **content-pipeline task, not code**: videos are produced externally and set as `video_url` per story via the existing admin metadata editor. No backend change.

**As built (T8)** — all in `StoriesVideo.tsx`; the route name stays `video` (no navigation or time-tracking changes). The pre-video screen shows `metadata.description.text` as the summary (falls back to a generic prompt when empty) with a "Start video" button; the player mounts only after that click, so direct `<video>` gets `autoPlay` and YouTube embeds get `?autoplay=1`, both honored as user-gesture autoplay. Continue is gated on `videoWatched` (ended or >80% progress) for direct video only; YouTube stays ungated, per the spec. Stories with no `video_url` keep the existing "Skip to next step" screen.

## Phase 2 — Identify (3:30)

Replaces the vocab dropdown-cloze page (`StoriesVocab.tsx`). New component `StoriesIdentify.tsx`, route `stories/:id/identify`.

Behavior:

1. Render the full Hebrew story text (all lines) with the current line highlighted in sync with narration — this is exactly what `StoryLine.tsx` + `useAudioPlayer` already do (line-granular sync via per-line audio files; **no word-level timestamps needed or available**).
2. Target vocab words rendered in a distinct color: match `vocabulary_items` whose `lexical_form` is in the story's `target_vocabulary` set, using existing `position_start/end` offsets. Extend the segment renderer (model: `VocabTextRenderer.tsx` segments, or the `TextSegment` type in `src/apis/types/responses.go`) with a `"target"` segment type.
3. When a line containing a target word finishes playing: pause (via the pause-on-specific-lines extension from F4) and open a popup showing (a) the lexical form, (b) auto-playing word audio (signed URL from `target_vocabulary.audio_path`), (c) the story's 5 target-word images in randomized order. Student must click the correct image to dismiss; wrong picks show feedback and allow retry.
4. On dismiss: **replay the same line**, then continue to the next line.
5. Record correct/incorrect picks server-side for scoring — reuse the append-only pattern: new tables `identify_correct_answers` / `identify_incorrect_answers` shaped like `vocab_correct_answers` (schema.sql L177), new endpoint `POST /api/stories/:id/check-identify` modeled on `CheckVocab` (`src/apis/handlers/stories-vocab.go:178` → `models.SaveVocabScore` in `src/pkg/models/save-scores.go`).

New student endpoint: `GET /api/stories/:id/identify` returning lines (segmented with target markers), per-line signed audio URLs, and the 5 target words with signed word-audio + image URLs. Model it on `GetVocabPage` (`stories-vocab.go:19`).

If a target word appears fewer than 2 times, that's a content-authoring error — validate in the admin editor, not at runtime.

## Phase 3 — Translate (2:30)

Reworks the existing `StoriesTranslate.tsx` (433 lines). Today it pauses after **every** line and asks "Do you fully comprehend?" with a reveal option; requested lines are saved via `PUT /api/stories/:id/translate?lines=[...]` into `translation_requests` (UNIQUE(user_id, story_id), `requested_lines INT[]`).

New behavior (audio plays continuously; the student interrupts):

1. Play all lines from the start with sync highlighting, **no automatic pause**.
2. Student may click a line to request its translation while that line is playing or while the following line is playing. Clicking any other line does nothing.
3. On request: let the current line finish → 2-second silent pause (prediction beat) → reveal the English translation (reuse `StoryLine`'s existing `showTranslation`/`translation` props) → hold 5 seconds → resume.
4. Constraints (all client-side state, persisted at the end):
   - Minimum 4, maximum 7 requests per story. Disable further requests at 7.
   - No more than 3 **consecutive** lines translated; after 3 in a row, requests are disabled until a line plays untranslated.
5. If the story ends with fewer than 4 requests: restart audio from line 1, show a "fast forward" button (skip to next line), and auto-skip lines already translated (the skip-lines extension from F4). Loop until the minimum is met.
6. On completion, persist the requested line set via the existing `PUT /api/stories/:id/translate` — `translation_requests.requested_lines` already fits. The existing `GET /api/stories/:id/translate` (all lines + translations, from `line_translations`) also fits unchanged.

This phase is almost entirely a frontend rewrite of `StoriesTranslate.tsx` plus the `useAudioPlayer` extensions; backend is unchanged.

**As built (T9)** — the interaction logic is a pure reducer in `frontend/app/lib/translateMachine.ts` (phases `idle → playing ⇄ paused`, `playing → awaitingLineEnd → predicting → revealing → playing`, and `complete`), unit-tested in `translateMachine.test.ts` before being wired to audio. `StoriesTranslate.tsx` is split into a data-loading shell and a `TranslateSession` that owns the machine and the audio hook. Decisions worth knowing:

- **Side effects are commands, not flags.** The reducer emits `{type: "playFrom", index}` with a `commandSeq`; the component runs each command once from an effect. The 2s/5s beats are single component timers keyed on the phase, so there is never more than one timer or one pending resume in flight.
- **One request in flight at a time.** Clicks during `awaitingLineEnd` / `predicting` / `revealing` are ignored; eligibility is re-evaluated against the new current/previous line once playback resumes.
- **The consecutive cap is checked by line adjacency**, not by a temporal streak: a request is refused if it would make a run of more than 3 translated lines, joined from either side. This matters on restart passes, where a request can sit next to lines translated on an earlier pass.
- **Restart passes end as soon as the minimum is met** (right after that reveal), rather than replaying to the end again; the first pass always plays through.
- **Short stories lower the minimum.** `effectiveMinRequests(lineCount)` is `min(4, lineCount − ⌊lineCount/4⌋)`, so a story too short to hold four requests under the cap can still complete instead of looping forever.
- **`useAudioPlayer` gained `onPlaybackEnd` and `onPauseAfterLine`** (T6 style: optional, default unchanged, tested) so the component learns "the story ran out" and "I stopped after line N" as explicit events from the hook rather than inferring them from `isPlaying` / `currentLineIndex` flips — inferring from `isPlaying` misfired when the student paused and resumed while a request was waiting. Pending requests use `pauseOnLines = {currentLine}`, which the hook reads at `ended` time — if a click lands as a line ends, the pause simply happens after the next line. Pausing while a request waits keeps the request; resume replays the current line and waits for its end again.
- Requested lines are still persisted 0-indexed via the unchanged `PUT /api/stories/:id/translate?lines=[…]`, once, on completion.

## Phase 4 — Produce (5:00)

Entirely net-new. New component `StoriesProduce.tsx`, route `stories/:id/produce`. Replaces the grammar click-hunting page (`StoriesGrammar.tsx`) as the grammar-focused phase.

Behavior:

1. `GET /api/stories/:id/produce` returns the two segments (English text, position in story for context display, grammar point) — from the new `produce_segments` table (F3). Show the surrounding Hebrew story text with the segment's slot indicated.
2. For each segment: show the English, a `Textarea` (`frontend/app/components/ui/Textarea.tsx`) for the Hebrew attempt, and a 90-second countdown. On submit **or timeout**, reveal the reference Hebrew below the student's attempt for self-comparison, then a button to advance to segment 2.
3. After both segments: popup (ConfirmDialog pattern) showing the authored `produce_explanation` — how the grammar point works in each segment, what it contributes, and how the two compare.
4. `POST /api/stories/:id/produce` submits `{ segment_id, student_text }`. Backend stores it (new table `produce_submissions(id, user_id, story_id, segment_id, student_text, ai_score, ai_feedback, graded_at, created_at)`) and grades it.

**AI grading (net-new integration):**

- Add the Anthropic Go SDK (`github.com/anthropics/anthropic-sdk-go`) to `go.mod`; new env var `ANTHROPIC_API_KEY` (document in CLAUDE.md's env list). New model file `src/pkg/models/ai_grading.go`.
- Grade synchronously in the submit handler (segments are 5–10 words; a single small-model call is fast — use `claude-haiku-4-5`). Prompt: reference Hebrew + student Hebrew + grammar point name/description → return a 0–100 accuracy score and one-sentence feedback as JSON. Store both on `produce_submissions`.
- **Failure mode matters:** if the AI call fails, store the submission with `ai_score NULL` and return success to the student (do not block progression on a grading outage); the Score page treats ungraded as pending.
- Scoring integration: `GetScoresData` (`src/apis/handlers/stories-score.go`) gains `produce_accuracy` (mean of the two `ai_score`s) folded into `overall_accuracy`.

## Phase 5 — Recall (2:15)

Entirely net-new. New component `StoriesRecall.tsx`, route `stories/:id/recall`.

Behavior:

1. Play the full story audio with **no text shown** (reuse `useAudioPlayer` with the text panel hidden; a simple progress indicator is fine). No skipping.
2. After playback: `GET /api/stories/:id/recall` returns the 5 sentences from `recall_sentences` (Hebrew text + signed image URL each), **shuffled server-side** with `sequence_order` withheld.
3. Drag-and-drop ordering UI (`@dnd-kit/sortable`, F4): student arranges the 5 cards into story order and submits.
4. `POST /api/stories/:id/check-recall` with the ordered sentence IDs; backend compares to `sequence_order` and returns per-position correctness. Record attempts in `recall_correct_answers` / `recall_incorrect_answers` (append-only pattern) for scoring. Decide: single-attempt scored, or retry-until-correct with attempts counted (recommend the latter — it matches the vocab/grammar retry model that `CalculateScoreWithRetriesAllowed` already handles).

## Score page changes

`frontend/app/components/StoriesScore.tsx` (472 lines) + `GetScoresData` in `src/apis/handlers/stories-score.go`:

- Replace the Vocabulary/Grammar accuracy cards with **Identify**, **Produce** (AI-graded), and **Recall** accuracy cards; extend `ScoreData` (Go struct at `stories-score.go:15`, TS interface at `StoriesScore.tsx:7-21`) accordingly.
- Time breakdown becomes five phases (watch/identify/translate/produce/recall) — see F1 route-matching changes.
- `MissingActivity` incomplete-detection (`no_data` / `insufficient_time`) must cover the three new phases: identify → any identify answer rows, produce → both submissions present, recall → a recall attempt present.

## Hardest parts — extra care required

Ranked by risk. See also `developer_review.md` items marked ⚡S26, which interact with this work.

1. ~~**Schema evolution without a migration system.**~~ **Resolved by T1/T3.** Version-controlled goose migrations in `src/pkg/database/migrations/` are applied on startup and are SQLC's schema source, so an ALTER now applies everywhere and codegen can't drift from the deployed schema. Add a numbered migration for each further change; don't reintroduce a standalone `schema.sql`.
2. **The Translate-phase audio state machine.** Continuous playback + click-to-request (valid only on current/next line) + 2s predict pause + 5s hold + 4–7 quota + 3-consecutive cap + end-of-story restart with fast-forward and auto-skip of translated lines. This is the most intricate interaction logic in the app, layered onto `useAudioPlayer`'s prefetched `HTMLAudioElement` sequencing — full of race conditions (user clicks during a transition, line ends during the 2s pause, restart mid-hold). Model it as an explicit state machine (`useReducer` with named states like `playing / predicting / revealing / restarting`), not ad-hoc `useState` flags, and write unit tests for the transition table before wiring up audio.
3. **AI grading of Hebrew.** Net-new external dependency: API key management, cost control (rate-limit the submit endpoint — developer_review.md #2 must be fixed first), latency of a synchronous call in the request path, graceful degradation when the API is down (fail open, `ai_score NULL`), and prompt quality — grading introductory-student Hebrew against a reference needs tolerance for spelling/nikkud variation and must not crush beginners. Budget real iteration time on the prompt with sample student answers.
4. **Content authoring becomes the bottleneck.** Every story now needs a video, 5 target words with audio + images, 2 produce segments with references and an explanation, and 5 recall sentences with images and ordering. The admin-editor surface area (F2/F3) is larger than any single student phase, and stories missing the new content must degrade sanely (navigation should skip phases whose content is absent, not 500).
5. **`useAudioPlayer` extensions ripple across three phases.** Identify, Translate, and Recall all depend on the extended hook. A regression there breaks the whole flow — extend behind new options with the existing behavior as the default, and test the hook in isolation (the repo already has vitest + happy-dom set up).
6. **Scoring/completeness rework.** `GetScoresData` must handle mixed generations of data: students mid-story when the flow changes, old stories without new content, ungraded produce submissions. Define explicitly what "complete" means per phase before touching `stories-score.go`.

## Suggested implementation order

1. **F1** navigation + page types + empty placeholder pages (flow walkable end-to-end early).
2. **F3/F2 schema + admin CRUD** (target vocab, produce segments, recall sentences, image storage) — content authoring unblocks everything else.
3. **Translate** rework (no backend changes; exercises the audio-player extensions needed by Identify/Recall).
4. **Identify** (builds on audio extensions + target vocab + images).
5. **Recall** (dnd-kit + recall content).
6. **Produce** + AI grading.
7. **Score page** last, once all answer tables exist.

After each step, run the checks from CLAUDE.md (backend: `gofmt`/`go vet`/`go test`/`go build`; frontend: `npm run format`/`lint`/`typecheck`/`build`) and `sqlc generate` after any query change.

## Content-authoring requirements per story (non-code)

- Two-sentence plot summary; ~90s Ken Burns video (external production) → `video_url`.
- 5 target vocab words, each appearing ≥2× in the text, each with word audio + one image.
- Per-line English translations (existing `line_translations` tooling).
- Two Produce segments (5–10 English words each, featuring the grammar point) + reference Hebrew + contrastive explanation text.
- 5 Recall sentences (one per target word, major narrative events) with images and correct order.
