# UI Improvement Plan v2

Dựa trên 18 screenshots chụp tại 6 resolutions (375, 414, 768, 1280, 1440, 1920) ×
3 routes (home, lesson, combined), ghi nhận 12 vấn đề UI cụ thể. Plan này tập
trung vá các lỗi layout và polish responsive — không mở rộng design system.

## 1. Audit Findings

### 1.1 Layout collapse ở desktop (1440+, 1920)

Layout `.app-shell` 3-col + `.board-panel` 2-col internal + `.lesson-panel` ở
col 3 với `grid-row: 1/-1`. Kết quả:

- **Vấn đề L1 (Critical)**: Right column (col 3) có ~700px empty space giữa
  "Điểm cần nhớ" card (top) và EvalChart strip (bottom). Nguyên nhân:
  `.board-panel` ở row 3 span cols 2-3, internal 2-col grid có col 2 bị
  `align-content: start` → minimap+graph+eval pack ở top, để trống phần còn
  lại. Trong khi `.lesson-panel` ở col 3 với content ngắn tạo "white block"
  phía dưới card. Hai layout giao thoa → gap lớn.
- **Vấn đề L2 (Major)**: EvalChart dùng `grid-column: 2/-1` của board-panel
  2-col internal, nên nó nằm trong col 2 (1fr) thay vì stretch full width.
  Trên màn 1440+ chỉ hiện ~480px wide — quá nhỏ cho chart.
- **Vấn đề L3 (Major)**: MoveMinimap dùng `aspect-ratio: 16/9` ở col 1fr
  (~420px) → ~237px tall, quá cao. Minimap hiển thị 1 hàng dots trên nền
  dotted, đọc không ra "minimap".

### 1.2 Mobile (375, 414)

- **Vấn đề M1 (Critical)**: Combined page mobile thiếu structure:
  - "Tổng hợp toàn bộ lesson" header lạc vị trí (không có ngữ cảnh).
  - Tab pills "Bàn cờ | Phân tích | Biên" chỉ có 1 tab "Bàn cờ" active.
  - "Bắt đầu" button không có nhãn rõ (chỉ icon flag).
- **Vấn đề M2 (Major)**: Brand header "Học bằng thế cờ thật" wraps 3 dòng
  ở 375px. Logo 帥 chiếm ~64px, tiêu đề wrap 3 dòng → header cao ~120px,
  ăn vào viewport.
- **Vấn đề M3 (Minor)**: Search input trên mobile bị sticky khi scroll
  category list (chưa test kỹ — verify bằng scroll).

### 1.3 Tablet (768)

- **Vấn đề T1 (Critical)**: Sidebar vẫn render inline ở 768px (~250px wide),
  khiến board bị ép xuống ~280px. Phải collapse thành off-canvas từ <1024px
  hoặc <1100px.
- **Vấn đề T2 (Major)**: Ở 768px, `.lesson-panel` (col 3) vẫn hiển thị
  principles card, làm board column chật.
- **Vấn đề T3 (Major)**: Combined page ở 768px: tree label "1.Pháo 8 b..."
  bị cắt phải.

### 1.4 MoveGraph rendering

- **Vấn đề G1 (Major)**: MoveGraph ở 1920px và 1440px hiển thị chỉ 1 hàng
  dots kéo dài. Layout single-row, không có depth indication. Nguyên nhân:
  d3-hierarchy tree layout với shallow depth (chỉ 1-2 levels mở rộng từ
  start) → tất cả nodes collapse vào 1 hàng. Cần force multi-row khi
  width > height.
- **Vấn đề G2 (Minor)**: MoveGraph empty area (background dotted grid) quá
  nổi bật so với nội dung. Tone down opacity hoặc dùng subtle pattern.

### 1.5 Combined page (1440)

- **Vấn đề C1 (Major)**: Board column có ~700px empty space dưới board
  vì col 2 (1.4fr) stretch full height nhưng board chỉ ~640px. Cần anchor
  content xuống bottom (engine analysis, move comment) HOẶC column 1 bên
  phải nên có secondary content.
- **Vấn đề C2 (Minor)**: "Bắt đầu" button lơ lửng ở rất dưới trái, không
  trong flow hợp lý.

## 2. Phased Plan

### Phase F1: Critical Layout (1 ngày)

Sửa L1, L2, L3, T1. Tập trung vào 3-col bento chuẩn.

