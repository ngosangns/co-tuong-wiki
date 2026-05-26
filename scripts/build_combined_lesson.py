#!/usr/bin/env python3
"""Build a single combined lesson artifact from the production lesson catalog."""

from __future__ import annotations

import argparse
import copy
import json
import sys
import uuid
from pathlib import Path
from typing import Any


NAMESPACE = uuid.UUID("0d2d3945-448a-4a4e-9eb2-cc48ba9e5c75")
DEFAULT_ID = str(uuid.uuid5(NAMESPACE, "co-tuong-wiki:combined-lesson:v1"))


def repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


def resolve_path(path: str | Path) -> Path:
    candidate = Path(path)
    if candidate.is_absolute():
        return candidate
    return repo_root() / candidate


def stable_uuid(*parts: object) -> str:
    return str(uuid.uuid5(NAMESPACE, ":".join(str(part) for part in parts)))


def load_catalog(path: Path) -> list[dict[str, Any]]:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        print(f"ERROR: input file not found: {path}", file=sys.stderr)
        raise SystemExit(2)
    except json.JSONDecodeError as error:
        print(f"ERROR: input file is not valid JSON: {error}", file=sys.stderr)
        raise SystemExit(2)

    if not isinstance(data, list):
        print("ERROR: input must be a JSON array of lessons", file=sys.stderr)
        raise SystemExit(2)
    return data


def require_string(value: Any, field: str, context: str) -> str:
    if not isinstance(value, str) or not value.strip():
        print(f"ERROR: {context} is missing string field {field!r}", file=sys.stderr)
        raise SystemExit(2)
    return value


def lesson_line_title(lesson: dict[str, Any], line: dict[str, Any], line_number: int) -> str:
    lesson_title = require_string(lesson.get("title"), "title", "lesson")
    line_title = line.get("title")
    if isinstance(line_title, str) and line_title.strip():
        return f"{lesson_title} · {line_title.strip()}"
    return f"{lesson_title} · Biến {line_number}"


def build_combined_lesson(lessons: list[dict[str, Any]], lesson_id: str, title: str) -> tuple[dict[str, Any], dict[str, Any]]:
    combined_lines: list[dict[str, Any]] = []
    source_lessons = 0
    source_lines = 0
    source_moves = 0
    generated_move_ids: set[str] = set()

    for lesson_index, lesson in enumerate(lessons, start=1):
        lesson_id_source = require_string(lesson.get("id"), "id", f"lesson #{lesson_index}")
        lines = lesson.get("lines")
        if not isinstance(lines, list) or not lines:
            print(f"ERROR: lesson {lesson_id_source} must contain at least one line", file=sys.stderr)
            raise SystemExit(2)

        source_lessons += 1
        lesson_initial_fen = lesson.get("initialFen")

        for line_index, line in enumerate(lines, start=1):
            if not isinstance(line, dict):
                print(f"ERROR: lesson {lesson_id_source} line #{line_index} must be an object", file=sys.stderr)
                raise SystemExit(2)

            line_id_source = require_string(line.get("id"), "id", f"lesson {lesson_id_source} line #{line_index}")
            moves = line.get("moves")
            if not isinstance(moves, list):
                print(f"ERROR: lesson {lesson_id_source} line {line_id_source} moves must be an array", file=sys.stderr)
                raise SystemExit(2)

            combined_moves = []
            for move_index, move in enumerate(moves, start=1):
                if not isinstance(move, dict):
                    print(f"ERROR: lesson {lesson_id_source} line {line_id_source} move #{move_index} must be an object", file=sys.stderr)
                    raise SystemExit(2)

                source_move_id = require_string(move.get("id"), "id", f"lesson {lesson_id_source} line {line_id_source} move #{move_index}")
                next_move = copy.deepcopy(move)
                next_move["id"] = stable_uuid("move", lesson_id_source, line_id_source, source_move_id, move_index)
                if next_move["id"] in generated_move_ids:
                    print(f"ERROR: generated duplicate move id {next_move['id']}", file=sys.stderr)
                    raise SystemExit(2)
                generated_move_ids.add(next_move["id"])
                combined_moves.append(next_move)

            next_line = {
                "id": stable_uuid("line", lesson_id_source, line_id_source, line_index),
                "title": lesson_line_title(lesson, line, line_index),
                "moves": combined_moves,
            }
            if isinstance(lesson_initial_fen, str) and lesson_initial_fen.strip():
                # Line-level FEN lets the combined page replay lessons with independent starting positions.
                next_line["initialFen"] = lesson_initial_fen.strip()

            combined_lines.append(next_line)
            source_lines += 1
            source_moves += len(combined_moves)

    combined = {
        "id": lesson_id,
        "title": title,
        "category": "Tổng hợp",
        "difficulty": "Tất cả",
        "lines": combined_lines,
        "choice": {
            "prompt": "",
            "options": [],
        },
    }

    report = {
        "source_lessons": source_lessons,
        "source_lines": source_lines,
        "source_moves": source_moves,
        "combined_lines": len(combined_lines),
        "combined_moves": sum(len(line["moves"]) for line in combined_lines),
        "line_level_initial_fen": sum(1 for line in combined_lines if "initialFen" in line),
    }
    return combined, report


def main() -> int:
    parser = argparse.ArgumentParser(description="Build one combined lesson artifact from lessons.json")
    parser.add_argument("--input", default="apps/api/data/lessons/lessons.json", help="Source lessons JSON array")
    parser.add_argument("--output", default="apps/api/data/lessons/combined-lesson.json", help="Combined lesson JSON output")
    parser.add_argument("--id", default=DEFAULT_ID, help="Stable id for the combined lesson")
    parser.add_argument("--title", default="Tổng hợp toàn bộ lesson", help="Title for the combined lesson")
    parser.add_argument("--dry-run", action="store_true", help="Print a report without writing the output file")
    args = parser.parse_args()

    input_path = resolve_path(args.input)
    output_path = resolve_path(args.output)
    lessons = load_catalog(input_path)
    combined, report = build_combined_lesson(lessons, args.id, args.title)

    print(json.dumps(report, ensure_ascii=False, indent=2))
    if args.dry_run:
        print("DRY RUN: no file written")
        return 0

    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(combined, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"Wrote combined lesson: {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
