# Cờ Tướng Wiki

Trang web học cờ tướng bằng wiki tương tác: bài học, thế cờ, cạm bẫy khai cuộc và nước đánh hiển thị trực tiếp trên bàn cờ.

## Stack

- Frontend: Vue 3 + TypeScript, Vite, Tailwind CSS, Lucide icons
- Backend: Go `net/http`
- Data: JSON lesson catalog served by the Go API

## Scripts

```sh
task install
task dev
task build
task test
task test:web
```

The API defaults to `http://127.0.0.1:8090`. If port `8090` is already in use, run the API with another port and point the web app at it:

```sh
PORT=8091 task dev:api
VITE_API_URL=http://127.0.0.1:8091 task dev:web
```

Useful focused tasks:

```sh
task dev:api
task dev:web
task build:web
task test:api
task test:web
task test:web:watch
task lessons:combined:dry-run
task lessons:combined
task build:fairy-stockfish
```

### Multi-engine analysis

The API defaults to a single engine (Pikafish) for `/api/analyze`. To enable
Fairy-Stockfish as a secondary engine, build it once with
`task build:fairy-stockfish ~/path/to/Fairy-Stockfish-src` and point the
server at the resulting binary:

```sh
export ENGINE_FAIRY_PATH="$(pwd)/bin/fairy-stockfish"
task dev:api
```

With both engines configured, the API can fan out analyses:

```sh
curl -s -X POST http://127.0.0.1:8090/api/analyze \
  -H "Content-Type: application/json" \
  -d '{"fen":"rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1","sideToMove":"red","engines":["pikafish","fairy-stockfish"]}'
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
