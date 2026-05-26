---
title: "Lesson Experience"
description: "Current user-facing lesson catalog, board playback, and engine analysis behavior."
type: feature
status: implemented
tags: ["lessons", "analysis", "frontend"]
source_paths:
  - "apps/api/internal/httpapi/server.go"
  - "apps/api/internal/lessons"
  - "apps/web/src"
related:
  - "../_sync.md"
---

# Lesson Experience

## Meta

- Trang thai: implemented
- Pham vi: Lesson catalog, lesson playback, choice feedback, and engine analysis API behavior
- Nguon code: `apps/api/internal/httpapi/server.go`, `apps/api/internal/lessons`, `apps/web/src`
- Tuan thu: Khong ap dung
- Links: [Docs Sync State](../_sync.md)

## Behavior

The app serves an interactive Co Tuong lesson catalog from the Go API and renders it in the Vue frontend. Lessons are grouped by category, sorted in a stable learning order, and opened from persistent `/lessons/{id}` routes. Selecting a lesson expands its category and updates browser history so back/forward navigation can restore the active lesson.

Each lesson contains a compact payload with UUID-based lesson, line, and move identifiers. The API summary response includes `id`, `title`, `category`, and `difficulty`; lesson detail responses include optional `initialFen`, playable lines, and a choice prompt. Deprecated fields such as `slug`, `summary`, `tags`, `principles`, move `notation`, move `title`, and move `evaluation` are not part of the current lesson payload contract.

The frontend replays lesson moves on a 9x10 board, shows the current move path in the graph, and displays choice options when the active ply reaches the lesson prompt. A shared principles panel provides static learning guidance from `apps/web/src/content/principles.ts`.

## Engine Analysis

`POST /api/analyze` delegates position analysis to a configured UCI engine. Invalid requests return `400`, missing engine configuration returns `503`, engine timeouts return `504`, and other engine failures return `502`.

Timeouts are mapped through `analysis.ErrEngineTimeout` so the public JSON error message does not expose low-level `context deadline exceeded` text. The response still uses the existing `{ "error": "..." }` shape expected by the frontend.

## Validation Coverage

Backend tests cover lesson repository loading, UUID and payload-shape constraints, legal replay validation, UCI timeout classification, and HTTP timeout response mapping. The web build validates the current Vue and TypeScript surface.
