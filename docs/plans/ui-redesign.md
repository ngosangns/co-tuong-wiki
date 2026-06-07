# Plan: Toàn Bộ UI Redesign — Editorial × Dark-tech

## Meta

- **Phạm vi**: Rebuild toàn bộ UI theo audit design-taste skill
- **Stack giữ nguyên**: Vue 3 + TS + Tailwind v4 + d3-hierarchy
- **Theme**: Dark by default, light optional
- **Reference**: design-taste-frontend skill (anti-slop), Section 0–9

## 0. Design Read (một câu)

> Reading this as: an editorial/data hybrid for Vietnamese Xiangqi learners (Nhập môn → Cao cấp) who want both the cultural depth of traditional chess instruction and the analytical power of a Stockfish-grade engine, leaning toward **Editorial typography × Dark-tech UI** (think Notion meets Lichess, with a Vietnamese woodgrain warmth).

## 1. Three Dials

| Dial | Hiện tại | Mục tiêu | Lý do |
|------|----------|----------|-------|
| **DESIGN VARIANCE** | 3/10 (rigid 3-col) | **6/10** (bento + editorial mix) | Bento cho workspace, magazine cho article, full-bleed cho graph |
| **MOTION_INTENSITY** | 4/10 (entry fade-up lặp 4×) | **3/10** (chỉ informative motion) | Tool dùng 100×/ngày; motion phải motivate, không phải decorate |
| **VISUAL_DENSITY** | 7/10 (board + 7 panels cùng viewport) | **5/10** (giảm ~30%) | Audience bao gồm learners, không chỉ engine users |

## 2. Aesthetic Direction

**Editorial × Dark-tech hybrid.** Hai reference families từ skill (Section 2.B):

- **Editorial**: principle lists, move comments, lesson articles → serif Vietnamese cho body, sans cho UI, mono cho notation
- **Dark-tech**: dark warm palette, mono for engine output (FEN, depth, nps), single accent for primary actions

**Banned**: Glassmorphism, Bento đa-tile (content quá sequential), Brutalist (audience tin vào "uy tín giáo dục"), Playful illustrations (audience nghiêm túc).

## 3. Slop Đã Phát Hiện (từ audit)

| # | Slop | File:line | Mức |
|---|------|-----------|------|
| 1 | `text-transform: uppercase` trên tiếng Việt | style.css:246, 1603, 682, 2193 | 🔴 Quick |
| 2 | MoveGraph hardcoded hex (bypass theme) | MoveGraph.vue:325-330, 383 | 🔴 Quick |
| 3 | LibraryPanel search break focus-visible | LibraryPanel.vue:79 | 🔴 Quick |
| 4 | Brand mark 象 (Tượng) → niche piece | LibraryPanel.vue:65 | 🔴 Quick |
| 5 | `Home` icon trong breadcrumb (không chess) | MoveBreadcrumb.vue:90 | 🔴 Quick |
| 6 | "ply" jargon → "nước" | EvaluationPanel.vue:166 | 🔴 Quick |
| 7 | Eyebrow 10px < 11px WCAG floor | style.css:1716, 680, 1599 | 🔴 Quick |
| 8 | Preview piece opacity 0.48 (quá subtle) | style.css:1230 | 🔴 Quick |
| 9 | Principle list italic + low-contrast | style.css:1664 | 🔴 Quick |
| 10 | Sidebar overlay pure-black 45% (warm palette clash) | style.css:184-185 | 🔴 Quick |
| 11 | Mixed active-color (red AND teal cho "active") | style.css:366, 432, 2149, 2315 | 🟡 Medium |
| 12 | Card recipe lặp 8× cùng border/radius/padding | nhiều file | 🟡 Medium |
| 13 | `v-motion` entry fade-up lặp 4× | App.vue:143, 217, 162, CombinedLessonPage.vue:215 | 🟡 Medium |
| 14 | Bento opportunity missed (workspace 3-col cứng) | App.vue, CombinedLessonPage.vue | 🔵 Big |
| 15 | Typography chưa có serif Vietnamese (chỉ Geist) | index.html:14 | 🔵 Big |
| 16 | Engine "ply" jargon nhiều chỗ | EvaluationPanel.vue, EvalChart.vue | 🟡 Medium |
| 17 | MiniMap không có loading state | MoveMinimap.vue | 🟡 Medium |
| 18 | Empty states generic ("Chưa có dữ liệu") | EvalChart.vue, OpeningSuggestions.vue | 🟡 Medium |
| 19 | 3 column grid ép mọi thứ vào khung giống nhau | style.css:68 | 🔵 Big |
| 20 | Graph hardcoded color → theme inconsist | MoveGraph.vue | 🟡 Medium |

## 4. Nguyên Tắc Khóa

### 4.1 Color Lock
- **Red (accent)**: primary CTA, active state, danger, brand mark. Một mình nó là "do this".
- **Teal (primary)**: secondary action, graph highlight, opening book suggestions, success indicator. "Data/state" color.
- **Banned**: bất kỳ CTA nào mà không phải `accent` (red); tất cả error/danger/active/primary phải cùng tông.
- **Palette rotate** (Phase 8 anti-slop): dùng warm wood + cream + ink (đã có), KHÔNG thêm beige+brass mới.

