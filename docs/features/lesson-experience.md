---
title: "Lesson Experience"
description: "Current user-facing lesson catalog, combined lesson page, board playback, and engine analysis behavior."
type: feature
status: implemented
tags: ["lessons", "analysis", "frontend"]
source_paths:
  - "apps/api/internal/httpapi/server.go"
  - "apps/api/internal/lessons"
  - "scripts/build_combined_lesson.py"
  - "apps/web/src"
related:
  - "../specs/planning/combined-lesson-page.md"
  - "../_sync.md"
---

# Lesson Experience

## Meta

- Trang thai: implemented
- Pham vi: Lesson catalog, combined lesson playback, choice feedback, and engine analysis API behavior
- Nguon code: `apps/api/internal/httpapi/server.go`, `apps/api/internal/lessons`, `scripts/build_combined_lesson.py`, `apps/web/src`
- Tuan thu: Khong ap dung
- Links: [Combined Lesson Page Plan](../specs/planning/combined-lesson-page.md), [Docs Sync State](../_sync.md)

## Behavior

The app serves an interactive Co Tuong lesson catalog from the Go API and renders it in the Vue frontend. Lessons are grouped by category, sorted in a stable learning order, and opened from persistent `/lessons/{id}` routes. Selecting a lesson expands its category and updates browser history so back/forward navigation can restore the active lesson.

Each lesson contains a compact payload with UUID-based lesson, line, and move identifiers. The API summary response includes `id`, `title`, `category`, and `difficulty`; lesson detail responses include optional lesson-level `initialFen`, optional line-level `initialFen`, playable lines, and a choice prompt. Deprecated fields such as `slug`, `summary`, `tags`, `principles`, move `notation`, move `title`, and move `evaluation` are not part of the current lesson payload contract.

The frontend replays lesson moves on a 9x10 board, shows the current move path in the graph, and displays choice options when the active ply reaches the lesson prompt. A shared principles panel provides static learning guidance from `apps/web/src/content/principles.ts`.

## Combined Lesson Page

`scripts/build_combined_lesson.py` builds `apps/api/data/lessons/combined-lesson.json` from the production catalog. The script uses stable UUID v5 identifiers so repeated runs produce stable output, flattens every source lesson line into one combined lesson line, and stores source lesson `initialFen` on the generated line when the original lesson starts from a custom position.

`GET /api/combined-lesson` serves a lightweight overview of the built artifact without adding it to the normal lesson catalog. The overview includes line metadata, move counts, piece counts, and a phase label derived from the effective line starting position. Combined lessons without a lesson-level `initialFen` expose the standard Xiangqi starting FEN so the opening graph has an explicit root state before any move is played. Lines with 28 or more pieces are classified as opening, lines with 14-27 pieces as middlegame, and lines with fewer than 14 pieces as endgame. The overview does not include the full move list. `POST /api/combined-lesson/next-steps` accepts the active phase, effective initial FEN, current move prefix, offset, and limit, then returns one bounded move window containing only the lines that belong to the active graph node. `GET /api/combined-lesson/moves` remains available for phase-wide move windows, and `GET /api/combined-lesson/lines/{id}/moves` remains available for targeted line windows.

The `/combined` frontend page renders a focused workspace with only the board, phase-split move trees, engine panel, and compact previous/next step controls. It does not render the catalog sidebar, lesson topbar, choice prompt, or principles panel. Opening, middlegame, and endgame trees live in separate expandable panels instead of one shared scroll area. The page hydrates the active node with a single next-steps request and keeps the full combined move corpus on the backend. Each phase tree uses `d3-hierarchy` to lay out independent lesson lines under one hidden root, then adds a shared start-position node for every group of lines with the same effective `initialFen`. Loaded plies are folded into a trie by move prefix, so later matching steps stay merged and branch points appear as selectable child move choices instead of duplicated per-line paths.

## Engine Analysis

`POST /api/analyze` delegates position analysis to a configured UCI engine. Invalid requests return `400`, missing engine configuration returns `503`, engine timeouts return `504`, and other engine failures return `502`.

Timeouts are mapped through `analysis.ErrEngineTimeout` so the public JSON error message does not expose low-level `context deadline exceeded` text. The response still uses the existing `{ "error": "..." }` shape expected by the frontend.

## Validation Coverage

Backend tests cover lesson repository loading, UUID and payload-shape constraints, legal replay validation, combined lesson endpoint behavior, UCI timeout classification, and HTTP timeout response mapping. The web build validates the current Vue and TypeScript surface.
