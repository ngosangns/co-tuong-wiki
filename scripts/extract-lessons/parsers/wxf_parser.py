"""
Minimal parser for WXF (Xiangqi game notation format).

WXF is a text-based format used by some tools (similar to PGN but for Xiangqi).
See: data/external/xiangqi-setup/doc/*.wxf for examples.

Current support: basic extraction of FEN (if present) + move list.
Full move notation parsing (P3+1, c8.7 etc.) is complex; for now we store
raw moves and rely on initialFen for the position.

Only a handful of WXF files exist in data/external today (mostly demos).
"""

from __future__ import annotations

import re
import uuid
from dataclasses import dataclass
from pathlib import Path
from typing import List, Optional


@dataclass
class WXFGame:
    title: str
    initial_fen: Optional[str] = None
    moves_raw: List[str] = None   # raw notation strings
    result: Optional[str] = None
    source_file: str = ""


def parse_wxf_file(path: str | Path) -> Optional[WXFGame]:
    path = Path(path)
    if not path.exists():
        return None

    try:
        text = path.read_text(encoding="utf-8", errors="ignore")
    except Exception:
        return None

    if "FORMAT" not in text and "START{" not in text:
        return None

    # Extract FEN if present (WXF sometimes has "FEN  <fen>")
    fen_match = re.search(r"FEN\s+([^\r\n]+)", text)
    initial_fen = fen_match.group(1).strip() if fen_match else None

    # Extract moves block
    moves_block = ""
    m = re.search(r"START\{(.*?)\}END", text, re.DOTALL)
    if m:
        moves_block = m.group(1)

    # Very rough split of moves (numbered or space separated)
    raw_moves = re.findall(r"[\d]+\.\s*([^ \r\n]+)\s+([^ \r\n]+)", moves_block)
    flat_moves = []
    for red, black in raw_moves:
        flat_moves.append(red)
        if black:
            flat_moves.append(black)

    if not flat_moves and not initial_fen:
        return None

    title = path.stem.replace("_", " ")

    # Try to get result
    res = re.search(r"RESULT\s+([^\r\n]+)", text)
    result = res.group(1).strip() if res else None

    return WXFGame(
        title=title,
        initial_fen=initial_fen,
        moves_raw=flat_moves,
        result=result,
        source_file=str(path),
    )


def collect_wxf_files(base_path: str | Path) -> List[Path]:
    base = Path(base_path)
    if not base.exists():
        return []
    return sorted(base.rglob("*.wxf"))


def process_wxf_directory(base_path: str | Path, limit: Optional[int] = None) -> List[dict]:
    base = Path(base_path)
    files = collect_wxf_files(base)
    if limit:
        files = files[:limit]

    lessons = []
    for fpath in files:
        game = parse_wxf_file(fpath)
        if not game:
            continue

        lesson = {
            "id": str(uuid.uuid4()),
            "title": game.title,
            "category": "Ván chọn lọc",  # or try to infer
            "difficulty": "Trung cấp",
            "source": {
                "type": "wxf",
                "file": game.source_file,
                "result": game.result,
            },
            "lines": [
                {
                    "id": str(uuid.uuid4()),
                    "title": "Biến chính",
                    "moves": [
                        {
                            "id": str(uuid.uuid4()),
                            "side": "red" if i % 2 == 0 else "black",
                            "from": {"file": 0, "rank": 0},   # placeholder – real parsing would fill this
                            "to": {"file": 0, "rank": 0},
                            "comment": f"Raw WXF: {mv}",
                        }
                        for i, mv in enumerate(game.moves_raw or [])
                    ],
                }
            ],
        }
        if game.initial_fen:
            lesson["initialFen"] = game.initial_fen

        lessons.append(lesson)

    return lessons
