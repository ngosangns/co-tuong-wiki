"""
Parser cho định dạng XQF (định dạng binary phổ biến của phần mềm cờ tướng Trung Quốc).

Sử dụng thư viện cchess (https://pypi.org/project/cchess/).

Cài đặt:
    python3 -m pip install cchess --break-system-packages

XQF files trong data/external/ChessQ/gamebooks/ là nguồn rất tốt cho:
- Thế sát (mating puzzles)
- Tàn cuộc
- Kỹ thuật giết tướng

Hỗ trợ:
- Đọc FEN khởi tạo
- Danh sách nước đi (ICCS)
- Kết quả ván
- Tên file làm tiêu đề
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field
from pathlib import Path
from typing import List, Optional, Tuple

# Lazy import — chỉ ném lỗi khi thực sự dùng XQF parser
_cchess_read = None

def _get_read_from_xqf():
    global _cchess_read
    if _cchess_read is None:
        try:
            from cchess import read_from_xqf as _read
            _cchess_read = _read
        except ImportError as e:
            raise ImportError(
                "Thiếu thư viện cchess (cần cho XQF).\n"
                "Cài đặt: python3 -m pip install cchess --break-system-packages"
            ) from e
    return _cchess_read


@dataclass
class XQFGMove:
    side: str  # 'red' or 'black'
    from_file: int
    from_rank: int
    to_file: int
    to_rank: int
    notation: str
    comment: str = ""
    move_number: int = 0


@dataclass
class XQFGame:
    title: str
    source_file: str
    result: str = "*"
    initial_fen: Optional[str] = None
    movelist: List[XQFGMove] = field(default_factory=list)
    subcategory: Optional[str] = None   # ví dụ: "象棋残局杀势"


def iccs_to_coords(iccs: str) -> Tuple[Tuple[int, int], Tuple[int, int]]:
    """
    Chuyển ICCS (ví dụ 'g6e7') sang (file, rank) 0-based của project.
    file: a=0 ... i=8
    rank: 0-9 (phù hợp với hệ tọa độ lessons.json)
    """
    if len(iccs) != 4:
        raise ValueError(f"Invalid ICCS move: {iccs}")
    f1 = ord(iccs[0].lower()) - ord('a')
    r1 = int(iccs[1])
    f2 = ord(iccs[2].lower()) - ord('a')
    r2 = int(iccs[3])
    if not (0 <= f1 <= 8 and 0 <= r1 <= 9 and 0 <= f2 <= 8 and 0 <= r2 <= 9):
        raise ValueError(f"ICCS out of range: {iccs}")
    return (f1, r1), (f2, r2)


def _detect_first_side_from_fen(fen: str) -> str:
    """FEN Xiangqi: phần cuối trước ' - ' hoặc sau placement có 'w'/'b'."""
    parts = fen.split()
    if len(parts) >= 2:
        side = parts[1].lower()
        if side in ('w', 'r', 'red'):
            return 'red'
        if side in ('b', 'black'):
            return 'black'
    return 'red'  # default an toàn cho nhiều puzzle endgame


def parse_xqf(path: str | Path) -> Optional[XQFGame]:
    """
    Parse một file .xqf thành XQFGame.
    Trả về None nếu file lỗi hoặc không có nước đi.
    """
    path = Path(path)
    if not path.exists():
        return None

    try:
        read_from_xqf = _get_read_from_xqf()
        game = read_from_xqf(str(path))
    except Exception as exc:
        print(f"  [XQF] read error {path.name}: {exc}")
        return None

    # Lấy FEN
    try:
        initial_fen = game.init_board.to_fen()
    except Exception:
        initial_fen = None

    # Lấy danh sách nước (ưu tiên main line)
    iccs_lines = []
    try:
        raw = game.dump_iccs_moves()
        if raw and isinstance(raw, list):
            # Có thể là list of lists khi có biến
            if raw and isinstance(raw[0], list):
                iccs_lines = raw[0]
            else:
                iccs_lines = raw
    except Exception:
        iccs_lines = []

    if not iccs_lines:
        return None

    # Xác định bên đi trước
    first_side = _detect_first_side_from_fen(initial_fen or "")

    moves: List[XQFGMove] = []
    for i, iccs in enumerate(iccs_lines):
        if not isinstance(iccs, str) or len(iccs) != 4:
            continue
        try:
            (f1, r1), (f2, r2) = iccs_to_coords(iccs)
        except Exception:
            continue

        move_number = i + 1
        side = first_side if i % 2 == 0 else ('black' if first_side == 'red' else 'red')

        moves.append(XQFGMove(
            side=side,
            from_file=f1,
            from_rank=r1,
            to_file=f2,
            to_rank=r2,
            notation=iccs,
            move_number=move_number,
        ))

    if not moves:
        return None

    # Title: dùng tên file (sau này có thể dịch hoặc làm đẹp)
    stem = path.stem
    title = stem.replace("_", " ").replace("-", " ")

    # Result (nếu có)
    result = "*"
    try:
        if game.info and "result" in game.info:
            result = game.info["result"]
    except Exception:
        pass

    # Subcategory từ thư mục cha (ví dụ: 象棋残局杀势)
    parent = path.parent.name
    subcategory = parent if parent else None

    return XQFGame(
        title=title,
        source_file=str(path),
        result=result,
        initial_fen=initial_fen,
        movelist=moves,
        subcategory=subcategory,
    )


def collect_xqf_files(base_path: str | Path) -> List[Path]:
    """Tìm tất cả file .xqf dưới thư mục."""
    base = Path(base_path)
    if not base.exists():
        return []
    return sorted(base.rglob("*.xqf"))


def process_xqf_directory(
    base_path: str | Path,
    limit: Optional[int] = None,
    category_hint: Optional[str] = None,
) -> List[dict]:
    """
    Quét thư mục chứa .xqf, parse và trả về list lesson dict (gần schema lessons.json).
    Không ghi file — caller sẽ quyết định output.
    """
    import uuid

    files = collect_xqf_files(base_path)
    if limit:
        files = files[:limit]

    lessons: List[dict] = []
    errors = 0

    for i, fpath in enumerate(files):
        print(f"[{i+1}/{len(files)}] XQF: {fpath.relative_to(base_path) if base_path else fpath.name}")

        game = parse_xqf(fpath)
        if not game or not game.movelist:
            errors += 1
            continue

        move_count = len(game.movelist)

        # Category gợi ý
        if category_hint:
            category = category_hint
        elif game.subcategory and "残局" in game.subcategory:
            category = "Tàn cuộc"
        elif game.subcategory and ("杀" in game.subcategory or "杀势" in game.subcategory):
            category = "Thế sát"
        else:
            category = "Tàn cuộc"

        # Difficulty thô
        if move_count <= 6:
            difficulty = "Nhập môn"
        elif move_count <= 12:
            difficulty = "Trung cấp"
        else:
            difficulty = "Cao cấp"

        # Chuyển moves
        lesson_moves = []
        for m in game.movelist:
            lesson_moves.append({
                "id": str(uuid.uuid4()),
                "side": m.side,
                "from": {"file": m.from_file, "rank": m.from_rank},
                "to": {"file": m.to_file, "rank": m.to_rank},
                "comment": m.comment or "",
            })

        lesson = {
            "id": str(uuid.uuid4()),
            "title": game.title,
            "category": category,
            "difficulty": difficulty,
            "source": {
                "type": "xqf",
                "file": game.source_file,
                "result": game.result,
            },
            "lines": [
                {
                    "id": str(uuid.uuid4()),
                    "title": "Biến chính",
                    "moves": lesson_moves,
                }
            ],
        }
        if game.initial_fen:
            lesson["initialFen"] = game.initial_fen

        lessons.append(lesson)
        print(f"  OK: {len(lesson_moves)} moves → {category} / {difficulty}")

    if errors:
        print(f"  (Bỏ qua {errors} file lỗi hoặc rỗng)")

    return lessons
