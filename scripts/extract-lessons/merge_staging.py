#!/usr/bin/env python3
"""
Safe merger for staging lesson files produced by the extract scripts.

Usage examples:
    # Dry run (recommended first)
    python merge_staging.py --input extracted_external_chessq_eglib.json --dry-run

    # Merge one or more staging files
    python merge_staging.py --input extracted_external_chessq_endgames.json extracted_external_xiokuai_fen.json

    # Merge everything in the current directory matching a pattern
    python merge_staging.py --input-glob "extracted_external_*.json"

    # Custom target (with backup)
    python merge_staging.py --input staging/ --target ../../apps/api/data/lessons/lessons.json

Always:
- Creates a timestamped backup of the target before writing.
- Deduplicates by normalized title (and optionally by FEN + main line fingerprint).
- Only adds lessons that don't already exist.
- Produces a clear report.
"""

import argparse
import hashlib
import json
import re
import shutil
import sys
from datetime import datetime
from pathlib import Path
from typing import List, Dict, Any, Optional, Set

# Lightweight Chinese → Vietnamese term dictionary (subset of translate_lessons.py)
CHESS_TERM_DICT = {
    "残局": "Tàn cuộc", "杀局": "Thế sát", "杀法": "Sát pháp",
    "开局": "Khai cuộc", "中局": "Trung cuộc",
    "基本": "Cơ bản", "战术": "Chiến thuật", "价值": "Giá trị",
    "实用": "Thực dụng", "原理": "Nguyên lý", "常见": "Phổ biến",
    "策略": "Chiến lược", "练习": "Bài tập", "实战": "Thực chiến",
    "对局": "Ván đấu", "象棋": "Cờ tướng",
}


def translate_title(title: str) -> str:
    """Simple term-by-term translation for Chinese lesson titles."""
    result = title
    for cn, vi in CHESS_TERM_DICT.items():
        result = result.replace(cn, vi)
    return result


def normalize_title(title: str) -> str:
    """Lowercase, strip, collapse whitespace for duplicate detection."""
    if not title:
        return ""
    t = title.strip().lower()
    t = re.sub(r'\s+', ' ', t)
    return t


def lesson_fingerprint(lesson: Dict[str, Any]) -> str:
    """
    Create a content-based fingerprint for deeper dedup.
    Uses initialFen (if present) + first 4 moves of the main line.
    """
    parts = []
    if lesson.get("initialFen"):
        parts.append(lesson["initialFen"].split()[0])  # placement only

    lines = lesson.get("lines", [])
    if lines:
        moves = lines[0].get("moves", [])
        for m in moves[:4]:
            parts.append(f"{m.get('side','')}{m.get('from',{})}->{m.get('to',{})}")

    key = "|".join(parts)
    if not key:
        return ""
    return hashlib.md5(key.encode("utf-8")).hexdigest()[:16]


def load_lessons(path: Path) -> List[Dict[str, Any]]:
    if not path.exists():
        print(f"ERROR: File not found: {path}")
        sys.exit(2)
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, list):
        print(f"ERROR: {path} does not contain a JSON array of lessons")
        sys.exit(2)
    return data


def strip_deprecated_fields(lesson: Dict[str, Any]) -> Dict[str, Any]:
    """Remove legacy fields that are no longer part of the active schema."""
    deprecated = {"slug", "summary", "tags", "principles", "notation", "evaluation"}
    cleaned = {k: v for k, v in lesson.items() if k not in deprecated}

    # Also clean inside lines/moves if needed
    for line in cleaned.get("lines", []):
        for move in line.get("moves", []):
            for bad in ("notation", "title", "evaluation"):
                move.pop(bad, None)
    return cleaned


def same_move_content(left: Dict[str, Any], right: Dict[str, Any]) -> bool:
    """Compare move payloads while ignoring generated IDs."""
    return (
        left.get("side") == right.get("side")
        and left.get("from") == right.get("from")
        and left.get("to") == right.get("to")
        and left.get("comment", "") == right.get("comment", "")
    )


