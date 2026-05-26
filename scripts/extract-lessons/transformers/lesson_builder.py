"""
Chuyển đổi DhtmlXQGame → Lesson JSON theo schema của dự án.
"""

import uuid
from parsers.dhtmlxq_parser import DhtmlXQGame, DhtmlXQMove


def determine_category(subcategory: str, game_type: str, move_count: int) -> str:
    """Xác định category từ subcategory (folder path) trước, sau đó metadata."""
    if subcategory:
        sub_lower = subcategory.lower()
        if 'cam-bay' in sub_lower or 'cạm bẫy' in sub_lower:
            return 'Cạm bẫy khai cuộc'
        elif 'puzzle' in sub_lower or 'câu đố' in sub_lower or 'thế sát' in sub_lower:
            return 'Thế sát'
        elif 'end' in sub_lower or 'tàn' in sub_lower or '残局' in sub_lower:
            return 'Tàn cuộc'
        elif 'mid' in sub_lower or 'trung' in sub_lower or '中局' in sub_lower:
            return 'Trung cuộc'
        elif 'opening' in sub_lower or 'khai' in sub_lower or '开局' in sub_lower:
            return 'Khai cuộc'
        elif 'selected' in sub_lower or 'chọn' in sub_lower:
            return 'Ván chọn lọc'
        elif 'tournament' in sub_lower or 'giải' in sub_lower:
            return 'Giải đấu'

    # Fallback: dựa vào game_type
    game_type_lower = game_type.lower()
    if 'khai cuộc' in game_type_lower or '开局' in game_type_lower:
        return 'Khai cuộc'
    elif 'tàn cuộc' in game_type_lower or '残局' in game_type_lower:
        return 'Tàn cuộc'
    elif 'trung cuộc' in game_type_lower or '中局' in game_type_lower:
        return 'Trung cuộc'
    elif 'thế sát' in game_type_lower or '杀局' in game_type_lower:
        return 'Thế sát'
    elif 'puzzle' in game_type_lower or 'câu đố' in game_type_lower:
        return 'Câu đố'

    # Fallback cuối: dựa vào số nước đi
    if move_count <= 15:
        return 'Khai cuộc'
    elif move_count <= 40:
        return 'Trung cuộc'
    else:
        return 'Tàn cuộc'


def determine_difficulty(move_count: int, game_type: str) -> str:
    """Ước lượng độ khó."""
    if move_count <= 10:
        return 'Nhập môn'
    elif move_count <= 25:
        return 'Trung cấp'
    else:
        return 'Cao cấp'


def build_lesson(game: DhtmlXQGame) -> dict:
    """Chuyển DhtmlXQGame thành Lesson JSON."""

    move_count = len(game.movelist)
    category = determine_category(game.subcategory, game.game_type, move_count)
    difficulty = determine_difficulty(move_count, game.game_type)
    lesson_id = str(uuid.uuid4())

    # Chuyển moves sang format dự án
    lesson_moves = []
    for move in game.movelist:
        lesson_moves.append({
            "id": str(uuid.uuid4()),
            "side": move.side,
            "from": {
                "file": move.from_file,
                "rank": move.from_rank
            },
            "to": {
                "file": move.to_file,
                "rank": move.to_rank
            },
            "comment": move.comment
        })

    lesson = {
        "id": lesson_id,
        "title": game.title,
        "category": category,
        "difficulty": difficulty,
        "lines": [
            {
                "id": str(uuid.uuid4()),
                "title": "Biến chính",
                "moves": lesson_moves
            }
        ],
        "choice": {
            "prompt": "Bạn đánh giá nước đi nào tốt nhất trong ván cờ này?",
            "options": [
                {
                    "moveId": lesson_moves[0]["id"] if lesson_moves else "",
                    "label": "Theo biến chính",
                    "verdict": "correct",
                    "feedback": "Đây là biến chính của ván cờ."
                }
            ]
        }
    }

    if game.initial_fen:
        lesson["initialFen"] = game.initial_fen

    return lesson
