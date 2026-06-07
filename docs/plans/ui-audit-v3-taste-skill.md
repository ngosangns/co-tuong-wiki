# UI Audit v3 (Taste-Skill Pass)

Audit từ 36 screenshots (6 resolutions × 2 color schemes × 3 routes) trên
localhost:5173, áp dụng `design-taste-frontend` skill (anti-slop + three
dials + section discipline). Phát hiện 9 vấn đề cụ thể; plan 4 phases.

## 0. Design Read

> *"Reading this as: interactive learning product (Co Tuong / Chinese chess)
> for intermediate Vietnamese-speaking players, with an Editorial × Dark-tech
> language (locked in Plan v1), leaning toward native CSS + serif reading
> type + restrained motion + warm wood + bento workspace."*

**Three dials (held from Plan v1):**
- `DESIGN_VARIANCE: 6` — asymmetric bento, sidebar weight, off-center mini-marks
- `MOTION_INTENSITY: 3` — only hover/active states + 200ms overlay fade
- `VISUAL_DENSITY: 5` — card-bounded but breathing; 16/24/20 rhythm

**Not applicable to this product:** Hero / CTA / nav height / marquee /
zigzag cap rules (these are landing-page rules). Applied: anti-AI-tells,
typography discipline, color lock, eyebrow restraint, materiality,
dark mode parity, mobile collapse, motion restraint, copy self-audit.

## 1. Findings

### 1.1 Card over-use in right column (Materiality violation)

`.board-side-panel` ở 1440+ hiển thị 3 cards stacked với cùng elevation
(`card-outlined`): MoveMinimap, MoveGraph, EvalChart. Không có hierarchy
hierarchy thật giữa 3 cái — chúng cùng cấp data display. Skill §4.4:
*"Use cards ONLY when elevation communicates real hierarchy. Otherwise
group with border-t, divide-y, or negative space."*

Đề xuất: chuyển 3 cards thành 1 panel với `border-top` chia 3 sections
(Minimap strip + Graph canvas + EvalChart). Hoặc: 1 panel outer, 2 inner
divider-separated rows (Graph + Eval). MoveMinimap là metadata overlay
trên graph — không cần card riêng.

### 1.2 "Chưa chọn nước" details card redundant

Khi user chưa chọn nước, MoveGraph shell hiển thị aside `<div class="graph-details">`
chứa "Chưa chọn nước / Bắt đầu / Biến chính giữ tiên". Khi đã chọn, aside
update thành notation + side. Nhưng:
- Notation đã hiện ngay trên graph node label rồi
- "Biến chính giữ tiên" line link không có giá trị trong flow chính
- Aside trống ~50px khi không có move

Đề xuất: bỏ aside, fold "Bắt đầu" indicator vào MoveGraph toolbar (đã có
"Fit" button). Một toolbar với breadcrumb-style current-position + Fit/Reset
actions. Bớt 1 DOM subtree + bớt 1 visual element.

### 1.3 Eyebrow redundancy

Đếm trong lesson view desktop: 8 labels (Tổng hợp, Khai cuộc, Động cơ,
Opening book, Đánh giá ván cờ, Sơ đồ toàn cảnh, Cây nước đi, Bắt đầu).
Skill §4.7: max 1 eyebrow / 3 sections cho landing. Ở product app thì
strict ratio lỏng hơn, nhưng vẫn có 2 thừa:
- "Tổng hợp" trên Điểm cần nhớ — icon Lightbulb đã nói "summary"
- "Sơ đồ toàn cảnh" trên MoveMinimap — "minimap" đã rõ từ tên class

