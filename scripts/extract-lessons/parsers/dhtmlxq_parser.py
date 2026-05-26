"""
Parser cho định dạng DhtmlXQ (định dạng file ván cờ tướng phổ biến).

Format mỗi file:
[DhtmlXQ]
[DhtmlXQ_title]tiêu đề[/DhtmlXQ_title]
[DhtmlXQ_binit]...[/DhtmlXQ_binit]
[DhtmlXQ_movelist]...[/DhtmlXQ_movelist]
[DhtmlXQ_commentN]bình luận[/DhtmlXQ_commentN]
...
[/DhtmlXQ]

Movelist: chuỗi số, mỗi nước 4 chữ số (ffrr = from-file, from-rank, to-file, to-rank)
Hệ tọa độ: file 0-8 (trái→phải), rank 0-9 (đen→đỏ)
Khớp với hệ tọa độ dự án.
"""

import re
from typing import List, Optional
from dataclasses import dataclass


@dataclass
class DhtmlXQMove:
    side: str  # 'red' hoặc 'black'
    from_file: int
    from_rank: int
    to_file: int
    to_rank: int
    notation: str
    comment: str = ""
    move_number: int = 0


@dataclass
class DhtmlXQGame:
    title: str
    event: str
    date: str
    place: str
    red: str
    black: str
    result: str
    opening: str
    remark: str
    author: str
    game_type: str
    movelist: List[DhtmlXQMove]
    initial_fen: Optional[str] = None
    subcategory: Optional[str] = None


def parse_tag(content: str, tag: str) -> str:
    """Trích xuất giá trị từ tag DhtmlXQ."""
    pattern = rf"\[DhtmlXQ_{tag}\](.*?)\[/DhtmlXQ_{tag}\]"
    match = re.search(pattern, content, re.DOTALL)
    return match.group(1).strip() if match else ""


def parse_movelist(movelist_str: str) -> List[DhtmlXQMove]:
    """
    Parse chuỗi movelist thành danh sách moves.
    Mỗi nước = 4 chữ số: [from_file][from_rank][to_file][to_rank]
    """
    moves = []
    # Loại bỏ khoảng trắng và xuống dòng
    cleaned = re.sub(r'\s+', '', movelist_str)

    for i in range(0, len(cleaned), 4):
        chunk = cleaned[i:i+4]
        if len(chunk) < 4:
            break

        try:
            from_file = int(chunk[0])
            from_rank = int(chunk[1])
            to_file = int(chunk[2])
            to_rank = int(chunk[3])
        except (ValueError, IndexError):
            continue

        move_number = len(moves) + 1
        side = 'red' if move_number % 2 == 1 else 'black'

        # Tạo notation đơn giản, không kèm số thứ tự nước đi.
        notation = f"({from_file},{from_rank})→({to_file},{to_rank})"

        moves.append(DhtmlXQMove(
            side=side,
            from_file=from_file,
            from_rank=from_rank,
            to_file=to_file,
            to_rank=to_rank,
            notation=notation,
            move_number=move_number
        ))

    return moves


def parse_comments(content: str, move_count: int) -> dict:
    """Parse tất cả comments theo số thứ tự nước đi."""
    comments = {}
    pattern = r"\[DhtmlXQ_comment(\d+)\](.*?)\[/DhtmlXQ_comment\d+\]"
    for match in re.finditer(pattern, content, re.DOTALL):
        move_num = int(match.group(1))
        comment = match.group(2).strip()
        comments[move_num] = comment
    return comments


