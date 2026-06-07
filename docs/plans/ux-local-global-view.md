# Plan: Local + Global View, AI Phân Tích, Mở Rộng Engine

## Meta

- Phạm vi: Cải tiến UX (nhìn cục bộ + nhìn toàn cục), tầng phân tích AI, mở rộng API & data layer, kiến trúc dọn dẹp.
- Stack hiện tại giữ nguyên: Vue 3 + TS + Tailwind 4 + d3-hierarchy (frontend); Go `net/http` (backend); Pikafish UCI (engine chính).
- Thêm vào: Fairy-Stockfish UCI làm engine phụ cho puzzle/so sánh; bộ lưu trữ opening book + move classification cache.
- Bám sát: `docs/features/lesson-experience.md`, `docs/plans/redesign-responsive-ui.md` (mobile/responsive đã có kế hoạch riêng).
- Trạng thái: chờ duyệt.

## 1. Đánh giá hiện trạng

### 1.1 Kiến trúc

| Tầng | Tốt | Cần cải |
|------|-----|---------|
| Backend Go | `httpapi`, `analysis`, `lessons`, `cacheutil` được tách rõ, test pass, `next-steps` dùng trie collapse giúp dữ liệu 6k lessons còn mượt. | `internal/analysis` chỉ hỗ trợ `uci`; enum `wukong` đã có trong `engine/types.ts` nhưng backend chưa có. Thiếu trait `Engine` interface để plug thêm engine mới. |
| Frontend Vue | `useLessonPlayer` + `useMoveEvaluation` tách logic khỏi view; `XiangqiBoard` thuần presentational; `MoveGraph` dùng d3-hierarchy với normalize tách `__tree_root__` synthetic gọn. | `App.vue` đang phình ~400 dòng, ôm cả 3 layout desktop/tablet/mobile. `MoveGraph` ăn cả trie build + layout + selection trong 1 file. `CombinedLessonPage` duplicate rất nhiều logic với `App.vue`. |
| Cache & ETag | Backend có `X-Data-Version` + ETag + byte cache; frontend có memory + sessionStorage + in-flight dedup. | Cache chỉ ở mức HTTP, chưa có per-position engine eval cache phía client. |
| Error model | Timeout engine map qua `ErrEngineTimeout` rồi mới trả JSON, không leak `context deadline exceeded`. | Chưa có global error boundary Vue; lỗi engine hiện chỉ hiện trong `EvaluationPanel`. |
| Tests | Backend test phủ cache, UCI timeout, combined endpoint; frontend chỉ có `vue-tsc` build-time check. | Không có e2e cho flow board → graph, không có test cho `useMoveEvaluation`, `useLessonPlayer`, FEN parse. |

### 1.2 Data

- 6,261 lessons / 6,281 lines / 35,183 moves / 11 categories / 4 difficulty levels / 811 lessons có `choice`.
- `lessons.json` ~21MB, `combined-lesson.json` ~16MB, đã pre-built.
- 5,844/6,261 lessons có `initialFen` riêng (đa số là tàn cuộc/trung cuộc bắt đầu từ FEN tùy biến).
- `moves` chỉ có `{side, from, to, comment}` — **chưa có** `evaluation`, `classification`, `openingName`, `themeTags`.
- Chưa có opening book (WXF, ePD, ECO mapping) hay dữ liệu tần suất nước đi trong ván đấu thật.

### 1.3 Business logic

- `analysis` đã có: warm UCI process, serialize search, coalesce in-flight theo key, cache 10 phút theo `(engine, depth, time, FEN, nextMove)`.
- `useMoveEvaluation` debounce 180ms + `AbortController` tránh gọi trùng; cache phía client 10 phút.
- `useLessonPlayer` chỉ xử lý single-line replay; chưa có concept "current branch in tree" — chỉ biết `activeLine` + `activeMoveIndex`.
- `MoveGraph` trie-build merge các line có chung prefix; state machine đơn giản `start/active/past/future`.
- `CombinedLessonPage` làm mới thêm: phase split (Khai cuộc/Trung cuộc/Tàn cuộc theo `pieceCount`), `next-steps` POST active node, lazy hydration.

### 1.4 UX

