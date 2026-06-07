---
title: "Lesson Experience"
description: "Current user-facing lesson catalog, combined lesson page, board playback, and engine analysis behavior."
type: feature
status: implemented
tags: ["lessons", "analysis", "frontend"]
source_paths:
  - "apps/api/internal/httpapi/server.go"
  - "apps/api/internal/cacheutil"
  - "apps/api/internal/lessons"
  - "scripts/build_combined_lesson.py"
  - "apps/web/src"
related:
  - "../specs/planning/combined-lesson-page.md"
  - "../specs/planning/performance-optimization.md"
  - "../specs/planning/smart-cache-strategy.md"
  - "../plans/ui-redesign.md"
  - "../_sync.md"
---

# Lesson Experience

## Meta

- Trang thai: implemented
- Pham vi: Lesson catalog, combined lesson playback, choice feedback, and engine analysis API behavior
- Nguon code: `apps/api/internal/httpapi/server.go`, `apps/api/internal/cacheutil`, `apps/api/internal/lessons`, `scripts/build_combined_lesson.py`, `apps/web/src`
- Tuan thu: Khong ap dung
- Links: [Combined Lesson Page Plan](../specs/planning/combined-lesson-page.md), [Performance Optimization Plan](../specs/planning/performance-optimization.md), [Smart Cache Strategy](../specs/planning/smart-cache-strategy.md), [UI Redesign Plan](../plans/ui-redesign.md), [Docs Sync State](../_sync.md)

## Behavior

The app serves an interactive Co Tuong lesson catalog from the Go API and renders it in the Vue frontend. Lessons are grouped by category, sorted in a stable learning order, and opened from persistent `/lessons/{id}` routes. Selecting a lesson expands its category and updates browser history so back/forward navigation can restore the active lesson.

Each lesson contains a compact payload with UUID-based lesson, line, and move identifiers. The API summary response includes `id`, `title`, `category`, and `difficulty`; lesson detail responses include optional lesson-level `initialFen`, optional line-level `initialFen`, playable lines, and a choice prompt. Deprecated fields such as `slug`, `summary`, `tags`, `principles`, move `notation`, move `title`, and move `evaluation` are not part of the current lesson payload contract.

Stable read endpoints expose cache headers derived from the backing JSON artifact. `GET /api/categories`, `GET /api/lessons`, `GET /api/lessons/{id}`, and combined lesson read endpoints return `ETag`, `Last-Modified`, `X-Data-Version`, and `X-Cache`; matching `If-None-Match` requests return `304` without a response body. The frontend API client keeps memory cache entries, stores catalog/lesson/combined overview data in `sessionStorage`, deduplicates matching in-flight requests, and reuses cached bodies when the API returns `304`.

The frontend replays lesson moves on a 9x10 board, shows the current move path in the graph, and displays choice options when the active ply reaches the lesson prompt. A compact engine panel appears below the board and directly above the active move comment, with the previous/next controls below the comment. Graph details keep only node metadata and the selected line action. The board also renders translucent one-ply previews for every immediate child move under the active graph node, including an arrow from the current piece square to the preview destination, so branch options and direction are visible before the user advances. Clicking a preview selects the matching child graph node and advances the board to that position. A shared principles panel provides static learning guidance from `apps/web/src/content/principles.ts`.

## Combined Lesson Page

`scripts/build_combined_lesson.py` builds `apps/api/data/lessons/combined-lesson.json` from the production catalog. The script uses stable UUID v5 identifiers so repeated runs produce stable output, skips source lines without playable moves, flattens every playable source lesson line into one combined lesson line, and stores source lesson `initialFen` on the generated line when the original lesson starts from a custom position.

`GET /api/combined-lesson` serves a lightweight overview of the built artifact without adding it to the normal lesson catalog. The overview includes line metadata, move counts, piece counts, and a phase label derived from the effective line starting position. Combined lessons without a lesson-level `initialFen` expose the standard Xiangqi starting FEN so the opening graph has an explicit root state before any move is played. Lines with 28 or more pieces are classified as opening, lines with 14-27 pieces as middlegame, and lines with fewer than 14 pieces as endgame. The overview does not include the full move list. `POST /api/combined-lesson/next-steps` accepts the active phase, effective initial FEN, current move prefix, offset, and limit, then returns one bounded move window containing only the lines that belong to the active graph node. The backend caches encoded combined overview and move-window responses in memory, keyed by combined artifact version and normalized request parameters. `GET /api/combined-lesson/moves` remains available for phase-wide move windows, and `GET /api/combined-lesson/lines/{id}/moves` remains available for targeted line windows.

The `/combined` frontend page renders a focused three-column workspace: the board in the first column, previous/next controls above compact engine analysis and active move comment in the second column, and the phase-split move tree in the third column. It does not render the catalog sidebar, lesson topbar, choice prompt, or principles panel. Opening, middlegame, and endgame trees live in separate expandable panels instead of one shared scroll area. The page hydrates the active node with a single next-steps request and keeps the full combined move corpus on the backend. The board preview uses the hydrated active-node children, then updates when lazy next-step data arrives; selecting a preview follows the same graph-node navigation path and loads the next lazy window for the new active node. Each phase tree uses `d3-hierarchy` to lay out independent lesson lines under one hidden root, then adds a shared start-position node for every group of lines with the same effective `initialFen`. Loaded plies are folded into a trie by move prefix, so later matching steps stay merged and branch points appear as selectable child move choices instead of duplicated per-line paths. The graph renderer keeps topology rendering separate from active-node state updates, so moving through already-loaded nodes updates SVG state without rebuilding the whole graph.

## Engine Analysis

`POST /api/analyze` delegates position analysis to a configured UCI engine. The analysis service keeps a shared UCI process warm for matching engine configuration, serializes searches through that process, coalesces concurrent requests for the same key, and caches recent responses for 10 minutes by FEN, side to move, next move, and search settings. Successful HTTP responses include `X-Cache` with `miss`, `hit`, or `coalesced`. Invalid requests return `400`, missing engine configuration returns `503`, engine timeouts return `504`, and other engine failures return `502`.

Timeouts are mapped through `analysis.ErrEngineTimeout` so the public JSON error message does not expose low-level `context deadline exceeded` text. The response still uses the existing `{ "error": "..." }` shape expected by the frontend.

## Validation Coverage

Backend tests cover lesson repository loading, UUID and payload-shape constraints, legal replay validation, combined lesson endpoint behavior, conditional cache `304` behavior, UCI timeout classification, analysis coalescing, and HTTP timeout response mapping. The web build validates the current Vue and TypeScript surface.

## Design Language

The lesson view follows the Editorial × Dark-tech language documented in
`../plans/ui-redesign.md`. The board sits in a two-column bento: the
playable surface on the left, engine analysis + active comment on the
right, with the move graph and opening suggestions in a second bento
band beneath. Reading content (principles, move comments, wiki body)
uses Noto Serif Vietnamese for editorial weight; chrome (controls,
labels, eval deltas) keeps Geist. Red marks primary action or threat;
teal marks engine data. The brand mark is 帥. See the plan for the
locked three-dial contract and the ban list.