def shared_move_prefix_length(lines: List[Dict[str, Any]]) -> int:
    """Return the common move prefix length required by move graph rendering."""
    if not lines:
        return 0
    prefix_length = len(lines[0].get("moves", []))
    for line in lines[1:]:
        index = 0
        moves = line.get("moves", [])
        first_moves = lines[0].get("moves", [])
        while index < prefix_length and index < len(moves) and same_move_content(first_moves[index], moves[index]):
            index += 1
        prefix_length = index
    return prefix_length


def normalize_lines_for_graph(lesson: Dict[str, Any]) -> None:
    """
    Keep imported lessons compatible with the current move graph contract.
    Multi-line lessons must share at least one leading move before branching;
    external files that only provide independent/empty lines are flattened to
    the first line instead of failing repository validation.
    """
    lines = lesson.get("lines", [])
    if len(lines) > 1 and shared_move_prefix_length(lines) == 0:
        lesson["lines"] = [lines[0]]


def normalize_moves(lesson: Dict[str, Any]) -> None:
    """Remove parser placeholders that do not represent playable moves."""
    source_type = lesson.get("source", {}).get("type")
    for line in lesson.get("lines", []):
        if source_type in {"md-book", "xqf"}:
            line["moves"] = []
            continue
        line["moves"] = [
            move
            for move in line.get("moves", [])
            if move.get("from") != move.get("to")
        ]


def normalize_choice_options(lesson: Dict[str, Any]) -> None:
    """Drop imported choice options that do not point at a real move ID."""
    choice = lesson.get("choice")
    if not isinstance(choice, dict):
        return

    move_ids = {
        move.get("id")
        for line in lesson.get("lines", [])
        for move in line.get("moves", [])
        if move.get("id")
    }
    options = [
        option
        for option in choice.get("options", [])
        if option.get("moveId") in move_ids
    ]
    if options:
        choice["options"] = options
    else:
        lesson.pop("choice", None)


def normalize_lesson_for_target(lesson: Dict[str, Any]) -> Dict[str, Any]:
    """Apply target-schema cleanup that is needed before dedup and writing."""
    normalize_moves(lesson)
    normalize_lines_for_graph(lesson)
    normalize_choice_options(lesson)
    return lesson


def lesson_dedup_keys(lesson: Dict[str, Any], use_content_dedup: bool) -> Dict[str, str]:
    """
    Build stable duplicate keys without treating missing content as a match.
    Empty fingerprints are ignored so lessons with no FEN and no moves are not
    collapsed into one accidental duplicate.
    """
    keys = {
        "id": lesson.get("id", ""),
        "title": normalize_title(lesson.get("title", "")),
        "fingerprint": "",
    }
    if use_content_dedup:
        keys["fingerprint"] = lesson_fingerprint(lesson)
    return keys


def find_duplicate(
    keys: Dict[str, str],
    by_id: Dict[str, Dict[str, Any]],
    by_title: Dict[str, Dict[str, Any]],
    by_fingerprint: Dict[str, Dict[str, Any]],
) -> Optional[Dict[str, Any]]:
    """Return the existing lesson matching any non-empty dedup key."""
    if keys["id"] and keys["id"] in by_id:
        return by_id[keys["id"]]
    if keys["title"] and keys["title"] in by_title:
        return by_title[keys["title"]]
    if keys["fingerprint"] and keys["fingerprint"] in by_fingerprint:
        return by_fingerprint[keys["fingerprint"]]
    return None


def add_to_indexes(
    lesson: Dict[str, Any],
    by_id: Dict[str, Dict[str, Any]],
    by_title: Dict[str, Dict[str, Any]],
    by_fingerprint: Dict[str, Dict[str, Any]],
    use_content_dedup: bool,
) -> None:
    """Index a lesson by all available dedup keys."""
    keys = lesson_dedup_keys(lesson, use_content_dedup)
    if keys["id"]:
        by_id[keys["id"]] = lesson
    if keys["title"]:
        by_title[keys["title"]] = lesson
    if keys["fingerprint"]:
        by_fingerprint[keys["fingerprint"]] = lesson