**Điểm mạnh:**
- Board 9x10 render SVG-friendly với file labels, palace marker, river border, preview arrows trỏ từ nước hiện tại sang preview.
- `nextMovePreviews` highlight các nước con của node hiện tại ngay trên bàn cờ (UX rất tốt cho branch awareness).
- Combined page tách phase rõ ràng, gom mọi lesson cùng `initialFen` thành 1 start node chung.
- Mobile đã có 3 breakpoints (1180/760/480), sticky controls, swipe, tabs Board/Graph/Info.

**Điểm yếu (gap cần giải quyết theo yêu cầu user):**
1. **Tách rời local ↔ global**: Board ở cột 1, graph ở cột 3, không có liên kết trực quan — user phải dò mắt qua 2 vùng. Click node trên graph thì board nhảy nhưng không có breadcrumb path, không có minimap.
2. **Thiếu overview tổng quan**: Không có score-graph (eval chart kiểu Stockfish) để thấy toàn bộ ván. Không có "where am I in the tree" indicator.
3. **Engine analysis bị cô lập**: `EvaluationPanel` chỉ hiện 1 best move, không so sánh với nước trong lesson. Không phân loại best/good/inaccuracy/mistake/blunder.
4. **Graph chưa phóng to/thu nhỏ**: `d3-hierarchy` chỉ layout 1 lần với `nodeSize([72, 210])`. Với combined lesson (6k+ lines) thì node rất nhỏ, không pan/zoom được.
5. **Không có opening book/explorer**: Không biết nước nào "phổ biến", tần suất thắng/thua.
6. **Comment lesson là string tự do**, không có gắn concept phase/principle/equipment. Khi user tua qua 35k nước thì không có summary.
7. **Search chỉ filter theo title**, không search theo move (e.g. "tìm lesson có Pháo đầu C2=5").

## 2. Đề xuất engine & thư viện

### 2.1 Engine

| Engine | Vai trò | Lý do | Trạng thái |
|--------|--------|-------|------------|
| **Pikafish** (đang có) | Phân tích chính | NNUE + Stockfish-derived, mạnh nhất open-source Xiangqi, có sẵn `.nnue` net, hỗ trợ `MultiPV`, `Ponder`. | OK |
| **Fairy-Stockfish** (mới) | Phụ: puzzle, so sánh, "vì sao nước này sai" | Đa biến, có thể chạy ruleset Xiangqi tự định nghĩa; cho phép benchmark với Pikafish. Build 1 lần, switch engine bằng config. | Cần tích hợp |
| Wukong (tùy chọn sau) | Lightweight cho mobile/không có binary | Code đã có enum stub, có thể triển khai nhanh nếu cần. | Defer |

**Lưu ý cho Fairy-Stockfish:**
- Cần file net Xiangqi (Fairy-Stockfish dùng `.nnue` riêng); tải từ Fairy-Stuff hoặc dùng net Stockfish- Xiangqi port.
- Một số UCI option khác Pikafish (`UCI_Variant`, `UCI_Chess960`...). Cần chuẩn hóa interface `EngineAdapter` để không lệ thuộc chi tiết UCI.
- Không nên dùng Fairy-Stockfish làm engine chính; nó chậm và yếu hơn Pikafish cho Xiangqi thuần.

### 2.2 Thư viện frontend

| Mục đích | Đề xuất | Lý do |
|----------|---------|-------|
| Pan/zoom graph | `d3-zoom` (đã có sẵn `d3-hierarchy`) | Cùng hệ sinh thái, không thêm bundle lớn. |
| Eval chart | `uplot` hoặc tự code SVG | Dataset ~30-100 điểm/bài; SVG đủ nhanh, uplot quá overkill. |
| Diff highlight nước đi | `colord` hoặc tự tính | Tính sớm, không cần thư viện. |
| Trie/build d3 | Đã có `d3-hierarchy` | Không cần thêm. |
| Virtualization list | `@tanstack/vue-virtual` | Cho sidebar khi số lesson > 1000. |
| Fuzzy search lesson | `fuse.js` hoặc `flexsearch` | Client-side fuzzy trên ~6k records; fuse.js ~7KB gzipped. |
| Test e2e | `vitest` + `@vue/test-utils` | Thiếu test Vue, cần thêm. |
| Lint/format | `eslint` + `prettier` (chưa có) | Đã có `vue-tsc` typecheck; thiếu lint. |

### 2.3 Thư viện backend

