#!/usr/bin/env python3
"""Build an opening book from the production lesson catalog.

For each playable lesson line, walk the first N plies and count how often
each (position -> move) edge appears. The output is a flat JSON dictionary
keyed by FEN with an array of child moves carrying frequency, notation, and
an opening name when one is recognised.

Win rate is not computed here because the lesson catalog does not record
game results. The plan calls for merging in CCRL/Pikafish book data later
to enrich the frequency and add win/loss statistics.
"""

from __future__ import annotations

import argparse
import json
import sys
import uuid
from collections import defaultdict
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Iterable

REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_INPUT = REPO_ROOT / "apps/api/data/lessons/lessons.json"
DEFAULT_OUTPUT = REPO_ROOT / "apps/api/data/book/opening-book.json"

DEFAULT_XIANGQI_FEN = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"
OPENING_DEPTH = 8
PIECE_FEN_KIND = {
    "k": "general",
    "a": "advisor",
    "b": "elephant",
    "n": "horse",
    "h": "horse",
    "r": "chariot",
    "c": "cannon",
    "p": "soldier",
}
# Map our internal "kind" name back to the standard Xiangqi FEN letter.
# K=king/general, A=advisor, B=bishop/elephant, N=knight/horse, R=rook/chariot,
# C=cannon, P=pawn/soldier.
PIECE_KIND_TO_FEN_LETTER = {
    "general": "k",
    "advisor": "a",
    "elephant": "b",
    "horse": "n",
    "chariot": "r",
    "cannon": "c",
    "soldier": "p",
}
PIECE_LABELS_RED = {
    "general": "Tướng",
    "advisor": "Sĩ",
    "elephant": "Tượng",
    "horse": "Mã",
    "chariot": "Xe",
    "cannon": "Pháo",
    "soldier": "Tốt",
}
PIECE_LABELS_BLACK = {
    "general": "Tướng",
    "advisor": "Sĩ",
    "elephant": "Tượng",
    "horse": "Mã",
    "chariot": "Xe",
    "cannon": "Pháo",
    "soldier": "Binh",
}

NAMESPACE = uuid.UUID("0d2d3945-448a-4a4e-9eb2-cc48ba9e5c75")


@dataclass
class BoardState:
    pieces: list[tuple[int, int, str, str]] = field(default_factory=list)
    """List of (file, rank, side, kind). file in 0..8, rank in 0..9."""

    @classmethod
    def from_fen(cls, fen: str) -> "BoardState":
        placement = fen.strip().split(" ", 1)[0]
        ranks = placement.split("/")
        if len(ranks) != 10:
            raise ValueError(f"FEN must have 10 ranks, got {len(ranks)}")
        board = cls()
        for rank, row in enumerate(ranks):
            file = 0
            for symbol in row:
                if symbol.isdigit():
                    file += int(symbol)
                    continue
                side = "red" if symbol == symbol.upper() else "black"
                kind = PIECE_FEN_KIND.get(symbol.lower())
                if kind is None:
                    raise ValueError(f"Unsupported piece symbol {symbol!r} in FEN")
                board.pieces.append((file, rank, side, kind))
                file += 1
            if file != 9:
                raise ValueError(f"Rank {rank + 1} sums to {file} files, expected 9")
        return board

    def to_fen(self, side_to_move: str, move_number: int = 1) -> str:
        rows: list[str] = []
        for rank in range(10):
            row = ""
            empty = 0
            for file in range(9):
                piece = self.piece_at(file, rank)
                if piece is None:
                    empty += 1
                    continue
                if empty:
                    row += str(empty)
                    empty = 0
                symbol = PIECE_KIND_TO_FEN_LETTER[piece.kind]
                row += symbol.upper() if piece.side == "red" else symbol.lower()
            if empty:
                row += str(empty)
            rows.append(row)
        placement = "/".join(rows)
        turn = "w" if side_to_move == "red" else "b"
        return f"{placement} {turn} - - 0 {move_number}"

    def piece_at(self, file: int, rank: int) -> "Piece | None":
        for entry in self.pieces:
            if entry[0] == file and entry[1] == rank:
                return Piece(*entry)
        return None

    def apply(self, move: dict[str, Any]) -> "BoardState":
        side = move["side"]
        from_file, from_rank = move["from"]["file"], move["from"]["rank"]
        to_file, to_rank = move["to"]["file"], move["to"]["rank"]
        next_pieces: list[tuple[int, int, str, str]] = []
        moving: tuple[int, int, str, str] | None = None
        for entry in self.pieces:
            if entry[0] == from_file and entry[1] == from_rank and entry[2] == side:
                moving = entry
                continue
            if entry[0] == to_file and entry[1] == to_rank:
                continue
            next_pieces.append(entry)
        if moving is None:
            raise ValueError(f"No piece at {(from_file, from_rank)} for side {side}")
        next_pieces.append((to_file, to_rank, moving[2], moving[3]))
        return BoardState(pieces=next_pieces)

    def piece(self, file: int, rank: int) -> "Piece | None":
        return self.piece_at(file, rank)


@dataclass
class Piece:
    file: int
    rank: int
    side: str
    kind: str

    @property
    def label(self) -> str:
        return PIECE_LABELS_RED[self.kind] if self.side == "red" else PIECE_LABELS_BLACK[self.kind]