def merge_staging_files(
    staging_files: List[Path],
    target: Path,
    dry_run: bool = False,
    strip_deprecated: bool = True,
    use_content_dedup: bool = True,
    translate_titles: bool = False,
    interactive: bool = False,
) -> Dict[str, Any]:
    """
    Core merge logic with optional translation and interactive conflict resolution.
    Returns a report dict.
    """
    target_lessons: List[Dict[str, Any]] = []
    if target.exists():
        target_lessons = load_lessons(target)
        print(f"Loaded existing target: {len(target_lessons)} lessons")
    else:
        print("Target does not exist yet - will create new file")

    existing_titles: Set[str] = {normalize_title(l.get("title", "")) for l in target_lessons}
    by_id: Dict[str, Dict[str, Any]] = {}
    by_title: Dict[str, Dict[str, Any]] = {}
    by_fingerprint: Dict[str, Dict[str, Any]] = {}
    for existing in target_lessons:
        add_to_indexes(existing, by_id, by_title, by_fingerprint, use_content_dedup)

    added = []
    skipped_duplicate = []
    skipped_invalid = []
    conflicts = []   # list of (title, existing_lesson, new_lesson)

    for sf in staging_files:
        print(f"\nProcessing staging file: {sf.name}")
        try:
            incoming = load_lessons(sf)
        except Exception as e:
            print(f"  ERROR loading {sf}: {e}")
            skipped_invalid.append(str(sf))
            continue

        for lesson in incoming:
            if not lesson.get("id") or not lesson.get("title"):
                skipped_invalid.append(lesson.get("title") or lesson.get("id", "<no id/title>"))
                continue

            if strip_deprecated:
                lesson = strip_deprecated_fields(lesson)
            lesson = normalize_lesson_for_target(lesson)

            if translate_titles:
                original_title = lesson.get("title", "")
                translated = translate_title(original_title)
                if translated != original_title:
                    lesson["title"] = translated
                    lesson["_original_title"] = original_title   # keep for audit

            keys = lesson_dedup_keys(lesson, use_content_dedup)
            existing_match = find_duplicate(keys, by_id, by_title, by_fingerprint)

            if existing_match:
                # Check if it's a real conflict (same title, different content)
                fp = keys["fingerprint"]
                if keys["title"] == normalize_title(existing_match.get("title", "")) and lesson_fingerprint(existing_match) != fp:
                    conflicts.append((lesson.get("title"), existing_match, lesson))
                skipped_duplicate.append(lesson.get("title", "<unknown>"))
                continue

            target_lessons.append(lesson)
            existing_titles.add(keys["title"])
            add_to_indexes(lesson, by_id, by_title, by_fingerprint, use_content_dedup)
            added.append(lesson.get("title", "<unknown>"))

    # Interactive conflict resolution
    resolved = 0
    if interactive and conflicts:
        print(f"\n=== {len(conflicts)} title conflicts detected (same title, different content) ===")
        for title, existing, new in conflicts:
            print(f"\nConflict: {title}")
            print(f"  Existing fingerprint: {lesson_fingerprint(existing)}")
            print(f"  New      fingerprint: {lesson_fingerprint(new)}")
            choice = input("  Keep [e]xisting / [n]ew / [s]kip both? (e/n/s) ").strip().lower()
            if choice == "n":
                # Replace existing with new
                target_lessons = [l for l in target_lessons if normalize_title(l.get("title","")) != normalize_title(title)]
                target_lessons.append(new)
                resolved += 1
                print("  → Kept new version")
            elif choice == "e":
                print("  → Kept existing version")
            else:
                print("  → Skipped both (no change)")

    report = {
        "timestamp": datetime.now().isoformat(),
        "target": str(target),
        "staging_files": [str(f) for f in staging_files],
        "before_count": len(target_lessons) - len(added) - resolved,
        "added_count": len(added),
        "skipped_duplicate_count": len(skipped_duplicate),
        "skipped_invalid_count": len(skipped_invalid),
        "conflicts_found": len(conflicts),
        "conflicts_resolved": resolved,
        "after_count": len(target_lessons),
        "new_categories": sorted({l.get("category", "") for l in target_lessons if l.get("category")}),
        "translation_enabled": translate_titles,
    }

    if dry_run:
        print("\n=== DRY RUN - No changes written ===")
        print(f"Would add: {len(added)} lessons")
        print(f"Conflicts detected: {len(conflicts)}")
        print(f"Would skip (duplicates): {len(skipped_duplicate)}")
        print(f"Final count would be: {len(target_lessons)}")
        return report

    # Real write
    if target.exists():
        ts = datetime.now().strftime("%Y%m%d-%H%M%S")
        backup = target.with_name(f"{target.name}.bak-{ts}")
        shutil.copy2(target, backup)
        print(f"\nBackup created: {backup}")

    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(target_lessons, ensure_ascii=False, indent=2), encoding="utf-8")

    print("\n=== MERGE COMPLETE ===")
    print(f"Added: {len(added)} new lessons")
    print(f"Skipped duplicates: {len(skipped_duplicate)}")
    print(f"Conflicts: {len(conflicts)} (resolved interactively: {resolved})")
    print(f"Total in {target}: {len(target_lessons)}")

    return report


