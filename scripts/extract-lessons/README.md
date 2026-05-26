# Extract Lessons from External Data

Pipeline chuyển đổi dữ liệu cờ tướng từ `data/external/` thành lesson format dùng được cho `apps/api/data/lessons/lessons.json`.

## Triết lý an toàn

- **Mặc định không bao giờ ghi trực tiếp** vào `lessons.json` production.
- Tất cả lệnh extract xuất ra **file staging** (`extracted_external_*.json`) dưới thư mục này.
- Dùng `merge_staging.py` để import an toàn (tự backup + dedup).

## Các nguồn external được hỗ trợ tốt nhất

| Source                  | Lệnh                                      | Số lượng (ước tính) | Loại dữ liệu                  |
|-------------------------|-------------------------------------------|---------------------|-------------------------------|
| ChessQ XQF              | `--source external-xqf`                   | 151                 | Thế sát + Tàn cục             |
| xiokuai FEN             | `--source external-fen`                   | ~1.100+             | Mẫu hình tàn cục (định thức)  |
| ChessQ EGLIB/EPLIB      | `--source external-eglib`                 | 10.000+             | Sách danh tiếng (杀局谱, 适情雅趣...) |
| WXF                     | `--source external-wxf`                   | Rất ít              | Ván mẫu (demo)                |
| Excel 象棋残局          | `--source external-excel`                 | ~8k+ types          | Tàn cục theo chất liệu        |
| Xiangqi Book (MD)       | (tự động trong `--source external`)       | 9 chương + game     | Lý thuyết + Ván thực chiến    |
| **Broad scan**          | `--source external`                       | Tất cả trên         | Chạy nhiều parser cùng lúc    |

## Cách sử dụng

### 1. Extract (tạo file staging)

```bash
cd scripts/extract-lessons

# Chạy một nguồn cụ thể
python main.py --source external-xqf --limit 50
python main.py --source external-fen --limit 100
python main.py --source external-eglib --limit 200

# Chạy tất cả nguồn external cùng lúc (khuyến nghị)
python main.py --source external --limit 300
```

Kết quả sẽ là các file:
- `extracted_external_xqf.json`
- `extracted_external_fen.json`
- `extracted_external_eglib.json`
- `extracted_external_xiangqi_book.json`
- ...

### 2. Review

Mở các file staging và kiểm tra:
- Title (nhiều title vẫn bằng tiếng Trung)
- Category & Difficulty
- FEN (nếu có)
- Số lượng moves

### 3. Merge an toàn vào production

```bash
# Xem trước (khuyến khích)
python merge_staging.py --input-glob "extracted_external_*.json" --dry-run

# Merge thật (tự động backup + dedup)
python merge_staging.py --input-glob "extracted_external_*.json"

# Merge có dịch title (tiếng Trung → tiếng Việt cơ bản)
python merge_staging.py --input-glob "extracted_external_*.json" --translate

# Merge có xử lý xung đột tương tác
python merge_staging.py --input extracted_external_eglib.json --interactive
```

**Lưu ý quan trọng khi merge:**
- Script luôn tạo file backup: `lessons.json.bak-YYYYMMDD-HHMMSS`
- Dedup theo title (normalized) + fingerprint nội dung
- Có thể bật `--translate` để tự động dịch một số thuật ngữ phổ biến

## Cấu trúc thư mục

```
scripts/extract-lessons/
├── main.py                    # Entry point extract
├── merge_staging.py           # Tool merge an toàn
├── parsers/
│   ├── dhtmlxq_parser.py
│   ├── xqf_parser.py
│   ├── fen_parser.py
│   ├── eglib_parser.py
│   ├── wxf_parser.py
│   └── xiangqi_book_parser.py
├── transformers/
│   └── lesson_builder.py
├── extracted_*.json           # Các file staging (không commit)
├── README.md                  # File này
└── extract.log                # Log cũ (nếu có)
```

## Yêu cầu

- Python 3.8+
- Cho XQF: `pip install cchess --break-system-packages`
- (Optional) pandas/openpyxl cho Excel (chưa hỗ trợ đầy đủ)

## Mẹo

- Luôn dùng `--limit` khi test lần đầu.
- Dùng `--dry-run` trước khi merge thật.
- Nếu title bị trùng nhiều, dùng `--interactive` để quyết định thủ công.
- Sau khi merge thành công, commit file `lessons.json` + file backup mới nhất (nếu cần).

## Tương lai (có thể mở rộng)

- Parser cho `chinese-chess-dataset/象棋残局.xlsx` (đã có cơ bản)
- Extract chi tiết hơn từ `xiangqi-book` (ví dụ positions nhỏ lẻ)
- Hỗ trợ dịch title chất lượng cao hơn (Google Translate)

## Cách thêm nguồn mới (extensibility)

1. Tạo file `parsers/your_source_parser.py`
   - Hàm chính: `process_your_source(base_path, limit=None) -> list[dict]`
   - Trả về list lesson theo đúng schema (id, title, category, difficulty, lines, optional initialFen, source...)
   - Luôn output an toàn (không ghi production).

2. Import và hook vào `main.py`:
   - Thêm vào danh sách import.
   - Thêm nhánh `if args.source == 'external-yourname'` (và optional auto-detect).
   - Thêm vào khối `--source external` broad scan.

3. Cập nhật:
   - argparse choices
   - docstring và README.md (phần bảng nguồn + ví dụ lệnh)
   - `merge_staging.py` nếu cần logic dedup/translate đặc biệt.

Ví dụ đã làm: XQF, FEN, EGLIB, WXF, Excel, XiangqiBook.

---

Pipeline này được xây dựng để an toàn, lặp lại được và dễ mở rộng khi có thêm data external mới.