### 4.2 Typography Lock
- **Sans (Geist)**: UI, headers, buttons
- **Mono (Geist Mono)**: FEN, depth, nps, notation, ply numbers
- **Serif (Noto Serif Vietnamese hoặc tương đương)**: principle list body, board-move-comment, lesson title (nếu thấy editorial)
- **Banned**:
  - `text-transform: uppercase` trên tiếng Việt
  - `font-style: italic` cho body text (chỉ dùng cho citation nhỏ)
  - Font size < 11px

### 4.3 Shape Lock
- `--radius-sm: 6px` (chips, eyebrow)
- `--radius-md: 10px` (cards, buttons — bump từ 8 → 10 cho thoáng hơn)
- `--radius-lg: 16px` (panels chính như board, eval-chart)
- `--radius-pill: 999px` (chess pieces, avatar)
- Banned: stray `rounded-2xl` / `rounded-3xl`

### 4.4 Motion Lock
- **Reduced default**: chỉ informative motion (piece slide, hover translate-y, focus ring)
- **No entry fade-up**: 4× lặp hiện tại → bỏ hết, page render direct
- **Piece move**: 220ms slide giữ (đã communicate "moved from here to there")
- **Sidebar**: fade overlay + slide panel
- **Tab change**: cross-fade 120ms
- **Honor `prefers-reduced-motion`**: bắt buộc

### 4.5 Layout Lock
- **Lesson view (desktop)**: bento grid thay vì 3-col đều
  - Board 60% width (hero piece)
  - Engine eval 40% (bên phải, cao = board)
  - Opening suggestions 30% (dưới eval)
  - Breadcrumb full-width
  - Graph + Minimap: split 70/30 ở row dưới
  - Principles: cột cuối 30%
- **Combined view**: 2-col (board | analysis) + tab cho graph ở dưới
- **Mobile**: 1 cột + bottom dock + tabs (giữ nguyên)

## 5. Phases (ưu tiên theo impact/cost)

### Phase A — Quick Wins (1-2 ngày, ~12 fixes)
Mục tiêu: loại bỏ tất cả quick slop, không thay đổi kiến trúc.

**A.1 Typography**
- [ ] Loại bỏ `text-transform: uppercase` khỏi tất cả CSS rules có tiếng Việt
- [ ] Bump eyebrow 10px → 11px (4 chỗ)
- [ ] Import Noto Serif Vietnamese (hoặc font tương đương hỗ trợ diacritics) cho editorial
- [ ] Thêm font-feature-settings tabular-nums cho notation/FEN/depth

**A.2 Color**
- [ ] MoveGraph: thay hardcoded hex bằng CSS variables (`--color-graph-active`, etc.)
- [ ] Sidebar overlay: pure-black 45% → warm `rgb(20 18 14 / 55%)` + 10px blur
- [ ] Preview piece opacity 0.48 → 0.7
- [ ] Active color semantics: thống nhất red = active/primary, teal = data
- [ ] EvaluationPanel: thay "Engine" bằng "Động cơ"

**A.3 Brand & Icons**
- [ ] Brand mark: 象 → 帥 (general) hoặc 棋 (chess) — chọn 1
- [ ] MoveBreadcrumb: Home icon → Flag icon
- [ ] Search input: bỏ `focus-visible:ring-0` override
- [ ] Thay "ply" → "nước" (EvaluationPanel, EvalChart, MoveMinimap)
- [ ] Principle list: bỏ italic, đổi sang weight 600 + color text (không muted)

**A.4 Motion cleanup**
- [ ] Bỏ 4× `v-motion` entry fade-up
- [ ] Thêm `@media (prefers-reduced-motion: reduce)` block to disable piece transition
- [ ] Sidebar overlay: thêm transition fade

### Phase B — Card System Refactor (2-3 ngày)
Mục tiêu: phân biệt card hierarchy, không phải mọi thứ đều giống nhau.

**B.1 Card variants**
- [ ] Tạo `<Card variant="elevated|flat|outlined">` component
- [ ] Variant `elevated`: board stage, eval chart (có shadow + radius-lg)
- [ ] Variant `flat`: choice box, feedback (chỉ border, no shadow)
- [ ] Variant `outlined`: secondary panels (principles, opening suggestions)

**B.2 Apply variants**
- [ ] Refactor 8+ components dùng card recipe sang `<Card>` 
- [ ] Verify visual hierarchy: board nổi bật nhất → eval → opening → principles

### Phase C — Bento Workspace (1 tuần)
Mục tiêu: thay 3-col cứng bằng bento có rhythm.

**C.1 Desktop workspace**
- [ ] Thay `grid-template-columns: ... 3-col` bằng CSS Grid areas:
  ```
  ┌──────────────┬──────────────┐
  │              │  Eval Chart  │
  │     Board    ├──────────────┤
  │              │  Opening     │
  ├──────────────┴──────────────┤
  │        Breadcrumb           │
  ├──────────────┬──────────────┤
  │   Minimap    │   Graph      │
  ├──────────────┴──────────────┤
  │       Principles            │
  └─────────────────────────────┘
  ```
