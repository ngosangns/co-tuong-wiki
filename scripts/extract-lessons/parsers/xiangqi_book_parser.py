"""
Improved extractor for the "中国象棋入门到提高" book.

Focus on extracting concrete, high-value examples:
- Full real games from chapter 09 with move sequences + analysis
- Notable tactic / killing / endgame examples from other chapters

The book uses standard Chinese Xiangqi notation:
  炮二平五, 马 8 进 7, 车一平二, 兵七进一, etc.

We convert these to the project's (file 0-8, rank 0-9) coordinate system where possible.
"""

from __future__ import annotations

import re
import uuid
from pathlib import Path
from typing import List, Dict, Any, Optional, Tuple


CHAPTER_INFO = {
    "01-棋子价值": {"title": "Giá trị quân cờ và vận dụng quân lực", "category": "Lý thuyết cơ bản", "diff": "Nhập môn"},
    "02-基本战术": {"title": "Chiến thuật cơ bản", "category": "Chiến thuật", "diff": "Nhập môn"},
    "03-基本杀法": {"title": "Sát pháp cơ bản", "category": "Thế sát", "diff": "Trung cấp"},
    "04-实用残局": {"title": "Tàn cuộc thực dụng", "category": "Tàn cuộc", "diff": "Trung cấp"},
    "05-开局原理": {"title": "Nguyên lý khai cuộc", "category": "Khai cuộc", "diff": "Trung cấp"},
    "06-常见开局": {"title": "Các khai cuộc phổ biến", "category": "Khai cuộc", "diff": "Trung cấp"},
    "07-中局策略": {"title": "Chiến lược trung cuộc", "category": "Trung cuộc", "diff": "Cao cấp"},
    "08-练习题": {"title": "Bài tập", "category": "Bài tập", "diff": "Trung cấp"},
    "09-实战对局": {"title": "Pháo Đầu đối Bình Phong Mã - Ván đấu thực tế có phân tích", "category": "Ván chọn lọc", "diff": "Cao cấp"},
}


def chinese_file_to_index(chinese_num: str, side: str = 'red') -> int:
    """
    Convert Chinese file number (一二三四五六七八九 or Arabic) to 0-8 index.
    In standard Chinese notation:
    - Red's files are counted from their right (file 1 is Red's rightmost file from their view).
    - Our project: file 0 = leftmost from Red's perspective (a-file for Red).
    """
    map_cn = {'一': 0, '二': 1, '三': 2, '四': 3, '五': 4, '六': 5, '七': 6, '八': 7, '九': 8}
    if chinese_num.isdigit():
        num = int(chinese_num) - 1
        return max(0, min(8, num))
    return map_cn.get(chinese_num, 4)


DIRECTION = {'进': 'advance', '退': 'retreat', '平': 'flat'}


def parse_chinese_move(move_str: str, current_side: str = 'red', board_context: Optional[dict] = None) -> Optional[dict]:
    """
    Very basic parser for common Chinese Xiangqi moves.
    Returns dict with from/to if we can make a reasonable guess, else None.
    This is heuristic — full legal move generation would be better but heavy.
    """
    move_str = move_str.strip()
    if not move_str:
        return None

    # Normalize Arabic numbers sometimes used for black
    move_str = re.sub(r'(\d)', lambda m: { '1':'一','2':'二','3':'三','4':'四','5':'五','6':'六','7':'七','8':'八','9':'九' }.get(m.group(1), m.group(1)), move_str)

    # Pattern: Piece + File + Direction + Target
    match = re.match(r'([车马炮兵卒相象仕士将帅])?([一二三四五六七八九\d]+)(进|退|平)([一二三四五六七八九\d]+)', move_str)
    if not match:
        return None

    piece, from_file, direction, target = match.groups()
    piece = piece or '?'
    from_f = chinese_file_to_index(from_file, current_side)

    # Very rough rank handling (we don't have full board state)
    # For demonstration we create plausible moves; real usage should use a proper engine or board simulator.
    to_f = chinese_file_to_index(target, current_side)

    # Approximate rank change
    rank_delta = 1 if direction == '进' else (-1 if direction == '退' else 0)
    # Red moves "up" in our coordinate (increasing rank), Black moves "down"
    if current_side == 'black':
        rank_delta = -rank_delta

    # We don't know exact starting rank, so we produce a "suggested" move with comment containing original notation.
    # For the purpose of this extractor we store the original notation as comment and provide best-effort coordinates.
    return {
        "original": move_str,
        "from": {"file": from_f, "rank": 4 if current_side == 'red' else 5},   # rough center guess
        "to": {"file": to_f, "rank": 4 + rank_delta if current_side == 'red' else 5 - rank_delta},
        "comment": f"原文: {move_str}"
    }


