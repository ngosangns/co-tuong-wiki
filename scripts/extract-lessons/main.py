#!/usr/bin/env python3
"""
Pipeline extract dữ liệu ván cờ tướng từ các nguồn công khai.

Usage:
    python main.py --source vietcotuong --output ../../apps/api/data/lessons/extracted.json
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


def main():
    parser = argparse.ArgumentParser(description='Extract xiangqi game data from public sources')
    parser.add_argument('--source', choices=['vietcotuong', 'all'], default='vietcotuong',
                        help='Data source to extract from')
    parser.add_argument('--output', type=str, default='extracted_lessons.json',
                        help='Output JSON file path')
    parser.add_argument('--limit', type=int, default=None,
                        help='Limit number of files to process')
    parser.add_argument('--local-path', type=str, default=None,
                        help='Local path to DhtmlXQ files (skips GitHub API)')
    parser.add_argument('--category', type=str, default=None,
                        help='Filter by category folder (e.g., opening, puzzles, end-games)')

    args = parser.parse_args()

    if args.local_path:
        print(f"Collecting files from local path: {args.local_path}...")
        files = collect_dpxq_files_local(args.local_path)
        print(f"Found {len(files)} potential game files")

        lessons = process_files_local(files, limit=args.limit)
        print(f"\nSuccessfully extracted {len(lessons)} lessons")

        # Lưu output
        output_path = Path(args.output)
        output_path.parent.mkdir(parents=True, exist_ok=True)

        with open(output_path, 'w', encoding='utf-8') as f:
            json.dump(lessons, f, ensure_ascii=False, indent=2)

        print(f"Saved to: {output_path.absolute()}")

    elif args.source == 'vietcotuong':
        print("GitHub API rate limited. Please use --local-path with a cloned repo.")
        print("Example: --local-path /path/to/community-xiangqi-games-database/data/opening")
        return 1

    else:
        print("Source 'all' not yet implemented")
        return 1

    return 0


if __name__ == '__main__':
    sys.exit(main())