- **`pgn-mongodb`/`pikafish-cli`**: không cần, Pikafish đã là CLI.
- **`pikafish-go-bindings`**: không có, không cần — `os/exec` đã đủ.
- **Opening book**: tự build từ repo `xiangqi-mirror-db` / `pikafish-book` / WXF corpus, lưu JSON. Đã có parser WXF trong `scripts/extract-lessons/parsers`.
- **WASM engine**: không nên, bundle ~5MB, trade-off không tốt khi server có sẵn.

## 3. Cải tiến UX: Local + Global

### 3.1 Tầng nhìn (view layers)

| Layer | Kích thước | Mục đích | Tương tác |
|-------|-----------|----------|-----------|
| **Local** (Board + Inspector) | 60% chiều rộng desktop | Nước hiện tại, eval, comment, controls. | Click preview arrow, next/prev, swipe. |
| **Glance** (Breadcrumb + Mini-map) | 10-15% | Đường dẫn từ root → active node, mini-graph dạng tóm tắt. | Click node, click breadcrumb crumb. |
| **Global** (Full graph + Eval chart) | 25-30% | Toàn bộ cây nước đi, có thể pan/zoom, eval chart theo ply. | Pan/zoom, click node, click điểm chart. |

Trên mobile, 3 layer chuyển thành **3 tab** (đã có sẵn Board/Graph/Info) + **mini-bar ở cuối** (bottom dock) hiện breadcrumb + nút "mở graph".

### 3.2 Tính năng cụ thể

#### 3.2.1 Breadcrumb path
- Hiển thị dãy node từ `start → ... → active` với notation ngắn (`Mã 8 tiến 7`, `Pháo 2 bình 5`).
- Click crumb để nhảy board tới ply tương ứng.
- Mỗi crumb có icon `history` xám → `active` cam → `past` xanh.
- Component: `apps/web/src/components/MoveBreadcrumb.vue` (mới).

#### 3.2.2 Mini-map
- SVG thu nhỏ 200×120 hiển thị toàn bộ trie, mỗi node là 2-3px dot.
- Vùng hiện tại (active node ± 2 ply) được khoanh rectangle.
- Click mở `MoveGraph` ở tab Graph, kéo rect để pan full graph.
- Component: `apps/web/src/components/MoveMinimap.vue` (mới), dùng lại `treeGraph` d3-hierarchy với `nodeSize([8, 8])`.

#### 3.2.3 Evaluation chart
- Đường line chart X = ply, Y = `score.cp` (clamp ±900).
- Mỗi điểm = 1 ply. Điểm hiện tại có halo.
- Hover hiện tooltip: ply, FEN (mini), eval, best move.
- Màu: đỏ = đỏ hơn, đen = đen hơn, gradient giống `EvaluationPanel.score-track`.
- Có thể vẽ "lesson line" (đậm) và "engine best line" (chấm) cùng lúc để so sánh.
- Cache dữ liệu theo FEN (cache key như engine hiện tại + 1 cache layer cho toàn bài).
- Component: `apps/web/src/components/EvalChart.vue` (mới).

#### 3.2.4 Pan/zoom full graph
- Tích hợp `d3-zoom` vào `MoveGraph` (thêm `transform` group quanh `viewport`).
- Mouse wheel zoom, drag pan, double-click fit.
- Bỏ qua bước "fit" tự động khi user đang tương tác.
- Nút "Fit" và "Reset" trong toolbar.
- Cải tiến `apps/web/src/graph/treeGraph.ts`: thêm `TreeGraphRenderer.zoom()` API.

#### 3.2.5 Move classification (badge trên graph + board)
- Khi mở lesson, server pre-compute eval cho mỗi ply (lưu trong `combined-lesson.json` hoặc cache riêng).
- Phân loại: `best` (sai số ≤ 20cp), `good` (≤ 50cp), `inaccuracy` (≤ 100cp), `mistake` (≤ 300cp), `blunder` (> 300cp).
- Hiển thị: ring màu quanh piece khi replay, badge trên graph node.
- API mới: `GET /api/lessons/{id}/classification` (cached).

#### 3.2.6 Comparative mode
- Mở `CombinedLessonPage`, thêm toggle "So sánh nước đi với engine".
- Bật lên: hiện 2 marker trên board — nước lesson (vàng) + nước engine (xanh dương). Nếu trùng: marker xanh đậm.
- Mỗi lần replay ply mới, gọi `/api/analyze` với FEN trước, cache key như hiện tại.