Đề xuất: xóa 2 eyebrow này, giữ 1 word headline. Pattern: chỉ giữ eyebrow
khi nó categorize (như "Khai cuộc" trên principles = "đây là principles
của Khai cuộc category").

### 1.4 MoveMinimap not informative (Materiality + Data)

Trong 18 screenshots lesson, minimap luôn hiện 1 hàng ~5-10 dots + 1
dashed rectangle quanh current path. Không có:
- Tooltip on hover (position, FEN, move count)
- Visual hierarchy cho past vs future path
- Color state cho active sub-tree

Skill §1.1: data communicates. Hiện tại minimap = decorative dots.
Đề xuất: thêm hover tooltip + highlight past-path dots khác future dots.
Hoặc: bỏ minimap (MoveGraph đã show structure đầy đủ).

### 1.5 Numbers not in mono

Skill §1.0: ở VISUAL_DENSITY > 7, mandatory mono. App ở 5-6 nên optional.
Nhưng các số sau sẽ align tốt hơn nếu dùng mono:
- EvalChart plies "0 62" và "Cân bằng" tick labels
- Opening book percentages "74% 15% 4% 3% 2%"
- Move graph labels "1. Pháo 2 bình 5 · 14 biến"
- Phase tab counts "494 biến / 184 biến / 153 biến"
- Sidebar lesson count "302"
- Breadcrumb counters "48 ý", "12"

Geist Mono đã được load (`index.html`). Chỉ cần thêm class hoặc áp
`font-family: var(--font-mono)`. Nên dùng cho tất cả số + tick labels,
không cho body text. Cải thiện alignment + "data display" feel.

### 1.6 CJK italic annotation in Vietnamese-first UI

Opening book rows hiện tên opening + italic CJK trong ngoặc:
- "Bình phong mã *(屏风马)*"
- "Pháo đầu *(中央炮)*"
- "Tượng 7 tiến 5" (no CJK)

Audience là người Việt. Italic CJK là convention quốc tế, nhưng ở
app Việt-first, nên:
- Bỏ CJK, hoặc
- Đổi thành "(Pinyin)" nếu cần international context, hoặc
- Dùng CJK như label chính (red 14px), Vietnamese là sub-label (12px muted)

Hiện tại CJK ở italic 11px (eyebrow style) thì hơi nhỏ và mờ. Có thể
là eyebrow tag với bg subtle.

### 1.7 Combined page col 1 empty space

Ở 1440 combined, board (~700px tall) nằm ở col 1, còn col 2 (engine
status) và col 3 (graph) cao hơn (~1300px). Col 1 có ~600px trống
phía dưới board.

Đề xuất: chuyển `.combined-inspector` (engine status + comment) sang
col 1 dưới board. Move graph giữ col 3. 2-col thay vì 3-col khi ≥1024px.
Sẽ tận dụng vertical space tốt hơn.

### 1.8 Layout repetition across routes

Mỗi route (home, lesson, combined) dùng cùng 3-col layout:
- Sidebar | Main | Right-stack (hoặc Center | Empty | Graph ở combined)
- Bento trong bento
- Cùng card style

Skill §4.7 Section-Layout-Repetition Ban áp dụng lỏng cho product app,
nhưng combined page có thể phá vỡ pattern bằng:
- 2-col asymmetric (board lớn trái, graph panel phải) thay vì 3-col
- Board chiếm 60% width, phần còn lại là full-height graph + tabs
- Hoặc: full-width board ở trên, 3-col mini bento ở dưới

### 1.9 Mobile combined tabs cosmetic

Trên 375px:
- "Tổng hợp toàn bộ lesson" title ở giữa (text-center)
- "Bắt đầu" flag button full width
- Tabs "Bàn cờ | Phân tích | Biên" 3 equal pills

Functional, không có bug. Nhưng design có thể cải thiện:
- Title ở left-aligned sẽ cân với "Danh mục" left
- Tab pills có thể nén padding (đang có vẻ rộng)
- Tab active state dùng bg teal solid (OK)

## 2. Plan

### Phase T1 — Materiality & Eyebrow cleanup (1d)

Triển khai từ findings 1.1, 1.2, 1.3. Không thay đổi grid, chỉ thay đổi
weight.

- **T1.1**: Bỏ card-outlined khỏi MoveMinimap, MoveGraph, EvalChart.
  Wrap cả 3 trong 1 panel `.board-side-panel` với `border: 1px solid var(--color-border); background: var(--color-surface)`. Internal:
  - MoveMinimap: top strip `padding: 12px; border-bottom: 1px solid var(--color-border)`
  - MoveGraph: middle section, padding 12px, không border
  - EvalChart: bottom section, padding 12px, `border-top: 1px solid var(--color-border)`
- **T1.2**: Bỏ `<aside class="graph-details">` khỏi MoveGraph. Thay
  vào đó, MoveGraph toolbar có 1 slot "current position" bên trái Fit
  button (kiểu breadcrumb, e.g. "Vị trí chuẩn" hoặc "1. Pháo 2 bì").
- **T1.3**: Bỏ eyebrow "Tổng hợp" trên LessonInspectorPanel, bỏ
  "Sơ đồ toàn cảnh" trên MoveMinimap. Chỉ giữ icon + bold title.

### Phase T2 — Data display (1d)

Findings 1.4, 1.5, 1.6. Touch nhỏ nhưng ảnh hưởng nhiều đến
"data display" feel.

- **T2.1**: Add `.font-mono` class với `font-family: var(--font-mono)`
  (Geist Mono đã load ở index.html). Apply cho:
  - EvalChart plies (0, 6, 62) và "Cân bằng" label
  - Opening book percentages
  - Move graph labels (sau node name)
  - Sidebar lesson count "302", "105"
  - Principles count "48 ý", "12"
  - Phase tab counts "494 biến"
- **T2.2**: MoveMinimap: thêm `cursor: help` + tooltip khi hover node
  (dùng native `<title>` element đã có, enhance với visual hint).
  Highlight past-path dots: `--color-primary` cho past, `--color-text-soft`
  cho future (currently both are similar opacity).
- **T2.3**: CJK trong Opening book: chuyển từ italic inline thành
  small badge sau tên. Pattern: `Bình phong mã` (display) + `屏风马` 
  (small badge với bg subtle, color text-soft, 9-10px, mono).

### Phase T3 — Layout variance & content fill (1d)

Findings 1.7, 1.8. Touch vào grid structure.

- **T3.1**: Combined page 2-col asymmetric: `.combined-page` cols
  `minmax(420px, 1.6fr) minmax(320px, 0.9fr)` thay vì 3-col. Col 1
  chứa board + engine status + comment (1 flow dọc, internal flex col).
  Col 2 chứa minimap + graph (full height, sticky). MoveMinimap ở
  col 2 top, MoveGraph full height.
- **T3.2**: Responsive 1024px+ combined: 2-col như trên. <1024px:
  fallback 1-col stack hiện tại.
- **T3.3**: Tại >1440px: thử thêm "phase quick-jump" breadcrumb dưới
  graph (4 phase pills: All | Opening | Middlegame | Endgame) — optional
  discovery feature.

### Phase T4 — Mobile polish (0.5d)

Finding 1.9.

- **T4.1**: Mobile combined header: `text-align: left` thay vì center,
  align với `Danh mục` left.
- **T4.2**: Tab pills: `padding: 0 8px` (giảm 2px mỗi bên), `min-height: 36px`
  (giảm 4px).
- **T4.3**: Verify mobile graph ở tab "Biên" có render đúng khi active.

## 3. Validation

Sau mỗi phase:
- Re-shoot 36 screenshots (light + dark × 6 res × 3 routes)
- Visual diff: side-panel giảm 1 layer card, eyebrow count giảm
- Mono numbers align tốt hơn ở eval chart
- Combined page col 1 fill
- Build pass, 40 vitest pass, ESLint 0 errors, prettier clean

## 4. Out of Scope (chưa làm)

- Phase T3.3 (phase quick-jump discovery) — để v2
- Real images / hero assets (skill §4.8) — app không có landing page
- Comprehensive animation overhaul (đã cap motion 3/10)
- Light/dark mode toggle UI (đã có system auto)
- Piece SVG redesign (đang dùng text glyph — đủ editorial)
