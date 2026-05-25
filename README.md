# Cờ Tướng Wiki

Trang web học cờ tướng bằng wiki tương tác: bài học, thế cờ, cạm bẫy khai cuộc và nước đánh hiển thị trực tiếp trên bàn cờ.

## Stack

- Frontend: Vue 3 + TypeScript, Vite, Tailwind CSS, Lucide icons
- Backend: Go `net/http`
- Data: JSON lesson catalog served by the Go API

## Scripts

```sh
npm --prefix apps/web install
npm run dev:api
npm run dev:web
npm run build:web
npm run test:api
```

The API defaults to `http://127.0.0.1:8090`. If port `8090` is already in use, run the API with another port and point the web app at it:

```sh
PORT=8091 npm run dev:api
VITE_API_URL=http://127.0.0.1:8091 npm run dev:web
```

## Structure

```text
apps/
  api/  # Go backend for lessons, categories, and position analysis
  web/  # Vue frontend learning experience
```

## Lát Cắt Hiện Tại

Ứng dụng hiện có hệ thống bài học cờ tướng tương tác, gồm:

- Bàn cờ 9x10 hiển thị trạng thái theo từng nước.
- Steps nổi có thể bấm để tua lại/tua tới.
- Các nhóm bài khai cuộc, trung cuộc, tàn cuộc và thế sát.
- Câu hỏi lựa chọn nước đi kèm phản hồi.
- Backend API cung cấp lesson catalog/detail và đánh giá vị trí qua engine UCI được cấu hình bằng `ENGINE_PATH`.