### 3.3 Layout đề xuất (desktop)

```
┌──────────────┬────────────────────────────────┬─────────────────────────┐
│ Library      │ Breadcrumb (1 dòng)            │ Glance (minimap + tabs) │
│ 280px        ├────────────────────────────────┤                         │
│              │ Board + Inspector              │ Full graph (pan/zoom)   │
│              │  – Board 9x10                  │                         │
│              │  – Eval badge                  │                         │
│              │  – Comment                     │                         │
│              │  – Controls                    │                         │
│              ├────────────────────────────────┤                         │
│              │ Eval chart (80px height)       │                         │
└──────────────┴────────────────────────────────┴─────────────────────────┘
```

Trên mobile, 3 cột collapse thành 1 cột + bottom dock (breadcrumb + "mở graph" + "mở eval").

## 4. Cải tiến AI / Engine

### 4.1 Engine abstraction

Tạo interface trong `apps/api/internal/engine/` (mới):

```go
type Engine interface {
    Analyze(ctx context.Context, req Request) (Response, error)
    ID() string
    Capabilities() Capabilities
}

type Capabilities struct {
    MultiPV        bool
    Ponder         bool
    MaxHashMB      int
    SupportsXiangqi bool
    BuiltAt        time.Time
    NetFile        string
}
```

Refactor `analysis.Analyzer` → `engine.Engine`; giữ alias để không vỡ test. Pikafish wrap vào `PikafishAdapter`, Fairy-Stockfish wrap vào `FairyStockfishAdapter`. Switch qua `ENGINE_KIND` env hoặc `?engine=` query param.

### 4.2 Multi-engine response

`POST /api/analyze` nhận thêm field `engines: ["pikafish", "fairy"]`, response trả về từng engine một:

```json
{
  "primary": { "source": "pikafish", "score": {...}, "bestMove": {...} },
  "secondary": { "source": "fairy-stockfish", "score": {...}, "bestMove": {...} },
  "agreement": "same" | "different",
  "agreementMove": "..." | null
}
```

Coi Pikafish là primary, Fairy-Stockfish là secondary, không cache secondary trong 10 phút (cache 1 phút) vì dùng để so sánh.

### 4.3 Move classification endpoint

`POST /api/classify-line` nhận `{lineId, moves: [{side, from, to}], openingFen}` → trả `{plies: [{cpDelta, classification, bestMove}]}`.

Server pre-compute tuần tự từng ply bằng Pikafish. Cache kết quả theo `(lineId, openingFen, engineVersion)` trong 24h.

Tham khảo Lichess: best ≤ 10cp, excellent ≤ 30cp, good ≤ 80cp, inaccuracy ≤ 150cp, mistake ≤ 300cp, blunder > 300cp.

### 4.4 Opening book

Build từ WXF corpus trong `scripts/extract-lessons/parsers` → `apps/api/data/book/opening-book.json` (Pre-compute):
- Key: FEN ở ply N (cùng nước đầu tiên, 5 nước đầu).
- Value: `{moves: [{uci, frequency, winRate, eco, name}]}`.

`GET /api/opening?fen=...` trả về các nước con phổ biến + win rate. Frontend dùng để highlight trên board khi ở giai đoạn khai cuộc (5-7 nước đầu).

## 5. Cải tiến Data & API

### 5.1 Mở rộng schema lesson

Thêm vào `apps/api/internal/lessons/types.go` (không breaking — dùng `omitempty`):

```go
type Move struct {
    ID           string `json:"id"`
    Side         string `json:"side"`
    From         Coordinate `json:"from"`
    To           Coordinate `json:"to"`
    Comment      string `json:"comment,omitempty"`
    Theme        []string `json:"theme,omitempty"`         // mới: ["pin", "fork", "discovered"]
    Concept      string   `json:"concept,omitempty"`       // mới: "control center"
    Difficulty   int      `json:"difficulty,omitempty"`    // mới: 1-5 sao
    PlyNumber    int      `json:"plyNumber,omitempty"`
    BestMoveUCI  string   `json:"bestMoveUci,omitempty"`   // pre-computed
    Classification string `json:"classification,omitempty"` // best/good/mistake/...
}
```

