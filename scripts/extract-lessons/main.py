#!/usr/bin/env python3
"""
Pipeline extract dữ liệu ván cờ tướng từ các nguồn công khai (DhtmlXQ + XQF).

Mục tiêu chính:
- Chuyển data từ data/external (đặc biệt ChessQ/gamebooks *.xqf) thành lesson format.
- **Mặc định output sang file riêng** (staging) dưới scripts/extract-lessons/,
  KHÔNG ghi trực tiếp vào apps/api/data/lessons/lessons.json.

Usage ví dụ:

  # XQF từ data/external (ChessQ/gamebooks - 151 endgame/puzzle rất tốt)
  python main.py --source external-xqf --limit 30

  # FEN patterns (xiokuai ~1100+ thế tàn cục)
  python main.py --source external-fen --limit 50

  # EGLIB - cực kỳ mạnh: hàng nghìn vị trí từ sách danh tiếng
  # (杀局谱 6k, final_book 3.7k, 适情雅趣360, 基本杀法...)
  python main.py --source external-eglib --limit 200
  # Quét đúng thư mục gamebooks
  python main.py --local-path ../../data/external/ChessQ/gamebooks --source external-eglib

  # Excel 象棋残局 (endgame database)
  python main.py --source external-excel --limit 50

  # Quét XQF cụ thể
  python main.py --local-path ../../data/external/ChessQ/gamebooks --source external-xqf

  # DhtmlXQ cũ
  python main.py --local-path /path/to/dhtmlxq/files --output extracted_from_vietcotuong.json

Sau khi có file staging:
  - Review nội dung (đặc biệt title từ sách Trung)
  - Dùng `python merge_staging.py --input extracted_external_*.json` để merge an toàn vào production
    (script này tự backup + dedup theo title + fingerprint)

Các nguồn external mạnh nhất hiện hỗ trợ (--source):
  - external-xqf   : 151 file .xqf (象棋残局杀势 + killing techniques)
  - external-fen   : ~1100+ FEN từ xiokuai (mẫu hình tàn cục)
  - external-eglib : Hàng chục nghìn vị trí từ sách danh tiếng (杀局谱, 适情雅趣360, 基本杀法...)
  - external-wxf   : WXF files (ít nhưng chuẩn)
  - external-excel : Excel 象棋残局 (endgame database theo chất liệu)
  - external       : Chế độ broad – quét data/external và dispatch tất cả parser (XQF + FEN + EGLIB + WXF + Excel + XiangqiBook)

Yêu cầu (chỉ cho XQF):
  python3 -m pip install cchess --break-system-packages
"""

import argparse
import json
import os
import sys
import time
from pathlib import Path
from typing import List, Optional

# Thêm đường dẫn để import parsers và transformers
sys.path.insert(0, str(Path(__file__).parent))

from parsers.dhtmlxq_parser import parse_dpxq
from transformers.lesson_builder import build_lesson
from parsers.xqf_parser import process_xqf_directory, collect_xqf_files
from parsers.fen_parser import process_fen_directory, collect_fen_files
from parsers.eglib_parser import process_eglib_directory, collect_eglib_files
from parsers.wxf_parser import process_wxf_directory, collect_wxf_files
from parsers.xiangqi_book_parser import process_xiangqi_book
from parsers.excel_endgame_parser import process_excel_endgames


def fetch_github_directory(owner: str, repo: str, path: str, token: Optional[str] = None) -> List[dict]:
    """Fetch danh sách file/folder từ GitHub API."""
    import urllib.request
    import urllib.error
    import urllib.parse

    encoded_path = urllib.parse.quote(path, safe='/')
    url = f"https://api.github.com/repos/{owner}/{repo}/contents/{encoded_path}"
    headers = {"User-Agent": "Mozilla/5.0"}
    if token:
        headers["Authorization"] = f"token {token}"

    req = urllib.request.Request(url, headers=headers)

    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            return json.loads(response.read().decode('utf-8'))
    except urllib.error.HTTPError as e:
        if e.code == 404:
            return []
        raise
    except Exception as e:
        print(f"Error fetching {url}: {e}")
        return []


def fetch_raw_file(url: str) -> str:
    """Fetch nội dung file raw từ GitHub."""
    import urllib.request
    import urllib.error
    import urllib.parse

    # Encode URL to handle Unicode characters
    parsed = urllib.parse.urlparse(url)
    encoded_path = urllib.parse.quote(parsed.path, safe='/')
    encoded_url = urllib.parse.urlunparse(parsed._replace(path=encoded_path))

    headers = {"User-Agent": "Mozilla/5.0"}
    req = urllib.request.Request(encoded_url, headers=headers)

    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            return response.read().decode('utf-8', errors='ignore')
    except Exception as e:
        print(f"Error fetching file {encoded_url}: {e}")
        return ""