- [ ] Board area 60% width, eval+opening stacked 40%
- [ ] Breadcrumb full-width row
- [ ] Graph + minimap: 70/30 split
- [ ] Principles: separate panel với editorial typography

**C.2 Mobile**
- [ ] Giữ tab system, nhưng add bottom dock với 3 nút (Bàn cờ / Biến / Lý thuyết) thay vì 4 tabs
- [ ] Eval + opening book combine thành 1 "Phân tích" tab

### Phase D — Editorial Pass (3-4 ngày)
Mục tiêu: principle list + lesson article feel như publication, không phải tool.

**D.1 Serif Vietnamese**
- [ ] Import `Noto Serif Vietnamese` (self-host, font-display: swap)
- [ ] Apply cho `.principles-list`, `.board-move-comment`, `.wiki-article h2`
- [ ] Adjust line-height cho serif (1.6-1.7 thay vì 1.45)

**D.2 Reading column**
- [ ] Principle list: single column `max-width: 65ch`, indent blockquote style
- [ ] Move comment: leading-relaxed, serif body, monospace cho notation inline
- [ ] Lesson article (wiki): serif title + serif body, sans cho UI elements

**D.3 Marginalia (optional)**
- [ ] Pull-quote callouts cho principle items quan trọng
- [ ] Số thứ tự (1, 2, 3...) ở đầu mỗi principle (editorial numbering)

### Phase E — Polish & Validation (2-3 ngày)
Mục tiêu: validate changes, document, ship.

**E.1 Visual regression**
- [ ] Screenshot so sánh: light/dark, mobile/desktop, lesson view/combined view
- [ ] Run Lighthouse, fix bất kỳ regression nào

**E.2 Documentation**
- [ ] Update `docs/features/lesson-experience.md` với design language
- [ ] Cập nhật `docs/_index.md`
- [ ] Sync `docs/_sync.md`
- [ ] Cập nhật plan file này với status

**E.3 Test**
- [ ] Run all 40 vitest tests
- [ ] Run all Go tests
- [ ] Run lint + format
- [ ] Manual: tab qua các lesson, so sánh engine best với played

## 6. Out of Scope

- Real product photography (chess set lifestyle shots) — would require image gen tool or stock
- E2E test framework (Playwright) — high cost, defer sang phase riêng
- Backend changes — UI redesign không đụng API contract
- New features (so sánh mode đã done, classification đã done) — focus là aesthetic

## 7. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Bento grid phá vỡ layout hiện tại users đã quen | Medium | A/B test desktop, giữ mobile intact |
| Serif Vietnamese font load ảnh hưởng LCP | Medium | `font-display: swap`, preload font, subset Vietnamese |
| Card variant thay đổi visual identity | Low | A/B test, gather feedback |
| Active color semantics thay đổi muscle memory | Low | Document trong docs |
| Bento + editorial conflict với existing data tables | Low | Principles là duy nhất data-heavy, editorial chỉ apply ở đó |

## 8. Acceptance Criteria

- [ ] DESIGN VARIANCE = 6/10 (đo bằng layout patterns count: ≥ 3 families/page)
- [ ] MOTION_INTENSITY = 3/10 (no entry fade-up, chỉ informative motion)
- [ ] VISUAL_DENSITY = 5/10 (eval + opening gộp thành 1 tab mobile)
- [ ] `text-transform: uppercase` count trên tiếng Việt = 0
- [ ] Hardcoded hex colors in components = 0 (chỉ trong theme.css)
- [ ] Eyebrow < 11px count = 0
- [ ] Italic body text count = 0
- [ ] Card variants: ≤ 2 visual recipes per page
- [ ] Active color = 1 semantic (red = primary, teal = data)
- [ ] All 40 vitest tests pass
- [ ] All Go tests pass
- [ ] ESLint 0 errors
- [ ] Lighthouse LCP < 2.5s
- [ ] Lighthouse a11y score ≥ 95

## 9. Tiêu Chí Thành Công

- [ ] User feedback: "Trông như một ấn phẩm, không phải tool"
- [ ] Principle list dễ đọc hơn (eyebrow tracking, no italic)
- [ ] Move graph colors adapt light/dark
- [ ] Board nổi bật hơn trong workspace (bento)
- [ ] Typography hierarchy rõ ràng (serif body, sans UI, mono notation)
- [ ] Mobile gọn hơn (1 tab thay vì 2 cho analysis)

## 10. Open Questions

1. **Brand mark**: 帥 (general — iconic) hay 棋 (chess — semantic)? Recommend 帥.
2. **Serif font**: Noto Serif Vietnamese (Google) hay self-host alternative? Recommend Noto Serif Vietnamese.
3. **Layout**: Bento từ đầu hay incremental (chỉ thay 1 panel đầu tiên)? Recommend incremental.
4. **Editorial scope**: chỉ principle list, hay cả lesson article? Recommend principle list + board-move-comment trước.
5. **Card system**: tạo `<Card>` primitive hay inline class? Recommend primitive.