Migration: chạy script `scripts/classify_lessons.py` dùng Pikafish batch (multi-PV) trên toàn bộ 35k nước, ghi lại `bestMoveUci` + `classification`. Có thể chạy nền 1-2 giờ trên 1 máy dev.

### 5.2 API mới

| Method | Path | Mục đích | Cache |
|--------|------|----------|-------|
| GET | `/api/lessons/{id}/classification` | Phân loại nước đi cho cả lesson. | ETag + 1h |
| GET | `/api/lessons/{id}/opening` | Tra opening book cho FEN đầu. | ETag |
| GET | `/api/opening?fen=...` | Tra opening book. | ETag + 24h |
| POST | `/api/analyze` (mở rộng) | Hỗ trợ `engines` array, `multipv`, `untilDepth`. | 10' |
| POST | `/api/classify-line` | Phân loại 1 line (cho editor nội dung). | 24h |
| GET | `/api/lessons/search?q=...&theme=...&concept=...` | Search trong comment/theme. | 5' |

### 5.3 Backend performance

- Tách `apps/api/cmd/server` thành 2 binary: `lesson-api` (stateless) + `engine-pool` (stateful) → scale độc lập.
- `engine-pool` chạy nhiều process Pikafish + 1 Fairy-Stockfish, phân phối qua queue.
- Tuy nhiên với scope hiện tại (1 process, 1 user dev), giữ single-process, defer pool.

## 6. Cải tiến kiến trúc

### 6.1 Tách module

| Hiện tại | Đề xuất |
|----------|---------|
| `App.vue` 402 dòng | Tách: `useLessonWorkspace()` composable; `LessonTopbar.vue`, `LessonInspector.vue` components. |
| `MoveGraph.vue` 423 dòng + trie build inline | Tách: `composables/useMoveTrie.ts`, `composables/useGraphNavigation.ts`. |
| `CombinedLessonPage.vue` duplicate `App.vue` logic | Tái dùng `useLessonWorkspace()`; chỉ khác layout. |
| `engine/types.ts` chỉ ở frontend | Mirror sang `internal/engine/types.go` rồi generate. |

### 6.2 Testing

- Thêm `vitest` cho Vue: unit test `useLessonPlayer`, `useMoveEvaluation`, `MoveGraph` trie, FEN parse.
- E2E Playwright: flow "open lesson → next 3 moves → click graph node → expect board updated".
- Backend: bổ sung table test cho engine mock, classification endpoint.

### 6.3 Cleanup docs

- **Khôi phục 3 file specs bị xoá** (`docs/specs/planning/combined-lesson-page.md`, `performance-optimization.md`, `smart-cache-strategy.md`) từ git history (đã thấy trong `git log`: `8ac0faf feat(api,web,docs): add smart caching, persistent UCI engine and performance optimizations`).
- Sync lại `docs/_index.md` để khớp thực tế.
- Cập nhật `docs/features/lesson-experience.md` để chứa mô tả mới (eval chart, classification, opening book).

### 6.4 Lint/format

- Thêm `eslint` config (flat config, Vue + TS).
- Thêm `prettier` cho `apps/web/src` + `apps/api/internal`.
- Chạy pre-commit hook kiểm tra formatting.

## 7. Phase triển khai

### Phase 0 — Foundation (1-2 ngày)
- Khôi phục 3 docs bị xoá (từ `git show 8ac0faf:docs/specs/planning/...`).
- Cập nhật `docs/_index.md`, `docs/features/lesson-experience.md`.
- Tách `useLessonWorkspace()` composable, refactor `App.vue` xuống <200 dòng.
- Thêm `vitest`, viết test cho `useLessonPlayer`, FEN parse, trie build.

### Phase 1 — Move Breadcrumb + Mini-map (3-5 ngày)
- `MoveBreadcrumb.vue`, tích hợp vào `App.vue` + `CombinedLessonPage.vue`.
- `MoveMinimap.vue`, đặt cạnh breadcrumb.
- Verify pan/zoom mini-map mở full graph.
- Test responsive: trên mobile, đưa breadcrumb vào bottom dock.

### Phase 2 — Eval chart (3-5 ngày)
- API: thêm endpoint trả eval cho toàn bộ line (1 lần gọi khi mở lesson, cache ETag).
- `EvalChart.vue`, đặt dưới board inspector.
- Cache layer client: lưu eval theo `lessonId + lineId`.
- Khi user click điểm chart → nhảy board tới ply.