def extract_game_from_chapter09(md_path: Path) -> Optional[Dict[str, Any]]:
    """Extract the main annotated game from chapter 09 as a rich lesson."""
    try:
        text = md_path.read_text(encoding="utf-8", errors="ignore")
    except Exception:
        return None

    # Find the move list block
    start = text.find("具体走法如下：")
    if start == -1:
        return None

    # Extract lines that look like moves until we hit analysis or end
    lines = text[start:].splitlines()
    moves = []
    for line in lines[1:30]:   # safety limit
        m = re.match(r'^\d+\.\s+(.+?)\s{2,}(.+)$', line.strip())
        if m:
            red_mv, black_mv = m.groups()
            moves.append({"side": "red", "text": red_mv.strip()})
            moves.append({"side": "black", "text": black_mv.strip()})
        if "这一段的关键决策" in line or "当前局面判断" in line:
            break

    if not moves:
        return None

    lesson_moves = []
    side = 'red'
    for m in moves:
        parsed = parse_chinese_move(m["text"], m["side"])
        if parsed:
            lesson_moves.append({
                "id": str(uuid.uuid4()),
                "side": m["side"],
                "from": parsed["from"],
                "to": parsed["to"],
                "comment": parsed["comment"] + " | " + m["text"]
            })

    lesson = {
        "id": str(uuid.uuid4()),
        "title": "炮二平五对屏风马 - 实战对局完整复盘（第九章示例）",
        "category": "Ván chọn lọc",
        "difficulty": "Cao cấp",
        "source": {
            "type": "md-book",
            "file": str(md_path),
            "book": "中国象棋入门到提高",
            "chapter": "09-实战对局"
        },
        "lines": [
            {
                "id": str(uuid.uuid4()),
                "title": "主变 + 关键分析",
                "moves": lesson_moves
            }
        ],
        "summary": "Ván đấu minh họa đầy đủ từ khai cuộc đến trung cuộc và công vương, được phân tích chi tiết trong sách."
    }

    # Try to find a starting FEN hint (the book often shows a diagram after a few moves)
    return lesson


def process_xiangqi_book(base_path: str | Path, limit: int | None = None) -> List[Dict[str, Any]]:
    """Main entry point. Returns richer lessons than the first version."""
    base = Path(base_path)
    book_dir = base / "xiangqi-book"
    lessons: List[Dict[str, Any]] = []

    if not book_dir.exists():
        return lessons

    # First, the improved real game lesson from chapter 09
    game_chapter = book_dir / "09-实战对局.md"
    if game_chapter.exists():
        rich_game = extract_game_from_chapter09(game_chapter)
        if rich_game:
            lessons.append(rich_game)

    # Then the chapter-level theory lessons (lighter weight)
    md_files = sorted(book_dir.glob("*.md"))
    for md in md_files:
        stem = md.stem
        if not re.match(r"\d{2}-", stem):
            continue
        if stem == "09-实战对局" and lessons:   # already handled above with richer version
            continue

        info = CHAPTER_INFO.get(stem, {"title": stem, "category": "Lý thuyết", "diff": "Trung cấp"})

        lesson = {
            "id": str(uuid.uuid4()),
            "title": info["title"],
            "category": info["category"],
            "difficulty": info["diff"],
            "source": {
                "type": "md-book",
                "file": str(md),
                "book": "中国象棋入门到提高",
                "chapter": stem
            },
            "lines": [{"id": str(uuid.uuid4()), "title": "Nội dung lý thuyết", "moves": []}],
            "summary": f"Kiến thức cốt lõi từ chương {stem}."
        }
        lessons.append(lesson)

        if limit and len(lessons) >= limit:
            break

    # Note: Deeper per-example extraction (dozens of small tactics/endgame examples
    # scattered in chapters 02, 03, 04, 06...) would require a full Xiangqi move
    # parser + context tracking. Current focus is on the highest-value concrete
    # contribution (full annotated game in ch.09) + clean theory overviews.
    return lessons
