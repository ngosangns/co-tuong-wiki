# Plan: Redesign Giao Diện Responsive (Mobile + Desktop)

## Tóm tắt

Thiết kế lại layout và UX của `apps/web` để đạt chuẩn responsive thực sự:
- **Desktop**: 3 cột cân đối, tận dụng không gian rộng, giảm dead space
- **Tablet**: 2 cột rõ ràng, board và info song song
- **Mobile**: 1 cột gọn gàng, touch target lớn, sidebar có overlay, controls sticky, hỗ trợ swipe

## Phân tích vấn đề hiện tại

1. **Desktop**:
   - Board panel bị ép giữa 2 sidebar, khoảng trống nhiều khi màn hình >1400px
   - `lesson-topbar` nằm riêng grid-row, chiếm chiều cao không cần thiết
   - Graph và principles panel chưa tận dụng hết chiều cao viewport

2. **Mobile**:
   - `.app-shell` chuyển `display: block` tại 760px, không có cấu trúc rõ ràng
   - Sidebar chỉ toggle `display: none/flex`, thiếu overlay/backdrop -> dễ bấm nhầm
   - Board controls (`previous`/`next`) quá nhỏ, font-size 13px, min-height 36px < 44px chuẩn touch
   - Bàn cờ `width: min(100%, 460px)` có thể nhỏ hơn 300px trên điện thoại hẹp do padding 18px
   - MoveGraph hiển thị full-width liên tục, chiếm quá nhiều vertical space
   - Combined page graph chiếm `54svh`, không có cách thu gọn trên mobile

3. **Chung**:
   - Breakpoint chỉ có 1180px và 760px, thiếu breakpoint trung gian (980px)
   - Input search chưa có `font-size: 16px` -> iOS sẽ auto-zoom
   - Không có swipe gesture trên board
   - Thiếu `overscroll-behavior` cho sidebar

## Giải pháp & Thiết kế đề xuất

### Layout Grid mới

```
Desktop (>1180px):
  | Library (280px fixed) | Main (flex) | Info (380px fixed) |
  Trong Main: Board (center) + Controls + Comment
  Trong Info: Graph (flex-grow) + Principles (collapse)

Tablet (760px - 1180px):
  | Library (260px) | Main+Info (flex, stacked vertically) |

Mobile (<760px):
  | Mobile Header (toggle + title) |
  | Board Stage (full-width, max board size) |
  | Tab Bar: Board | Graph | Info |
  | Active Tab Content |
  | Sticky Bottom Controls |
```

### Chi tiết thay đổi

#### 1. App Shell & Sidebar
- Giữ `grid` ở desktop, chuyển `flex` ở mobile với `flex-direction: column`
- Library panel: fixed width 280px desktop, slide-in overlay mobile với backdrop `rgba(0,0,0,0.5)`
- Thêm `.sidebar-overlay` element trong App.vue
- `mobile-nav-toggle` đổi thành header bar rõ ràng hơn

#### 2. Board & Controls
- `.xiangqi-board-shell`: `width: min(100%, 520px)` trên mobile, giảm gap xuống 2px
- `.board-controls`: chuyển thành `position: sticky; bottom: 0` trên mobile với nền mờ `backdrop-filter: blur(8px)`
- Button previous/next: min-height 44px, font-size 15px trên mobile
- Thêm touch event listeners (swipe left/right) trong `XiangqiBoard.vue` để next/previous move

#### 3. MoveGraph
- Desktop: giữ nguyên trong `.board-side-panel`
- Mobile: chuyển vào tab "Graph", ẩn mặc định
- Combined page: graph có thể collapse trên mobile bằng `isCollapsible`

#### 4. Lesson Panel (Principles)
- Desktop: scrollable, max-height 60vh
- Mobile: đưa vào tab "Info", hiển thị dạng accordion

#### 5. Combined Page
- Desktop: 3 cột nhưng với `gap: 12px` và `padding: 16px`
- Mobile: tabs cho Board | Inspector | Graph

#### 6. Responsive Breakpoints mới
- `>1180px`: 3 cột desktop
- `760px - 1180px`: 2 cột tablet
- `<760px`: 1 cột mobile với tabs
- Thêm `480px`: micro-mobile adjustments (board max 100%, smaller padding)

#### 7. Accessibility & Touch
- Search input: `font-size: 16px` trên mobile
- Touch target tối thiểu 44px cho mọi interactive element
- `overscroll-behavior: contain` cho sidebar và graph canvas
- `prefers-reduced-motion`: giữ nguyên đã có, mở rộng thêm cho tab transitions

## Các Phase thực hiện

### Phase 1: Foundation & Sidebar (style.css + App.vue)
- Refactor `.app-shell` grid system
- Thêm sidebar overlay cho mobile
- Tối ưu `.mobile-nav-toggle`
- Điều chỉnh padding/spacing cho mobile-first
- Thêm breakpoint 480px

### Phase 2: Board & Controls (style.css + XiangqiBoard.vue + App.vue)
- Tăng board size trên mobile
- Sticky bottom controls với backdrop blur
- Swipe gestures cho board (touchstart/touchend)
- Tăng touch target buttons

### Phase 3: Panels & Tabs (App.vue + CombinedLessonPage.vue + style.css)
- Mobile tab navigation (Board | Graph | Info)
- MoveGraph responsive: collapsible trên mobile
- Principles panel accordion behavior
- Combined page tab layout

### Phase 4: Polish & Testing
- Animation transitions cho tab switching
- Test trên các viewport: 375px, 768px, 1024px, 1440px, 1920px
- Fix any visual regression
- Kiểm tra dark/light mode

## File cần sửa

| File | Thay đổi |
|------|----------|
| `apps/web/src/style.css` | Refactor layout grid, responsive breakpoints, mobile spacing, sticky controls, board sizing |
| `apps/web/src/App.vue` | Thêm sidebar overlay, mobile tab navigation, restructure template cho mobile |
| `apps/web/src/components/XiangqiBoard.vue` | Thêm swipe gesture handlers |
| `apps/web/src/components/CombinedLessonPage.vue` | Thêm mobile tabs cho board/inspector/graph |
| `apps/web/src/components/MoveGraph.vue` | Đảm bảo `isCollapsible` hoạt động tốt trên mobile |

## Tiêu chí hoàn thành

- [ ] Desktop (1440px+): 3 cột cân đối, không có khoảng trống thừa >100px
- [ ] Tablet (768px-1024px): 2 cột, board và graph hiển thị rõ, không bị overflow-x
- [ ] Mobile (375px-428px): 
  - Board >= 300px width
  - Controls sticky bottom, dễ bấm (44px+)
  - Sidebar có overlay, slide-in smooth
  - Graph và Principles nằm trong tab, không chiếm space liên tục
- [ ] Swipe left/right trên board chuyển nước đi
- [ ] Search input không gây zoom trên iOS Safari
- [ ] Không regression trên dark/light mode
- [ ] `prefers-reduced-motion` vẫn hoạt động