### Phase 3 — Pan/zoom full graph (2-3 ngày)
- Thêm `d3-zoom` vào `treeGraph.ts`.
- Cải tiến layout để không overlap node.
- Test với 1k+ node.

### Phase 4 — Engine abstraction + Fairy-Stockfish (1 tuần)
- Tạo `internal/engine/`, refactor `analysis` thành wrapper.
- Build Fairy-Stockfish binary (CMake + net).
- Mở rộng `/api/analyze` cho multi-engine.
- Mở rộng response shape.

### Phase 5 — Move classification (1 tuần)
- Viết `scripts/classify_lessons.py` batch pre-compute.
- API `/api/lessons/{id}/classification`.
- Component badge trên board + graph.
- Phân loại theo Lichess thresholds.

### Phase 6 — Opening book (3-5 ngày)
- Build `apps/api/data/book/opening-book.json` từ WXF corpus.
- API `/api/opening`.
- Highlight trên board ở 5-7 nước đầu (key FEN ở start).
- (Optional) Search lesson theo opening.

### Phase 7 — Comparative mode + AI improvements (1 tuần)
- Toggle "So sánh với engine" trong combined page.
- 2 marker trên board.
- Suggestion engine cho editor: gợi ý theme tag dựa trên move shape.

### Phase 8 — Polish (3-5 ngày)
- Animation cho breadcrumb transition.
- Hiệu ứng glow cho active node trên mini-map.
- A11y: keyboard nav trong graph (Tab/Arrow/Enter).
- Lint + format + e2e tests.

## 8. Tiêu chí hoàn thành

### UX
- [ ] User nhìn 1 lần thấy được: vị trí hiện tại (breadcrumb), toàn cảnh (mini-map), eval trend (chart), cờ (board) — không phải scroll.
- [ ] Click node trên full graph → breadcrumb update + board jump trong <200ms.
- [ ] Click điểm eval chart → board jump.
- [ ] Trên mobile (375px), bottom dock luôn hiện breadcrumb + 2 nút "Graph" "Eval".

### AI
- [ ] `POST /api/analyze` trả về 2 engine khi client gửi `engines: ["pikafish", "fairy-stockfish"]`.
- [ ] 35,183 nước đi đã có `classification` + `bestMoveUci` trong `lessons.json`.
- [ ] Opening book 6k+ position, response <50ms.
- [ ] Toggle "So sánh engine" hiển thị đồng thời 2 marker.

### Data & API
- [ ] Tất cả API list có ETag + X-Data-Version, conditional GET 304 đúng.
- [ ] `/api/lessons/{id}/classification` cache 1h, xử lý 1 lesson 50 nước trong <2s.
- [ ] Search lesson theo theme/concept hoạt động.

### Kiến trúc
- [ ] `App.vue` <200 dòng.
- [ ] `MoveGraph.vue` <200 dòng (sau khi tách composables).
- [ ] Backend có interface `Engine`, dễ thêm engine thứ 3.
- [ ] `vitest` cover ≥60% logic core (`useLessonPlayer`, `useMoveEvaluation`, `useMoveTrie`, FEN).
- [ ] E2E test 1 flow board → graph → eval chart.
- [ ] Docs: 3 file specs khôi phục + `lesson-experience.md` cập nhật + index đồng bộ.

## 9. Rủi ro & giảm thiểu

| Rủi ro | Tác động | Giảm thiểu |
|--------|---------|------------|
| Pre-compute 35k nước mất nhiều giờ | Chặn deploy | Chạy nền, persist incremental; cho phép fallback "no classification" cho lesson chưa pre-compute. |
| Fairy-Stockfish binary build trên macOS arm64 vs linux amd64 | Engine phụ không chạy trên 1 trong 2 môi trường | Build cả 2 binary, config `ENGINE_FAIRY_PATH` riêng; nếu không có thì response chỉ trả primary engine. |
| Pan/zoom trên 1k+ node chậm | UX giật | Ảo hóa node: chỉ render node trong viewport + buffer; dùng `requestAnimationFrame` debounce transform. |
| Sửa `App.vue` lớn dễ regress | Bug mobile/desktop | Tách nhỏ, mỗi phase 1 commit, test responsive sau mỗi phase. |
| `next-steps` API phải đổi shape | Breaking | Giữ endpoint cũ, thêm endpoint mới `/api/lessons/{id}/line-graph` cho phase 5+; dần migrate. |
| Pikafish binary quá lớn (~40MB) trong repo | Repo phình | Không commit binary; `Taskfile dev:api` đã assume user tự cài; document trong README. |