def binit_to_fen(binit: str) -> str:
    """Chuyển DhtmlXQ binit (32 cặp vị trí) sang Xiangqi FEN chuẩn."""
    if len(binit) < 64:
        return ""

    # Piece order trong DhtmlXQ: 32 quân cờ
    piece_order = [
        ('red', 'chariot'), ('red', 'horse'), ('red', 'elephant'), ('red', 'advisor'),
        ('red', 'general'), ('red', 'advisor'), ('red', 'elephant'), ('red', 'horse'),
        ('red', 'chariot'), ('red', 'cannon'), ('red', 'cannon'),
        ('red', 'soldier'), ('red', 'soldier'), ('red', 'soldier'), ('red', 'soldier'), ('red', 'soldier'),
        ('black', 'chariot'), ('black', 'horse'), ('black', 'elephant'), ('black', 'advisor'),
        ('black', 'general'), ('black', 'advisor'), ('black', 'elephant'), ('black', 'horse'),
        ('black', 'chariot'), ('black', 'cannon'), ('black', 'cannon'),
        ('black', 'soldier'), ('black', 'soldier'), ('black', 'soldier'), ('black', 'soldier'), ('black', 'soldier'),
    ]

    fen_map = {
        ('red', 'chariot'): 'R', ('red', 'horse'): 'N', ('red', 'elephant'): 'B',
        ('red', 'advisor'): 'A', ('red', 'general'): 'K', ('red', 'cannon'): 'C',
        ('red', 'soldier'): 'P',
        ('black', 'chariot'): 'r', ('black', 'horse'): 'n', ('black', 'elephant'): 'b',
        ('black', 'advisor'): 'a', ('black', 'general'): 'k', ('black', 'cannon'): 'c',
        ('black', 'soldier'): 'p',
    }

    # Tạo board 10x9 (rank x file), tất cả ô trống
    board = [['' for _ in range(9)] for _ in range(10)]

    for i in range(32):
        pair = binit[i*2:i*2+2]
        if pair == '99':
            continue
        x = int(pair[0])  # file
        y = int(pair[1])  # rank
        if 0 <= x < 9 and 0 <= y < 10:
            side, kind = piece_order[i]
            board[y][x] = fen_map[(side, kind)]

    # Build FEN placement string
    fen_rows = []
    for rank in range(10):
        row = ''
        empty = 0
        for file in range(9):
            if board[rank][file]:
                if empty > 0:
                    row += str(empty)
                    empty = 0
                row += board[rank][file]
            else:
                empty += 1
        if empty > 0:
            row += str(empty)
        fen_rows.append(row)

    return '/'.join(fen_rows)


def detect_first_side_from_binit(binit: str, first_move: DhtmlXQMove) -> str:
    """Detect which side moves first from binit and first move."""
    if len(binit) < 64:
        return 'red'

    piece_order = [
        'R', 'N', 'B', 'A', 'K', 'A', 'B', 'N', 'R', 'C', 'C',
        'P', 'P', 'P', 'P', 'P',
        'r', 'n', 'b', 'a', 'k', 'a', 'b', 'n', 'r', 'c', 'c',
        'p', 'p', 'p', 'p', 'p',
    ]

    for i in range(32):
        pair = binit[i*2:i*2+2]
        if pair == '99':
            continue
        x = int(pair[0])
        y = int(pair[1])
        if x == first_move.from_file and y == first_move.from_rank:
            piece = piece_order[i]
            return 'red' if piece.isupper() else 'black'

    return 'red'


def parse_dpxq(content: str) -> Optional[DhtmlXQGame]:
    """Parse toàn bộ file DhtmlXQ."""
    if not content or '[DhtmlXQ]' not in content:
        return None

    title = parse_tag(content, 'title')
    if not title:
        title = "Unknown Game"

    movelist_str = parse_tag(content, 'movelist')
    if not movelist_str:
        return None

    moves = parse_movelist(movelist_str)
    if not moves:
        return None

    # Parse comments
    comments = parse_comments(content, len(moves))
    for move in moves:
        if move.move_number in comments:
            move.comment = comments[move.move_number]

    # Chuyển binit sang FEN
    binit = parse_tag(content, 'binit')
    initial_fen = None
    default_binit = "0919293949596979891777062646668600102030405060708012720323436383"
    if binit and binit != default_binit:
        fen_placement = binit_to_fen(binit)
        if fen_placement:
            initial_fen = f"{fen_placement} w - - 0 1"

        # Detect first side from binit
        first_side = detect_first_side_from_binit(binit, moves[0])
        for i, move in enumerate(moves):
            if first_side == 'black':
                move.side = 'black' if i % 2 == 0 else 'red'
            else:
                move.side = 'red' if i % 2 == 0 else 'black'
            move.move_number = i + 1

    return DhtmlXQGame(
        title=title,
        event=parse_tag(content, 'event'),
        date=parse_tag(content, 'date'),
        place=parse_tag(content, 'place'),
        red=parse_tag(content, 'red'),
        black=parse_tag(content, 'black'),
        result=parse_tag(content, 'result'),
        opening=parse_tag(content, 'open'),
        remark=parse_tag(content, 'remark'),
        author=parse_tag(content, 'author'),
        game_type=parse_tag(content, 'type'),
        movelist=moves,
        initial_fen=initial_fen
    )
