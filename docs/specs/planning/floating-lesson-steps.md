---
title: "Lesson Choice And Move Graph"
description: "Current layout rules for lesson choices, move graph navigation, and model evaluation."
type: spec
status: implemented
tags: ["lesson-player", "move-graph", "xiangqi-ui"]
source_paths:
  - "apps/web/src/App.vue"
  - "apps/web/src/components/MoveGraph.vue"
  - "apps/web/src/composables/useLessonPlayer.ts"
  - "apps/web/src/engine/fen.ts"
  - "apps/api/data/lessons/lessons.json"
  - "apps/web/src/style.css"
related:
  - "./sync-lesson-board-state.md"
  - "./integrate-xiangqi-engine-evaluation.md"
---

# Lesson Choice And Move Graph

## Meta

- Trạng thái: implemented
- Phạm vi: lesson playhead controls, choice prompt placement, move graph sidebar layout
- Nguồn code: `apps/web/src/App.vue`, `apps/web/src/components/MoveGraph.vue`, `apps/web/src/composables/useLessonPlayer.ts`, `apps/web/src/engine/fen.ts`, `apps/api/data/lessons/lessons.json`, `apps/web/src/style.css`
- Links: [Sync Lesson Board State](./sync-lesson-board-state.md), [Integrate Xiangqi Engine Evaluation](./integrate-xiangqi-engine-evaluation.md)

## Bối Cảnh

Lesson player giữ một playhead duy nhất qua `useLessonPlayer`. Board, counter, current move, choice prompt, move graph và model evaluation đều đọc từ nguồn này để tránh lệch trạng thái. Mỗi lesson có thể dùng bàn cờ khai cuộc mặc định hoặc khai báo `initialFen` riêng; board replay luôn bắt đầu từ position của lesson trước khi áp các nước đang visible.

Các câu hỏi lựa chọn là hành động trực tiếp trên vị trí bàn cờ, nên hiển thị phía trên bàn cờ và chỉ xuất hiện đúng ply trước khi người học cần chọn. Move graph là navigation/overview của các biến, nên nằm trong right sidebar phía trên model evaluation để người học có thể đổi nhánh và theo dõi phân tích trong cùng một cột.

## Layout Hiện Tại

`App.vue` chia màn hình chính thành:

- `library-panel`: brand, theme switcher, search, topic list và lesson child items dưới topic active.
- `lesson-topbar`: wiki summary, tags và principles.
- `board-panel`: choice prompt đúng thời điểm, bàn cờ và nút previous/next.
- `lesson-panel`: `MoveGraph`, sau đó là `EvaluationPanel`.

Lesson selection nằm trong left sidebar: mỗi topic là parent item, các bài học là child items dưới topic đang active. Child lesson đang mở có active state riêng, và khi chọn bài thì category parent cũng được đồng bộ active.

`MoveGraph` nằm ngay trước `EvaluationPanel` trong right sidebar. Vị trí này giữ graph gần đánh giá model, đồng thời không đẩy bàn cờ xuống thấp. Việc chọn biến hóa nằm trong graph branch labels thay vì dùng một tab list riêng.

## Choice Visibility

Choice visibility suy ra từ dữ liệu lesson:

- Lấy toàn bộ `choice.options[].moveId`.
- Tìm index nhỏ nhất của các move id đó trong mọi `lesson.lines[].moves`.
- Chỉ hiện câu hỏi khi `activeMoveIndex` bằng index đó, tức là board đang ở trạng thái ngay trước các nước lựa chọn.
- Sau khi người học chọn option, `chooseMove` chuyển sang line chứa move đó và advance tới move được chọn, làm câu hỏi ẩn đi.

## Move Graph Behavior

`MoveGraph` nhận `lines`, `activeLineId` và `activeMoveIndex`. Component không sở hữu playhead riêng.

Graph hiển thị theo dạng cây:

- Component dựng graph model từ từng line path. Mỗi node có graph key dựa trên ply, bên đi, tọa độ đi/đến và `move.id`; mỗi edge nối hai node liên tiếp trong line.
- Các node cùng thuộc shared prefix của các line được gom vào trunk `Chung`.
- Mỗi lesson line chỉ render phần branch sau shared prefix, nên không duplicate các node trước khi tách nhánh.
- Canvas dùng node-link graph: mỗi nước là một node fixed-size, mỗi biến là một hàng, và các cạnh SVG nối trunk với từng nhánh hoặc nối các bước kế tiếp trong cùng biến.
- Data contract: các nước giống nhau trong shared prefix của mọi line phải dùng cùng `move.id`; các nước sau điểm rẽ nhánh phải giữ `move.id` riêng để choice và node branch trỏ đúng biến. `choice.options[].moveId` phải trỏ đến move tồn tại trong một branch.
- Node active, past và future đều suy ra từ `activeLineId` + `activeMoveIndex`.
- Khi click shared node, graph giữ branch đang active nếu branch đó chứa node; khi click branch node, graph chuyển đúng line và ply của node đó.
- Khi playhead đổi, component gọi `scrollIntoView({ block: 'nearest', inline: 'center' })` sau `nextTick()` để node active luôn visible.

Graph shell tham khảo cách tổ chức của preview graph trong `../ns-workspace`: toolbar/stats phía trên, canvas nền chấm ở giữa, node selection trong canvas và details của node active ở dưới. Phiên bản này dùng Vue/CSS/SVG nội bộ thay vì thêm dependency render graph.

## Lesson Data Contract

Lesson data được backend load từ `apps/api/data/lessons/lessons.json` và được kiểm qua Go tests:

- Khai cuộc có thể bỏ `initialFen` để dùng bàn cờ chuẩn.
- Trung cuộc, thế sát và tàn cuộc dùng `initialFen` để mô tả đúng thế xuất phát thay vì replay giả từ khai cuộc.
- Mỗi move phải có quân đúng bên ở source square, không được đi vào quân cùng phe, và phải hợp luật cơ bản theo loại quân.
- Pháo khi ăn phải có đúng một ngòi; khi không ăn không được có quân chắn.
- Mã không được bị chặn chân; tượng không qua sông; sĩ và tướng ở trong cung.
- Sau mỗi move, hai tướng không được nhìn thẳng nhau.
- Choice option phải trỏ đến move id tồn tại trong lesson.

## Styling

Các class chính:

- `.graph-shell`
- `.graph-toolbar`
- `.graph-layout`
- `.graph-canvas`
- `.graph-map`
- `.graph-edges`
- `.graph-edge`
- `.graph-branch-label`
- `.graph-step`
- `.graph-step.active`
- `.graph-step.past`
- `.graph-step.future`
- `.graph-details`

`graph-canvas` scroll được theo cả hai hướng khi cây dài. `graph-map` có kích thước tính từ số bước dài nhất và số biến, nên cạnh SVG và node giữ đúng tọa độ khi người học scroll. Node có kích thước ổn định để click hoặc active state không làm layout shift. Future node dùng opacity/blur nhẹ để giảm độ nhấn nhưng vẫn đọc được.

## Kiểm Chứng

- `npm --prefix apps/web run build` pass.
- `go test ./apps/api/...` pass.
- Khi `activeMoveIndex = 0`, không có step active.
- Bấm `Nước kế`, node tương ứng active, board replay đúng nước và graph auto-scroll tới node active.
- Bấm node graph, counter và board chuyển tới đúng ply.
- Bấm branch label trong graph, active branch đổi theo line.
- Bấm choice option, active line/step nhảy tới move được chọn.
- Choice prompt nằm phía trên bàn cờ và chỉ hiển thị đúng step trước choice.
- Move graph nằm trong sidebar phía trên model evaluation.