## 10. Quyết định đã chốt (2026-06-06)

| # | Câu hỏi | Quyết định | Ghi chú |
|---|---------|-----------|---------|
| 1 | Nguồn opening book | **WXF corpus** (~5k ván, đã có parser) + **CCRL/Pikafish book mở rộng** (~50k ván) | Build pipeline 2 bước: WXF làm nền, merge với CCRL/Pikafish book để tăng tần suất các nước mở. File output: `apps/api/data/book/opening-book.json`. |
| 2 | Fairy-Stockfish net | **Net mặc định** đi kèm binary (~5MB) | Dùng cho so sánh thứ cấp, không cần patch. |
| 3 | Auth + user profile | **Stateless, không auth** | Progress lưu `localStorage` nếu cần (ngoài scope). Repo giữ nguyên triết lý content-first. |
| 4 | Thứ tự execute | **Parallel UX + Engine** (2 track song song) | Track A: UX local+global (Phase 0→3). Track B: Engine expansion (Phase 4 + 5 lite). Merge trước khi Phase 6+7. |

## 11. Track song song sau chốt

### Track A — UX Local + Global (1 người, ~2 tuần)
- A0: Foundation (khôi phục docs, refactor `App.vue`, vitest) — 1-2 ngày
- A1: Breadcrumb + Mini-map — 3-5 ngày
- A2: Eval chart — 3-5 ngày
- A3: Pan/zoom full graph — 2-3 ngày

### Track B — Engine Expansion (1 người, ~2 tuần, chạy song song)
- B0: Tạo `internal/engine/`, refactor `analysis` thành wrapper, `Engine` interface — 2-3 ngày
- B1: Fairy-Stockfish build script (CMake + Makefile), config `ENGINE_FAIRY_PATH` — 1-2 ngày
- B2: Multi-engine `/api/analyze` response shape + tests — 2-3 ngày
- B3: Pre-compute classification cho opening subset (~2k nước đầu, dùng Pikafish) — 2-3 ngày
- B4: `/api/lessons/{id}/classification` + cache 1h — 1-2 ngày

### Merge gate (trước Phase 6+7)
- Track A và B phải merge vào `main` hoặc tạo integration branch `design/ux-engine-merged`.
- E2E test chạy cùng 1 flow: mở lesson → breadcrumb OK + minimap OK + eval chart OK + classification badge OK.
- Nếu conflict, ưu tiên Track A (UI) làm nền, Track B plug vào sau.

### Phase 6+7+8 chạy sequential sau khi merge
- Phase 6: Opening book (WXF + CCRL merge, ~1 tuần)
- Phase 7: Comparative mode (1 tuần)
- Phase 8: Polish (3-5 ngày)

## 12. Tiêu chí "Definition of Done" của toàn plan

Track A done:
- [ ] User mở 1 lesson thấy: breadcrumb path + minimap + eval chart + board + graph (pan/zoom) cùng lúc trên desktop.
- [ ] Mobile (375px): bottom dock breadcrumb + 2 nút "Graph"/"Eval" hoạt động.
- [ ] Click node trên graph → board + breadcrumb update trong <200ms.
- [ ] Click điểm chart → board update.
- [ ] `vitest` cover ≥60% logic core.

Track B done:
- [ ] `Engine` interface ở backend, Pikafish + Fairy-Stockfish đều implement.
- [ ] `/api/analyze?engines=pikafish,fairy-stockfish` trả 2 engine.
- [ ] Classification pre-compute cho 1 lesson opening đầu tiên, response <500ms.
- [ ] `/api/lessons/{id}/classification` cache 1h, 304 OK.

Combined done (sau Phase 6+7):
- [ ] Opening book merged 2 nguồn, response `/api/opening` <50ms.
- [ ] Comparative mode: 2 marker trên board.
- [ ] Docs đồng bộ: 3 specs khôi phục, `lesson-experience.md` cập nhật, index sạch.