def detect_subcategory(path_parts: tuple) -> str:
    """Detect subcategory from folder path parts."""
    path_str = '/'.join(path_parts).lower()

    if 'cam-bay' in path_str or 'cạm bẫy' in path_str:
        return 'Cạm bẫy khai cuộc'
    elif 'opening' in path_str or 'khai-cuoc' in path_str:
        return 'Khai cuộc'
    elif 'puzzle' in path_str or 'thế sát' in path_str or 'the-sat' in path_str:
        return 'Thế sát'
    elif 'end-game' in path_str or 'tàn-cuộc' in path_str or 'endgame' in path_str:
        return 'Tàn cuộc'
    elif 'mid-game' in path_str or 'trung-cuộc' in path_str or 'midgame' in path_str:
        return 'Trung cuộc'
    elif 'selected' in path_str or 'chọn' in path_str:
        return 'Ván chọn lọc'
    elif 'tournament' in path_str or 'giải' in path_str:
        return 'Giải đấu'

    return ''


def collect_dpxq_files_local(base_path: str) -> List[dict]:
    """
    Thu thập tất cả file DhtmlXQ từ local directory.
    Trả về list các dict có 'path', 'name'.
    """
    files = []
    base = Path(base_path)

    for path in base.rglob('*'):
        if path.is_file():
            name = path.name
            # File DhtmlXQ thường không có extension hoặc .dpxq
            if not name.endswith(('.md', '.json', '.txt', '.png', '.jpg', '.gif', '.svg', '.html')):
                files.append({
                    'path': str(path),
                    'name': name
                })

    return files


def process_files_local(files: List[dict], limit: Optional[int] = None) -> List[dict]:
    """Đọc và parse các file DhtmlXQ từ local."""
    lessons = []
    errors = []

    total = len(files) if limit is None else min(limit, len(files))

    for i, file_info in enumerate(files[:total]):
        print(f"[{i+1}/{total}] Processing: {file_info['path']}")

        try:
            with open(file_info['path'], 'r', encoding='utf-8', errors='ignore') as f:
                content = f.read()
        except Exception as e:
            errors.append(f"Read failed: {file_info['path']} - {e}")
            continue

        if not content:
            errors.append(f"Empty content: {file_info['path']}")
            continue

        # Kiểm tra xem có phải file DhtmlXQ không
        if '[DhtmlXQ]' not in content:
            print(f"  Skip: Not a DhtmlXQ file")
            continue

        game = parse_dpxq(content)
        if not game:
            errors.append(f"Parse failed: {file_info['path']}")
            continue

        if not game.movelist:
            errors.append(f"No moves: {file_info['path']}")
            continue

        # Detect subcategory from folder path
        path_parts = Path(file_info['path']).parts
        game.subcategory = detect_subcategory(path_parts)

        lesson = build_lesson(game)
        lessons.append(lesson)
        print(f"  OK: {game.title} ({len(game.movelist)} moves) [{lesson.get('category', '?')}]")

    if errors:
        print(f"\nErrors ({len(errors)}):")
        for err in errors[:10]:
            print(f"  - {err}")

    return lessons


def _is_production_path(p: Path) -> bool:
    """Tránh ghi trực tiếp lên file production lessons.json."""
    s = str(p).lower()
    return "apps/api/data/lessons" in s or "api/data/lessons/lessons.json" in s


def _resolve_safe_output(args, default_name: str) -> Path:
    """Quyết định file output an toàn (staging) trừ khi user explicit allow."""
    if args.output:
        out = Path(args.output)
    else:
        out = Path(__file__).parent / default_name

    if _is_production_path(out) and not args.allow_production:
        print("ERROR: Output path trỏ vào production lessons.json.")
        print("       Script này mặc định output sang file riêng (staging).")
        print("       Dùng --allow-production nếu bạn thực sự muốn ghi đè (không khuyến khích).")
        raise SystemExit(2)

    out.parent.mkdir(parents=True, exist_ok=True)
    return out


