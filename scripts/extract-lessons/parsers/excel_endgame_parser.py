"""
Parser cho file Excel tàn cục "象棋残局.xlsx" từ chinese-chess-dataset.

Nguồn: https://huggingface.co/datasets/di-zhang-fdu/Chinese-chess

Cấu trúc:
- Mỗi dòng = một loại tàn cục (theo chất liệu, ví dụ "炮 vs 卒卒卒象")
- Có tên tiếng Trung, thống kê thắng/hòa/thua
- Nhiều FEN cho từng loại (cột "最长局面（红先）", "最长局面（黑先）", ...)

Parser này tạo lesson theo kiểu:
- 1 lesson cho mỗi loại tàn cục (dùng "中文名" làm title)
- Category: "Tàn cuộc"
- 2 variations: "Đỏ tiên thủ" và "Đen tiên thủ" với FEN tương ứng

Điều này tạo ra hàng trăm bài học tàn cục chất lượng cao mà không làm phình lessons.json.

Không yêu cầu pandas/openpyxl (dùng zip + XML thuần).
"""

from __future__ import annotations

import re
import uuid
import zipfile
import xml.etree.ElementTree as ET
from pathlib import Path
from typing import List, Dict, Any, Optional


def _get_shared_strings(xlsx_path: Path) -> List[str]:
    with zipfile.ZipFile(xlsx_path, 'r') as z:
        if 'xl/sharedStrings.xml' not in z.namelist():
            return []
        ss_xml = z.read('xl/sharedStrings.xml')
        root = ET.fromstring(ss_xml)
        ns = {'main': 'http://schemas.openxmlformats.org/spreadsheetml/2006/main'}
        texts = []
        for t in root.findall('.//main:t', ns):
            texts.append(t.text or "")
        return texts


def _parse_sheet(xlsx_path: Path, shared: List[str]) -> List[List[str]]:
    """Trả về list of rows, mỗi row là list các giá trị string."""
    with zipfile.ZipFile(xlsx_path, 'r') as z:
        sheet_path = 'xl/worksheets/sheet1.xml'
        if sheet_path not in z.namelist():
            return []
        sheet_xml = z.read(sheet_path)
        root = ET.fromstring(sheet_xml)
        ns = {'main': 'http://schemas.openxmlformats.org/spreadsheetml/2006/main'}

        rows = []
        for row in root.findall('.//main:row', ns):
            row_vals = []
            for cell in row.findall('main:c', ns):
                v = cell.find('main:v', ns)
                if v is not None and v.text:
                    try:
                        idx = int(v.text)
                        val = shared[idx] if idx < len(shared) else v.text
                    except ValueError:
                        val = v.text
                else:
                    val = ""
                row_vals.append(val)
            rows.append(row_vals)
        return rows


def _is_fen(s: str) -> bool:
    if not s or "/" not in s:
        return False
    # Xiangqi FEN thường có 9 "/" và khá dài
    return s.count("/") >= 8 and len(s) > 25


def process_excel_endgames(
    xlsx_path: str | Path,
    limit: Optional[int] = None,
    sample_per_type: int = 2,
) -> List[dict]:
    """
    Parse Excel endgame dataset thành lessons.

    Mỗi loại tàn cục (một dòng) → 1 lesson, với 1-2 FEN đại diện.
    """
    xlsx_path = Path(xlsx_path)
    if not xlsx_path.exists():
        print(f"[Excel] File not found: {xlsx_path}")
        return []

    try:
        shared = _get_shared_strings(xlsx_path)
        rows = _parse_sheet(xlsx_path, shared)
    except Exception as e:
        print(f"[Excel] Failed to parse: {e}")
        return []

    if not rows:
        return []

    # Bỏ header
    data_rows = rows[1:] if len(rows) > 1 else rows

    lessons = []
    processed = 0

    for row in data_rows:
        if limit and processed >= limit:
            break

        if len(row) < 8:
            continue

        material = row[0].strip()          # KCKBPPP
        chinese_name = row[1].strip()      # "炮 vs 卒卒卒象"
        red_fen = row[5].strip() if len(row) > 5 else ""
        black_fen = row[6].strip() if len(row) > 6 else ""

        fens = []
        if _is_fen(red_fen):
            fens.append(("Đỏ tiên thủ", red_fen))
        if _is_fen(black_fen) and black_fen != red_fen:
            fens.append(("Đen tiên thủ", black_fen))

        if not fens:
            continue

        # Lấy thêm FEN khác trong dòng nếu có
        extra_fens = [v for v in row if _is_fen(v) and v not in [f[1] for f in fens]]
        for ef in extra_fens[:max(0, sample_per_type - len(fens))]:
            fens.append(("Vị trí khác", ef))

        # Tạo title đẹp
        title = chinese_name
        if material and material not in chinese_name:
            title = f"{chinese_name} ({material})"

        lesson = {
            "id": str(uuid.uuid4()),
            "title": title,
            "category": "Tàn cuộc",
            "difficulty": "Trung cấp",
            "source": {
                "type": "excel",
                "file": str(xlsx_path),
                "material": material,
                "original_name": chinese_name,
            },
            "lines": [],
        }

        for label, fen in fens[:sample_per_type]:
            lesson["lines"].append({
                "id": str(uuid.uuid4()),
                "title": label,
                "moves": [],
            })
            # Gắn FEN vào line (frontend hiện hỗ trợ initialFen ở lesson level, nhưng ta có thể để ở đây
            # hoặc promote FEN đầu tiên lên lesson level
            if "initialFen" not in lesson:
                lesson["initialFen"] = fen

        lessons.append(lesson)
        processed += 1

    print(f"[Excel] Extracted {len(lessons)} endgame lessons from {xlsx_path.name}")
    return lessons