def format_notation(board: BoardState, move: dict[str, Any]) -> str:
    """Render a Vietnamese short notation: 'Mã 8 tiến 7'."""
    piece = board.piece(move["from"]["file"], move["from"]["rank"])
    if piece is None:
        return f"({move['from']['file']},{move['from']['rank']})→({move['to']['file']},{move['to']['rank']})"
    from_file = piece.file + 1 if piece.side == "red" else 9 - piece.file
    to_file = move["to"]["file"] + 1 if piece.side == "red" else 9 - move["to"]["file"]
    if piece.file == move["to"]["file"]:
        verb = "bình"
    else:
        is_forward = (piece.side == "red" and move["to"]["rank"] < move["from"]["rank"]) or (
            piece.side == "black" and move["to"]["rank"] > move["from"]["rank"]
        )
        verb = "tiến" if is_forward else "thoái"
    return f"{piece.label} {from_file} {verb} {to_file}"


def stable_move_uuid(parent_fen: str, move: dict[str, Any]) -> str:
    signature = f"{parent_fen}|{move['side']}|{move['from']['file']},{move['from']['rank']}|{move['to']['file']},{move['to']['rank']}"
    return str(uuid.uuid5(NAMESPACE, signature))


OPENING_NAMES: dict[str, str] = {
    "c3c4": "Pháo đầu (中央炮)",
    "c3c5": "Pháo đầu cản mã",
    "h2e2": "Bình phong mã (屏风马)",
    "h2g3": "Bình phong mã chậm",
    "h2h3": "Mã lên cao",
    "b0c2": "Mã thông thủ (马跳出)",
    "b0a2": "Mã biên",
    "a0a1": "Xe pháo lên",
    "a0a2": "Xe lên 2",
    "d0d1": "Sĩ lên",
    "f0f1": "Sĩ lên phải",
    "g0g1": "Tượng lên",
    "e0e1": "Tướng lên",
    "i0h0": "Xe qua đường",
}


def build_book(lessons: Iterable[dict[str, Any]], depth: int) -> dict[str, dict[str, Any]]:
    edge_frequency: dict[tuple[str, str], int] = defaultdict(int)
    move_meta: dict[tuple[str, str], dict[str, Any]] = {}
    position_visit: dict[str, int] = defaultdict(int)

    for lesson in lessons:
        opening_fen = (lesson.get("initialFen") or "").strip() or DEFAULT_XIANGQI_FEN
        for line in lesson.get("lines", []):
            moves = line.get("moves") or []
            if not moves:
                continue
            board = BoardState.from_fen(opening_fen)
            for idx, move in enumerate(moves[:depth]):
                side = move.get("side")
                if side is None:
                    break
                if board.piece(move["from"]["file"], move["from"]["rank"]) is None:
                    # Replay is broken (data inconsistency); skip the rest of this line.
                    break
                try:
                    parent_fen = board.to_fen(side, (idx // 2) + 1)
                except Exception:
                    break
                notation = format_notation(board, move)
                move_key = f"{move['from']['file']},{move['from']['rank']}->{move['to']['file']},{move['to']['rank']}"
                edge = (parent_fen, move_key)
                edge_frequency[edge] += 1
                if edge not in move_meta:
                    move_meta[edge] = {
                        "notation": notation,
                        "name": OPENING_NAMES.get(f"{chr(ord('a') + move['from']['file'])}{9 - move['from']['rank']}{chr(ord('a') + move['to']['file'])}{9 - move['to']['rank']}", ""),
                    }
                position_visit[parent_fen] += 1
                try:
                    board = board.apply(move)
                except Exception:
                    break

    positions: dict[str, dict[str, Any]] = {}
    for (parent_fen, move_key), freq in edge_frequency.items():
        visits = position_visit[parent_fen]
        meta = move_meta[(parent_fen, move_key)]
        children = positions.setdefault(parent_fen, {"moves": [], "visits": visits})
        children["moves"].append({
            "move": move_key,
            "notation": meta["notation"],
            "name": meta["name"],
            "frequency": freq,
            "popularity": round(freq / max(visits, 1), 4),
        })

    for position in positions.values():
        position["moves"].sort(key=lambda item: (-item["frequency"], item["notation"]))

    return positions


def main() -> int:
    parser = argparse.ArgumentParser(description="Build an opening book from lesson moves")
    parser.add_argument("--input", default=str(DEFAULT_INPUT))
    parser.add_argument("--output", default=str(DEFAULT_OUTPUT))
    parser.add_argument("--depth", type=int, default=OPENING_DEPTH, help="Number of plies to mine per line")
    parser.add_argument("--dry-run", action="store_true", help="Print a summary, do not write output")
    args = parser.parse_args()

    input_path = Path(args.input)
    output_path = Path(args.output)
    if not input_path.is_absolute():
        input_path = REPO_ROOT / input_path
    if not output_path.is_absolute():
        output_path = REPO_ROOT / output_path

    with input_path.open(encoding="utf-8") as handle:
        lessons = json.load(handle)
    if not isinstance(lessons, list):
        print(f"ERROR: input must be a JSON array of lessons, got {type(lessons).__name__}", file=sys.stderr)
        return 2

    book = build_book(lessons, args.depth)

    total_positions = len(book)
    total_edges = sum(len(position["moves"]) for position in book.values())
    popular_edges = sum(1 for position in book.values() for m in position["moves"] if m["frequency"] >= 5)
    summary = {
        "positions": total_positions,
        "edges": total_edges,
        "popular_edges": popular_edges,
        "depth": args.depth,
        "lessons_processed": len(lessons),
    }
    print(json.dumps(summary, ensure_ascii=False, indent=2))

    if args.dry_run:
        print("DRY RUN: no file written")
        return 0

    output_path.parent.mkdir(parents=True, exist_ok=True)
    payload = {
        "version": 1,
        "depth": args.depth,
        "source": "lessons.json (curated catalog)",
        "stats": summary,
        "positions": book,
    }
    output_path.write_text(json.dumps(payload, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"Wrote opening book: {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