def main():
    parser = argparse.ArgumentParser(description='Extract xiangqi game data from public sources')
    parser.add_argument('--source', choices=['vietcotuong', 'external-xqf', 'external-fen', 'external-eglib', 'external-wxf', 'external-excel', 'external', 'all'], default='vietcotuong',
                        help='Data source. Use "external" for broad scan (dispatches to XQF/FEN/EGLIB/WXF/Excel/Book)')
    parser.add_argument('--output', type=str, default=None,
                        help='Output JSON file path (defaults to safe staging file under this dir for external sources)')
    parser.add_argument('--limit', type=int, default=None,
                        help='Limit number of files to process')
    parser.add_argument('--local-path', type=str, default=None,
                        help='Local path to game files (DhtmlXQ or .xqf). For external data use data/external/...')
    parser.add_argument('--category', type=str, default=None,
                        help='Filter by category folder (e.g., opening, puzzles, end-games)')
    parser.add_argument('--allow-production', action='store_true',
                        help='Allow writing directly into apps/api/data/lessons/ (use with extreme care)')

    args = parser.parse_args()

    # === XQF (external data) - Ưu tiên vì đây là nguồn phong phú nhất trong data/external ===
    if args.source == 'external-xqf' or (args.local_path and any(collect_xqf_files(args.local_path))):
        print("=== External XQF extraction (ChessQ/gamebooks and similar) ===")
        local = args.local_path or str(Path(__file__).parent.parent.parent / "data" / "external" / "ChessQ" / "gamebooks")
        print(f"Scanning: {local}")

        # Mặc định output sang file riêng, KHÔNG đụng production
        out_path = _resolve_safe_output(args, "extracted_external_chessq_endgames.json")

        lessons = process_xqf_directory(local, limit=args.limit, category_hint=args.category)
        print(f"\nSuccessfully extracted {len(lessons)} lessons from XQF files.")

        with open(out_path, 'w', encoding='utf-8') as f:
            json.dump(lessons, f, ensure_ascii=False, indent=2)

        print(f"Saved to staging file: {out_path.absolute()}")
        print("Review this file, then manually merge (or write a small merger) into apps/api/data/lessons/lessons.json")
        return 0

    # === FEN positions (xiokuai + các bộ sưu tập FEN khác) ===
    # Số lượng rất lớn (~1100+), phù hợp "Mẫu hình tàn cuộc"
    if args.source == 'external-fen' or (args.local_path and len(collect_fen_files(args.local_path)) > 5):
        print("=== External FEN extraction (position-only patterns, e.g. xiokuai) ===")
        local = args.local_path or str(Path(__file__).parent.parent.parent / "data" / "external" / "xiokuai-chinese_chess")
        print(f"Scanning: {local}")

        out_path = _resolve_safe_output(args, "extracted_external_xiokuai_fen.json")

        lessons = process_fen_directory(local, limit=args.limit)
        print(f"\nSuccessfully extracted {len(lessons)} FEN-based lessons.")

        with open(out_path, 'w', encoding='utf-8') as f:
            json.dump(lessons, f, ensure_ascii=False, indent=2)

        print(f"Saved to staging file: {out_path.absolute()}")
        print("These are mostly single-position studies. Review before merging into production lessons.json")
        return 0

    # === EGLIB / EPLIB (ChessQ text libraries - cực kỳ phong phú: 杀局谱, 适情雅趣, 基本杀法...) ===
    # Hàng nghìn vị trí từ sách danh tiếng Trung Quốc
    if args.source == 'external-eglib' or (args.local_path and len(collect_eglib_files(args.local_path)) > 0):
        print("=== External EGLIB/EPLIB extraction (ChessQ text endgame libraries) ===")
        local = args.local_path or str(Path(__file__).parent.parent.parent / "data" / "external" / "ChessQ" / "gamebooks")
        print(f"Scanning: {local}")

        out_path = _resolve_safe_output(args, "extracted_external_chessq_eglib.json")

        lessons = process_eglib_directory(local, limit=args.limit)
        print(f"\nSuccessfully extracted {len(lessons)} lessons from eglib/eplib files.")

        with open(out_path, 'w', encoding='utf-8') as f:
            json.dump(lessons, f, ensure_ascii=False, indent=2)

        print(f"Saved to staging file: {out_path.absolute()}")
        print("These come from famous books (适情雅趣, 杀局谱, 基本杀法...). Huge value for lessons.")
        return 0

    # === WXF (small number of files, standard Xiangqi notation) ===
    if args.source == 'external-wxf' or (args.local_path and len(collect_wxf_files(args.local_path)) > 0):
        print("=== External WXF extraction ===")
        local = args.local_path or str(Path(__file__).parent.parent.parent / "data" / "external")
        out_path = _resolve_safe_output(args, "extracted_external_wxf.json")
        lessons = process_wxf_directory(local, limit=args.limit)
        with open(out_path, 'w', encoding='utf-8') as f:
            json.dump(lessons, f, ensure_ascii=False, indent=2)
        print(f"Saved {len(lessons)} WXF lessons to staging file: {out_path.absolute()}")
        return 0

    # === Excel endgames (象棋残局.xlsx - rất giàu tàn cục theo chất liệu) ===
    if args.source == 'external-excel' or (args.local_path and str(args.local_path).endswith('.xlsx')):
        print("=== External Excel endgame extraction ===")
        xlsx = args.local_path or str(Path(__file__).parent.parent.parent / "data" / "external" / "chinese-chess-dataset" / "象棋残局.xlsx")
        out_path = _resolve_safe_output(args, "extracted_external_excel_endgames.json")
        lessons = process_excel_endgames(xlsx, limit=args.limit)
        with open(out_path, 'w', encoding='utf-8') as f:
            json.dump(lessons, f, ensure_ascii=False, indent=2)
        print(f"Saved {len(lessons)} Excel endgame lessons to staging file: {out_path.absolute()}")
        return 0

    # === Broad "external" – try to cover as much as possible from data/external ===
    if args.source == 'external':
        print("=== Broad external scan (XQF + FEN + EGLIB + WXF) ===")
        root = args.local_path or str(Path(__file__).parent.parent.parent / "data" / "external")
        print(f"Root: {root}")

        produced = []

        # XQF
        xqf_lessons = process_xqf_directory(root, limit=args.limit)
        if xqf_lessons:
            p = Path(__file__).parent / "extracted_external_xqf.json"
            p.write_text(json.dumps(xqf_lessons, ensure_ascii=False, indent=2), encoding="utf-8")
            produced.append(("XQF", len(xqf_lessons), p))

        # FEN
        fen_lessons = process_fen_directory(root, limit=args.limit)
        if fen_lessons:
            p = Path(__file__).parent / "extracted_external_fen.json"
            p.write_text(json.dumps(fen_lessons, ensure_ascii=False, indent=2), encoding="utf-8")
            produced.append(("FEN", len(fen_lessons), p))

        # EGLIB
        eglib_lessons = process_eglib_directory(root, limit=args.limit)
        if eglib_lessons:
            p = Path(__file__).parent / "extracted_external_eglib.json"
            p.write_text(json.dumps(eglib_lessons, ensure_ascii=False, indent=2), encoding="utf-8")
            produced.append(("EGLIB", len(eglib_lessons), p))

        # WXF
        wxf_lessons = process_wxf_directory(root, limit=args.limit)
        if wxf_lessons:
            p = Path(__file__).parent / "extracted_external_wxf.json"
            p.write_text(json.dumps(wxf_lessons, ensure_ascii=False, indent=2), encoding="utf-8")
            produced.append(("WXF", len(wxf_lessons), p))

        # Excel endgames
        excel_path = str(Path(root) / "chinese-chess-dataset" / "象棋残局.xlsx")
        if Path(excel_path).exists():
            excel_lessons = process_excel_endgames(excel_path, limit=args.limit)
            if excel_lessons:
                p = Path(__file__).parent / "extracted_external_excel_endgames.json"
                p.write_text(json.dumps(excel_lessons, ensure_ascii=False, indent=2), encoding="utf-8")
                produced.append(("Excel", len(excel_lessons), p))

        # xiangqi-book MD (improved: extracts the full annotated game from chapter 09 + theory overviews)
        book_lessons = process_xiangqi_book(root, limit=args.limit)
        if book_lessons:
            p = Path(__file__).parent / "extracted_external_xiangqi_book.json"
            p.write_text(json.dumps(book_lessons, ensure_ascii=False, indent=2), encoding="utf-8")
            produced.append(("XiangqiBook", len(book_lessons), p))

        print("\n=== Broad external scan complete ===")
        for kind, count, path in produced:
            print(f"  {kind}: {count} lessons → {path.name}")
        print("All outputs are staging files. Use merge_staging.py to import into production.")
        return 0

    # === DhtmlXQ local path (legacy path) ===
    if args.local_path:
        print(f"Collecting DhtmlXQ files from local path: {args.local_path}...")
        files = collect_dpxq_files_local(args.local_path)
        print(f"Found {len(files)} potential game files")

        lessons = process_files_local(files, limit=args.limit)
        print(f"\nSuccessfully extracted {len(lessons)} lessons")

        out_path = _resolve_safe_output(args, "extracted_lessons.json")

        with open(out_path, 'w', encoding='utf-8') as f:
            json.dump(lessons, f, ensure_ascii=False, indent=2)

        print(f"Saved to: {out_path.absolute()}")
        return 0

    if args.source == 'vietcotuong':
        print("GitHub API rate limited. Please use --local-path with a cloned repo.")
        print("Example: --local-path /path/to/.../data/opening")
        print("Useful external sources in this repo (all output to separate staging files):")
        print("  --source external-xqf     (ChessQ *.xqf)")
        print("  --source external-fen     (xiokuai *.fen)")
        print("  --source external-eglib   (ChessQ *.eglib – largest)")
        print("  --source external-wxf     (WXF files)")
        print("  --source external-excel   (Excel 象棋残局 - endgame database)")
        print("  --source external         (broad scan: all of the above + XiangqiBook)")
        return 1

    print("Source 'all' not yet implemented")
    return 1


if __name__ == '__main__':
    sys.exit(main())
