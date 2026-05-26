"""
Parser cho các bộ sưu tập vị trí FEN đơn (endgame patterns / tàn cục).

Nguồn chính hiện tại:
- data/external/xiokuai-chinese_chess/.../*.fen  (~1100+ files)

Các file FEN này thường chỉ chứa một dòng FEN (không có nước đi hay lời giải).
Phù hợp tạo lesson dạng "Mẫu hình tàn cuộc" / "Nhận diện thế cờ" để người học tự phân tích.

Mỗi FEN sẽ trở thành một lesson với:
- initialFen
- 1 line rỗng hoặc chỉ có comment hướng dẫn
- Category & title được sinh từ cấu trúc thư mục (rất có ý nghĩa phân loại)

Ví dụ thư mục:
  C 子力定式/双车兵类/B双车兵类-第03局.fen
  → "Mẫu hình tàn cuộc - 双车兵类"

Không phụ thuộc thư viện ngoài.
"""

from __future__ import annotations

import uuid
from dataclasses import dataclass
from pathlib import Path
from typing import List, Optional


@dataclass
class FenPosition:
    fen: str
    title: str
    source_file: str
    subcategory: Optional[str] = None   # ví dụ: "双车兵类"
    theme_folder: Optional[str] = None  # ví dụ: "C 子力定式"


def _clean_fen(text: str) -> str:
    """Lấy FEN sạch (bỏ khoảng trắng thừa, newline)."""
    return text.strip().split()[0] if text.strip() else ""


def _path_to_title_and_category(fpath: Path, base: Path) -> tuple[str, str, Optional[str]]:
    """
    Sinh title + category từ đường dẫn thư mục.
    Ưu tiên giữ nguyên tên Trung để sau này dịch.
    """
    rel = fpath.relative_to(base) if base else fpath
    parts = list(rel.parts[:-1])  # bỏ tên file
    filename = fpath.stem

    # Tìm theme từ path (bao gồm cả tên thư mục base nếu user quét thẳng vào leaf folder)
    all_path_parts = parts + [base.name] if base else parts
    theme = None
    for p in reversed(all_path_parts):
        if any(kw in p for kw in ("定式", "子力", "残局", "杀", "兵类", "车类", "马类", "炮")):
            theme = p
            break
    if not theme and parts:
        theme = parts[-1]

    # Title: ưu tiên lấy 1-2 folder cha có ý nghĩa (ví dụ 双车兵类)
    meaningful_parts = [p for p in all_path_parts if any(kw in p for kw in ("定式", "子力", "残局", "杀", "兵类", "车类", "马类", "炮"))]
    if meaningful_parts:
        title = f"{' / '.join(meaningful_parts[-2:])} - {filename}"
    elif theme:
        title = f"{theme} - {filename}"
    else:
        title = filename

    # Category
    category = "Tàn cuộc"
    path_str = str(rel).lower()
    if "定式" in path_str or "子力" in path_str or "mẫu hình" in path_str:
        category = "Mẫu hình tàn cuộc"
    elif "杀" in path_str or "thế sát" in path_str:
        category = "Thế sát"

    return title, category, theme


def parse_fen_file(path: str | Path, base_dir: Optional[Path] = None) -> Optional[FenPosition]:
    path = Path(path)
    if not path.exists() or not path.suffix.lower() == ".fen":
        return None

    try:
        raw = path.read_text(encoding="utf-8", errors="ignore")
        fen = _clean_fen(raw)
        if not fen or "/" not in fen:
            return None
    except Exception as e:
        print(f"  [FEN] read error {path.name}: {e}")
        return None

    base = base_dir or path.parent
    title, category, theme = _path_to_title_and_category(path, base)

    return FenPosition(
        fen=fen,
        title=title,
        source_file=str(path),
        subcategory=theme,
        theme_folder=theme,
    )


def collect_fen_files(base_path: str | Path) -> List[Path]:
    base = Path(base_path)
    if not base.exists():
        return []
    return sorted(base.rglob("*.fen"))


def process_fen_directory(
    base_path: str | Path,
    limit: Optional[int] = None,
) -> List[dict]:
    """
    Quét thư mục chứa *.fen, tạo lesson dicts (chỉ có initialFen + metadata).
    Rất phù hợp cho "pattern recognition" / "endgame studies".
    """
    base = Path(base_path)
    files = collect_fen_files(base)
    if limit:
        files = files[:limit]

    lessons: List[dict] = []
    errors = 0

    for i, fpath in enumerate(files):
        if i % 100 == 0:
            print(f"[{i+1}/{len(files)}] FEN: {fpath.relative_to(base) if base else fpath.name}")

        pos = parse_fen_file(fpath, base_dir=base)
        if not pos or not pos.fen:
            errors += 1
            continue

        lesson_id = str(uuid.uuid4())

        lesson = {
            "id": lesson_id,
            "title": pos.title,
            "category": "Mẫu hình tàn cuộc",   # hoặc lấy từ path_to_category nếu muốn đa dạng
            "difficulty": "Trung cấp",
            "source": {
                "type": "fen",
                "file": pos.source_file,
            },
            "initialFen": pos.fen,
            "lines": [
                {
                    "id": str(uuid.uuid4()),
                    "title": "Phân tích thế cờ",
                    "moves": [],   # FEN-only → không có chuỗi nước đi sẵn
                }
            ],
            # Gợi ý cho frontend: đây là vị trí cần người chơi tự tìm ý tưởng
            "choice": {
                "prompt": "Bạn thấy ý tưởng chính trong thế cờ này là gì?",
                "options": [
                    {
                        "moveId": "",
                        "label": "Tìm nước chiếu / ăn quân / chiếu bí",
                        "verdict": "info",
                        "feedback": "Hãy thử các nước chiếu, ăn, hoặc di chuyển Tướng/Tốt để tìm ý tưởng."
                    }
                ],
            },
        }

        lessons.append(lesson)

    if errors:
        print(f"  (Bỏ qua {errors} file FEN rỗng/lỗi)")

    print(f"Processed {len(lessons)} FEN positions from {base}")
    return lessons
