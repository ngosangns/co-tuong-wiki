# Docs Sync State

## Meta

- Synced commit: `HEAD`
- Synced at: `2026-06-07T13:35:00Z`
- Scope: UI redesign plan (`plans/ui-redesign.md`), 12 quick wins
  (typography, color, brand, motion), Card primitive, bento grid,
  Editorial typography pass (Noto Serif Vietnamese). See
  `plans/ui-redesign.md` Phase A→E completion.
- Status: synced

## Notes

- `plans/ui-redesign.md` captures the design-taste-frontend audit and
  the Editorial × Dark-tech redesign plan (DESIGN VARIANCE 3→6,
  MOTION_INTENSITY 4→3, VISUAL_DENSITY 7→5).
- Three dials codified in the style system:
  - Red = primary/active/danger, teal = secondary/data.
  - Shape: 6/10/16/pill (no stray rounded-2xl).
  - Banned: uppercase on Vietnamese, italic body, <11px text,
    the four-times-repeated entry fade-up.
- Editorial pass: Noto Serif Vietnamese self-hosted with `font-display:
  swap` for principle-group h3, board-move-comment, wiki-article
  h2 + p, and a pull-quote glyph on the move comment.
- MoveGraph is now theme-aware via `--color-graph-*` CSS variables
  resolved at render time through `getComputedStyle`.