def main():
    parser = argparse.ArgumentParser(
        description="Safely merge staging lesson JSON files into the production lessons.json"
    )
    parser.add_argument(
        "--input", "-i", nargs="+", type=Path,
        help="One or more staging JSON files to merge"
    )
    parser.add_argument(
        "--input-glob", type=str,
        help="Glob pattern for staging files (e.g. 'extracted_external_*.json')"
    )
    parser.add_argument(
        "--target", "-t", type=Path,
        default=Path("apps/api/data/lessons/lessons.json"),
        help="Target production lessons.json (default: apps/api/data/lessons/lessons.json)"
    )
    parser.add_argument(
        "--dry-run", action="store_true",
        help="Show what would happen without writing anything"
    )
    parser.add_argument(
        "--no-content-dedup", action="store_true",
        help="Only dedup by title (faster, less strict)"
    )
    parser.add_argument(
        "--keep-deprecated", action="store_true",
        help="Keep legacy fields like 'slug' (not recommended)"
    )
    parser.add_argument(
        "--translate", action="store_true",
        help="Apply lightweight Chinese→Vietnamese term translation on titles before merging"
    )
    parser.add_argument(
        "--interactive", action="store_true",
        help="Interactively resolve title conflicts (same title, different content)"
    )

    args = parser.parse_args()

    staging_files: List[Path] = []
    if args.input:
        staging_files.extend(args.input)
    if args.input_glob:
        base = Path(".")
        staging_files.extend(sorted(base.glob(args.input_glob)))

    # Remove duplicates in the list while preserving order
    seen = set()
    staging_files = [f for f in staging_files if not (str(f) in seen or seen.add(str(f)))]

    if not staging_files:
        print("ERROR: No staging files specified. Use --input or --input-glob")
        sys.exit(1)

    # Resolve target relative to repo root if needed
    target = args.target
    if not target.is_absolute():
        # Try to find repo root
        here = Path(__file__).resolve()
        for parent in [here] + list(here.parents):
            if (parent / "apps" / "api" / "data" / "lessons").exists():
                target = parent / target
                break

    print(f"Target file: {target}")
    print(f"Staging files to consider: {[f.name for f in staging_files]}")

    report = merge_staging_files(
        staging_files=staging_files,
        target=target,
        dry_run=args.dry_run,
        strip_deprecated=not args.keep_deprecated,
        use_content_dedup=not args.no_content_dedup,
        translate_titles=args.translate,
        interactive=args.interactive,
    )

    # Write a small merge report next to the target
    if not args.dry_run:
        report_path = target.parent / f"merge-report-{datetime.now().strftime('%Y%m%d-%H%M%S')}.json"
        report_path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
        print(f"\nMerge report written to: {report_path}")

    print("\nDone.")


if __name__ == "__main__":
    main()
