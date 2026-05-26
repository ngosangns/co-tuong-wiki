"""
Parser cho các file thư viện endgame của ChessQ (*.eglib, *.eplib).

Đây là định dạng text đơn giản từ các sách cờ tướng nổi tiếng:
- 杀局谱 (Sát cục phổ)
- 基本杀法 (Kỹ thuật sát cơ bản)
- 适情雅趣360 (Thích tình nhã thú - sách danh tiếng)
- 车马专集, 梦入神机杀局, final_book, v.v.

Format mỗi dòng (thường):
    Tên thế cờ|FEN đầy đủ

Một số file có thêm cột moves (ICCS), parser sẽ cố gắng xử lý cả 2 trường hợp.

Số lượng cực lớn:
- 杀局谱.eglib ~6k
- final_book.eglib ~3.7k
- ...

Rất phù hợp tạo hàng nghìn lesson "Thế sát" / "Tàn cuộc" / "Kỹ thuật giết tướng".

Không cần thư viện ngoài.
"""

from __future__ import annotations

import re
import uuid
from dataclasses import dataclass
from pathlib import Path
from typing import List, Optional


@dataclass
class EglibPosition:
    fen: str
    title: str
    source_file: str
    book_or_theme: Optional[str] = None   # ví dụ: "基本杀法", "适情雅趣360"


def _clean_title(raw: str) -> str:
    """Làm sạch title (loại bỏ rác encoding, ký tự lạ từ file cũ)."""
    # Loại bỏ control chars và garbage
    t = re.sub(r'[\x00-\x1f\x7f-\x9f]', '', raw)
    t = t.strip()
    # Nếu quá dài hoặc toàn rác → fallback
    if len(t) > 80 or not t or len([c for c in t if c.isalpha() or '\u4e00' <= c <= '\u9fff']) < 2:
        return "Endgame study"
    return t


def _extract_fen(raw: str) -> Optional[str]:
    """Lấy FEN từ phần thứ 2 (hoặc thứ 3 nếu có moves)."""
    # FEN thường bắt đầu bằng số hoặc chữ cái quân cờ
    candidates = raw.strip().split()
    for c in candidates:
        if "/" in c and len(c) > 20:   # FEN Xiangqi thường rất dài
            # Cắt phần thừa sau FEN (nếu có)
            fen = c.split()[0] if " " in c else c
            # Chuẩn hóa: chỉ lấy đến trước " - " hoặc phần extra
            if " - " in fen:
                fen = fen.split(" - ")[0]
            return fen.strip()
    return None


def parse_eglib_line(line: str, source_file: str, book_name: str) -> Optional[EglibPosition]:
    line = line.strip()
    if not line or line.startswith("#") or "|" not in line:
        return None

    parts = [p.strip() for p in line.split("|") if p.strip()]
    if len(parts) < 2:
        return None

    title_raw = parts[0]
    fen_candidate = parts[1] if len(parts) > 1 else parts[0]

    fen = _extract_fen(fen_candidate)
    if not fen or "/" not in fen or len(fen) < 15:
        return None

    title = _clean_title(title_raw)
    if title == "Endgame study":
        # Thử lấy từ FEN hoặc đặt tên theo sách
        title = f"{book_name} - {fen.split()[0][:20]}..."

    return EglibPosition(
        fen=fen,
        title=title,
        source_file=source_file,
        book_or_theme=book_name,
    )


def collect_eglib_files(base_path: str | Path) -> List[Path]:
    base = Path(base_path)
    if not base.exists():
        return []
    files = []
    for ext in ("*.eglib", "*.eplib"):
        files.extend(base.rglob(ext))
    return sorted(files)


def process_eglib_directory(
    base_path: str | Path,
    limit: Optional[int] = None,
) -> List[dict]:
    """
    Quét thư mục chứa *.eglib/*.eplib, tạo lesson dicts (chủ yếu là position-only).
    """
    base = Path(base_path)
    files = collect_eglib_files(base)
    if not files:
        print(f"No .eglib/.eplib found under {base}")
        return []

    lessons: List[dict] = []
    processed = 0
    errors = 0

    for fpath in files:
        book_name = fpath.stem  # ví dụ: "基本杀法", "杀局谱"
        print(f"Processing eglib: {fpath.name} (book: {book_name})")

        try:
            text = fpath.read_text(encoding="utf-8", errors="ignore")
        except Exception as e:
            print(f"  Read error: {e}")
            continue

        lines = text.splitlines()
        print(f"  {len(lines)} lines found")

        for line in lines:
            if limit and processed >= limit:
                break

            pos = parse_eglib_line(line, str(fpath), book_name)
            if not pos:
                errors += 1
                continue

            processed += 1

            # Category theo tên sách/folder
            cat = "Tàn cuộc"
            bn = book_name.lower()
            if "杀" in book_name or "杀法" in book_name or "杀局" in book_name:
                cat = "Thế sát"
            elif "车马" in book_name or "车炮" in book_name:
                cat = "Tàn cuộc"
            elif "适情雅趣" in book_name or "yaqu" in bn:
                cat = "Ván chọn lọc / Danh tác"

            lesson = {
                "id": str(uuid.uuid4()),
                "title": pos.title,
                "category": cat,
                "difficulty": "Trung cấp",
                "source": {
                    "type": "eglib",
                    "file": pos.source_file,
                    "book": book_name,
                },
                "initialFen": pos.fen,
                "lines": [
                    {
                        "id": str(uuid.uuid4()),
                        "title": "Phân tích / Kỹ thuật",
                        "moves": [],
                    }
                ],
            }
            lessons.append(lesson)

            if len(lessons) % 500 == 0:
                print(f"    ... {len(lessons)} lessons so far")

            if limit and len(lessons) >= limit:
                break

        if limit and len(lessons) >= limit:
            break

    print(f"Processed {len(lessons)} positions from eglib files (skipped {errors} bad lines)")
    return lessons