- **F1.1**: Refactor `.app-shell` → 3-col bento chuẩn:
  - `.library-panel` ở col 1, `position: sticky; top: 0; height: 100svh; overflow: auto`
    (đã sticky rồi, cần verify).
  - `.board-panel` ở col 2-3 với internal 2-col (1.55fr / 1fr).
  - `.lesson-panel` ở col 3, `position: sticky; top: 0; align-self: start; max-height: 100svh; overflow: auto`.
    Sticky để card luôn ở viewport khi board scroll. Bỏ `grid-row: 1/-1`.
- **F1.2**: `.board-side-panel` (chứa MoveMinimap+MoveGraph) → set
  `min-height: 0; height: 100%` và `align-content: stretch` (thay vì start)
  để 2 children chia đều height còn lại. Thêm `flex: 1 1 0` cho MoveMinimap
  và MoveGraph.
- **F1.3**: EvalChart: bỏ `grid-column: 2/-1` của board-panel internal, đổi
  thành `grid-column: 1 / -1` (full width band dưới board-side-panel) HOẶC
  cho EvalChart thành 1 child trong `.board-side-panel` với `flex: 1 1 0`.
  Chọn option 2: EvalChart ở trong board-side-panel để giữ 1-col flow đơn
  giản, eval chart full width của col 2 (1fr).
- **F1.4**: Tablet sidebar: thêm breakpoint `<= 1024px` để sidebar
  collapse thành off-canvas. Hiện tại chỉ collapse ở mobile breakpoint
  (~700px). Verify breakpoint hiện tại.

### Phase F2: Mobile Polish (0.5 ngày)

Sửa M1, M2, M3.

- **F2.1**: Brand header: `font-size: 14px` thay vì inherit; cắt còn
  "Học bằng thế cờ" 2 dòng max thay vì 3; hoặc collapse thành logo-only
  ở <414px.
- **F2.2**: Combined mobile — render full `<LessonMobileTabs>` (Bàn cờ |
  Phân tích | Biên) thay vì 1 tab. Hiện đang có `<LessonMobileTabs>` trong
  App.vue nhưng chưa được dùng trong CombinedLessonPage. Cần reuse.
- **F2.3**: Combined mobile — wrap "Bắt đầu" trong card với breadcrumb
  "Tổng hợp toàn bộ lesson" + 1 dòng description.

### Phase F3: Graph + Minimap (0.5 ngày)

Sửa G1, G2, L3.

- **F3.1**: MoveMinimap: giảm `aspect-ratio` từ `16/9` xuống `4/1` (chỉ
  strip ngang). Hoặc dùng fixed height `min(120px, 18vh)` thay vì aspect.
  Đổi node radius từ 6 xuống 3 để nodes nhỏ hơn, density cao hơn.
- **F3.2**: MoveGraph: detect khi `treeWidth > canvasWidth * 1.5` →
  force multi-row layout bằng cách set `nodeSep`/`rankSep` to `auto`
  thay vì `1.0`/`0.7`. Hoặc đơn giản hơn: đổi `rankdir` thành
  `LR` (left-right) thay vì `TB` (top-bottom) — tận dụng width.
- **F3.3**: Tone down dotted background opacity từ 0.10 xuống 0.04 ở
  `.graph-canvas` và `.move-minimap-canvas`.

### Phase F4: Combined Page Polish (0.5 ngày)

Sửa C1, C2, T3.

- **F4.1**: Combined page col 1: anchor content xuống bottom bằng
  `display: flex; flex-direction: column; justify-content: space-between`
  trong `.combined-board-column`. Hoặc gom board + engine + comment vào
  1 column card.
- **F4.2**: Tree label truncate: thêm `text-overflow: ellipsis;
  white-space: nowrap; max-width: 240px; overflow: hidden` cho
  `.move-graph-leaf-label` và tương đương ở CombinedLessonPage tree.
- **F4.3**: Combined page mobile: tab structure đầy đủ (F2.2) sẽ giải
  quyết "Bắt đầu" lơ lửng.

## 3. Validation

Sau mỗi phase:
- Chạy lại `shoot.mjs` để chụp 18 screenshots mới.
- Visual diff: principles card ở col 3 sticky, no gap. EvalChart full
  width. Mobile combined có tabs.
- Build pass, 40 vitest pass, ESLint 0 errors.

## 4. Out of Scope (chưa làm)

- Light/dark mode toggle (đã có theme system, chỉ thiếu UI switcher).
- Piece SVG variant (đang dùng text glyph 車馬象士將炮兵 — OK cho editorial).
- Animation polish (đã cap motion 3/10 trong plan cũ).
- Wiki article route (/wiki) — chưa có route.
- Opening suggestions highlight khi move match (đã có logic, chưa test).
