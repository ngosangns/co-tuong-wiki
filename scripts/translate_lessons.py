#!/usr/bin/env python3
"""
Dịch lessons từ tiếng Trung sang tiếng Việt.
Dùng dictionary thuật ngữ cờ tướng.
"""

import json
import re
from pathlib import Path

try:
    from deep_translator import GoogleTranslator
except ImportError:
    GoogleTranslator = None

try:
    import requests
except ImportError:
    requests = None

# Dictionary các thuật ngữ cờ tướng phổ biến
CHESS_DICT = {
    # Quân cờ
    '车': 'Xe', '马': 'Mã', '炮': 'Pháo', '兵': 'Tốt', '卒': 'Tốt',
    '象': 'Tượng', '相': 'Tượng', '士': 'Sĩ', '仕': 'Sĩ', '将': 'Tướng', '帅': 'Tướng',
    '車': 'Xe', '馬': 'Mã', '砲': 'Pháo',
    
    # Động tác
    '进': 'tiến', '退': 'lùi', '平': 'bình',
    '前': 'trước', '后': 'sau', '中': 'trung',
    '杀': 'chiếu bí', '将': 'chiếu', '吃': 'ăn', '打': 'đánh', '捉': 'bắt',
    '兑': 'đổi', '弃': 'hy sinh', '伏': 'mạch', '运': 'vận',
    '攻': 'công', '守': 'thủ', '围': 'vây', '困': 'khốn',
    
    # Kết quả
    '胜': 'thắng', '负': 'thua', '和': 'hòa', '败': 'bại',
    
    # Thuật ngữ
    '局': 'cục', '着': 'nước', '步': 'bước', '变': 'biến',
    '阵': 'trận', '势': 'thế', '型': 'hình', '法': 'pháp',
    
    # Khai cuộc
    '中炮': 'Pháo Trung', '顺手炮': 'Thuận Thủ Pháo', '列手炮': 'Liệt Thủ Pháo',
    '屏风马': 'Bình Phong Mã', '反宫马': 'Phản Cung Mã', '单提马': 'Đan Đề Mã',
    '仙人指路': 'Tiên Nhân Chỉ Lộ', '起马局': 'Khởi Mã Cục', '飞相局': 'Phi Tượng Cục',
    '过宫炮': 'Quá Cung Pháo', '士角炮': 'Sĩ Giác Pháo', '巡河炮': 'Tuần Hà Pháo',
    '横车': 'Hành Xe', '直车': 'Trực Xe', '盘头马': 'Bàn Đầu Mã',
    '两头蛇': 'Lưỡng Đầu Xà', '三兵': 'Tam Binh', '七兵': 'Thất Binh',
    
    # Tàn cuộc
    '残局': 'Tàn cuộc', '杀局': 'Sát cục', '排局': 'Bài cục',
    '古谱': 'Cổ phổ', '象局': 'Tượng cục', '和局': 'Hòa cục',
    
    # Chiến thuật
    '牵制': 'khiên chế', '围困': 'vây khốn', '突破': 'đột phá',
    '反击': 'phản kích', '对攻': 'đối công', '弃子': 'hy sinh quân',
    '得子': 'được quân', '失子': 'mất quân', '攻杀': 'công sát',
    '入局': 'nhập cục', '解杀': 'giải sát', '还杀': 'hoàn sát',
    
    # Số
    '第': 'thứ', '章': 'chương', '节': 'tiết', '局': 'cục',
    '一': '1', '二': '2', '三': '3', '四': '4', '五': '5',
    '六': '6', '七': '7', '八': '8', '九': '9', '十': '10',
    
    # Từ thường gặp
    '红方': 'bên Đỏ', '黑方': 'bên Đen', '红': 'Đỏ', '黑': 'Đen',
    '方': 'bên', '手': 'thủ', '着法': 'cách đi',
    '实战': 'thực chiến', '对局': 'đối cục', '比赛': 'thi đấu',
    '这是': 'đây là', '此时': 'lúc này', '至此': 'đến đây',
    '如果': 'nếu', '否则': 'nếu không', '因为': 'bởi vì', '所以': 'cho nên',
    '可以': 'có thể', '不能': 'không thể', '必须': 'phải', '只好': 'đành',
    '但是': 'nhưng', '然而': 'tuy nhiên', '因此': 'vì vậy', '于是': 'vì thế',
    '形势': 'thế trận', '局面': 'cục diện', '战机': 'cơ hội chiến đấu',
    '先手': 'tiên thủ', '后手': 'hậu thủ', '反先': 'phản tiên',
    '子力': 'tử lực', '兵力': 'binh lực', '实力': 'thực lực',
    '弱点': 'điểm yếu', '漏洞': 'sơ hở', '缺陷': 'khuyết điểm',
    '正确': 'chính xác', '错误': 'sai lầm', '准确': 'chính xác',
    '关键': 'quan trọng', '明显': 'rõ ràng', '显然': 'hiển nhiên',
    '看似': 'tựa hồ', '实则': 'thực tế', '其实': 'kỳ thực',
    '不但': 'không những', '而且': 'mà còn', '同时': 'đồng thờii',
    '准备': 'chuẩn bị', '计划': 'kế hoạch', '意图': 'ý đồ',
    '目的': 'mục đích', '作用': 'tác dụng', '效果': 'hiệu quả',
    '变化': 'biến hóa', '演变': 'diễn biến', '发展': 'phát triển',
    '形成': 'hình thành', '导致': 'dẫn đến', '引起': 'gây ra',
    '可能': 'có thể', '应该': 'nên', '必须': 'phải', '需要': 'cần',
    '注意': 'chú ý', '留意': 'lưu ý', '小心': 'cẩn thận',
    '重视': 'coi trọng', '强调': 'nhấn mạnh', '突出': 'nổi bật',
    '清楚': 'rõ ràng', '明白': 'hiểu rõ', '了解': 'hiểu biết',
    '知道': 'biết', '掌握': 'nắm vững', '把握': 'nắm bắt', '控制': 'khống chế',
    '利用': 'lợi dụng', '运用': 'vận dụng', '使用': 'sử dụng',
    '决定': 'quyết định', '确定': 'xác định', '肯定': 'khẳng định',
    '坚持': 'kiên trì', '保持': 'duy trì', '继续': 'tiếp tục',
    '进行': 'tiến hành', '开始': 'bắt đầu', '结束': 'kết thúc',
    '停止': 'dừng', '恢复': 'khôi phục', '重新': 'làm lại',
    '以后': 'sau này', '之后': 'sau đó', '然后': 'sau đó',
    '以前': 'trước đây', '之前': 'trước đó', '当时': 'lúc đó',
    '现在': 'bây giờ', '目前': 'hiện tại', '今后': 'tương lai',
    '曾经': 'từng', '已经': 'đã', '正在': 'đang', '将要': 'sắp',
    '最': 'nhất', '更': 'càng', '非常': 'rất', '十分': 'rất',
    '特别': 'đặc biệt', '比较': 'tương đối', '稍微': 'hơi',
    '很': 'rất', '太': 'quá', '好': 'rất', '多么': 'bao nhiêu',
    '怎么': 'thế nào', '什么': 'cái gì', '为什么': 'tại sao',
    '怎样': 'như thế nào', '如何': 'như thế nào', '谁': 'ai',
    '哪里': 'ở đâu', '什么时候': 'khi nào', '多少': 'bao nhiêu',
    '几': 'mấy', '些': 'một số', '等': 'v.v.', '其他': 'khác',
    '另外': 'ngoài ra', '一切': 'tất cả', '所有': 'tất cả',
    '整个': 'toàn thể', '大量': 'đại lượng', '许多': 'nhiều',
    '很多': 'rất nhiều', '多': 'nhiều', '少': 'ít', '大': 'lớn',
    '小': 'nhỏ', '高': 'cao', '低': 'thấp', '长': 'dài', '短': 'ngắn',
    '快': 'nhanh', '慢': 'chậm', '早': 'sớm', '晚': 'muộn',
    '新': 'mới', '旧': 'cũ', '远': 'xa', '近': 'gần',
    '上': 'trên', '下': 'dưới', '左': 'trái', '右': 'phải',
    '东': 'đông', '西': 'tây', '南': 'nam', '北': 'bắc',
    '中': 'giữa', '内': 'trong', '外': 'ngoài', '前': 'trước',
    '后': 'sau', '里': 'trong', '面': 'mặt', '头': 'đầu',
    '尾': 'đuôi', '根': 'căn', '顶': 'đỉnh', '底': 'đáy',
    '正': 'chính', '反': 'phản', '对': 'đúng', '错': 'sai',
    '真': 'thật', '假': 'giả', '虚': 'hư', '实': 'thực',
    '空': 'không', '满': 'đầy', '有': 'có', '无': 'không',
    '是': 'là', '非': 'phi', '好': 'tốt', '坏': 'xấu',
    '美': 'đẹp', '丑': 'xấu', '强': 'mạnh', '弱': 'yếu',
    '硬': 'cứng', '软': 'mềm', '冷': 'lạnh', '热': 'nóng',
    '明': 'sáng', '暗': 'tối', '亮': 'sáng', '黑': 'đen',
    '白': 'trắng', '红': 'đỏ', '绿': 'xanh', '蓝': 'lam',
    '黄': 'vàng', '清': 'trong', '浊': 'đục', '纯': 'thuần',
    '杂': 'tạp', '精': 'tinh', '粗': 'thô', '细': 'tế',
    '密': 'dày', '疏': 'thưa', '紧': 'chặt', '松': 'lỏng',
    '整': 'chỉnh', '乱': 'loạn', '脏': 'bẩn', '净': 'sạch',
    '生': 'sống', '熟': 'chín', '老': 'già', '嫩': 'non',
    '香': 'thơm', '臭': 'hôi', '甜': 'ngọt', '苦': 'đắng',
    '酸': 'chua', '辣': 'cay', '咸': 'mặn', '淡': 'nhạt',
    '干': 'khô', '湿': 'ướt', '平': 'bình', '和': 'hòa',
    '温': 'ấm', '凉': 'mát', '寒': 'lạnh', '暑': 'nóng',
    '冻': 'đông', '冰': 'băng', '雪': 'tuyết', '霜': 'sương',
    '露': 'sương', '雾': 'sương mù', '云': 'mây', '雨': 'mưa',
    '风': 'gió', '雷': 'sấm', '电': 'chớp', '光': 'ánh sáng',
    '影': 'bóng', '声': 'tiếng', '音': 'âm thanh', '色': 'màu',
    '香': 'hương', '味': 'vị', '见': 'thấy', '闻': 'nghe',
    '觉': 'cảm thấy', '知': 'biết', '思': 'nghĩ', '念': 'nghĩ',
    '想': 'nghĩ', '忆': 'nhớ', '忘': 'quên', '记': 'nhớ',
    '认': 'nhận', '识': 'biết', '道': 'đạo', '懂': 'hiểu',
    '明': 'hiểu', '白': 'hiểu', '楚': 'rõ', '了': 'hiểu',
    '解': 'hiểu', '释': 'giải thích', '说': 'nói', '讲': 'nói',
    '谈': 'nói', '论': 'bàn', '议': 'bàn', '评': 'đánh giá',
    '批': 'phê bình', '判': 'phán', '断': 'đoán', '决': 'quyết',
    '定': 'định', '论': 'lý luận', '估': 'ước lượng', '计': 'tính',
    '算': 'tính', '数': 'số', '量': 'lượng', '度': 'độ',
    '衡': 'cân', '测': 'đo', '称': 'cân', '重': 'nặng',
    '轻': 'nhẹ', '比': 'so', '较': 'so sánh', '赛': 'thi',
    '竞': 'đua', '争': 'tranh', '斗': 'đấu', '战': 'chiến',
    '打': 'đánh', '击': 'đánh', '攻': 'tấn công', '守': 'phòng thủ',
    '防': 'phòng', '御': 'ngự', '卫': 'bảo vệ', '护': 'bảo vệ',
    '保': 'bảo vệ', '照': 'chiếu', '顾': 'chăm sóc', '看': 'nhìn',
    '管': 'quản lý', '理': 'quản lý', '办': 'làm', '处': 'xử lý',
    '置': 'đặt', '安': 'an', '排': 'sắp xếp', '布': 'bố trí',
    '设': 'thiết lập', '立': 'lập', '建': 'xây', '造': 'tạo',
    '制': 'làm', '作': 'làm', '干': 'làm', '活': 'hoạt động',
    '工': 'công việc', '务': 'nhiệm vụ', '职': 'chức vụ',
    '业': 'nghề', '行': 'hành động', '专': 'chuyên môn',
    '产': 'sản xuất', '加': 'thêm', '做': 'làm',
    '忙': 'bận', '闲': 'rảnh', '累': 'mệt', '乏': 'mệt',
    '困': 'buồn ngủ', '倦': 'mệt', '疲': 'mệt', '辛': 'vất vả',
    '艰': 'khó khăn', '难': 'khó', '险': 'nguy hiểm',
    '危': 'nguy hiểm', '急': 'gấp', '迫': 'ép', '紧': 'khẩn cấp',
    '及': 'kịp', '时': 'thờii gian', '准': 'đúng', '逾': 'quá',
    '期': 'kỳ hạn', '过': 'quá', '延': 'kéo dài', '拖': 'kéo',
    '耽': 'trì hoãn', '搁': 'để', '误': 'lỗi', '差': 'khác',
    '谬': 'sai', '偏': 'lệch', '恰': 'đúng', '当': 'đúng',
    '适': 'phù hợp', '合': 'hợp', '宜': 'nên', '巧': 'tình cờ',
    '妙': 'tuyệt', '绝': 'tuyệt đối', '美': 'đẹp', '奇': 'lạ',
    '特': 'đặc biệt', '怪': 'lạ', '异': 'khác', '罕': 'hiếm',
    '见': 'thấy', '珍': 'quý', '贵': 'quý', '宝': 'báu vật',
    '珠': 'ngọc', '财': 'tài sản', '富': 'giàu', '豪': 'hào hoa',
    '华': 'hoa lệ', '丽': 'đẹp', '漂': 'đẹp', '观': 'xem',
    '完': 'hoàn thành', '善': 'tốt', '良': 'tốt', '优': 'tốt',
    '秀': 'xuất sắc', '杰': 'kiệt xuất', '出': 'xuất hiện',
    '卓': 'xuất sắc', '越': 'vượt', '伟': 'vĩ đại', '宏': 'vĩ đại',
    '壮': 'hùng tráng', '雄': 'anh hùng', '磅': 'mạnh mẽ',
    '礴': 'mạnh mẽ', '气': 'khí', '派': 'phái', '足': 'đủ',
    '汹': 'dữ dội', '恢': 'rộng lớn', '派': 'phong cách',
    '足': 'đủ', '恢': 'khôi phục',
}

TRANSLATION_CACHE: dict[str, str] = {}
TRANSLATABLE_LESSON_FIELDS = {
    'title',
    'comment',
    'prompt',
    'label',
    'feedback',
}

POST_REPLACEMENTS = {
    'Hiệp': 'Bài',
    'Mất': 'thua',
    'không rõ': 'chưa rõ',
    'không xác định': 'chưa rõ',
    'chưa xác định': 'chưa rõ',
    'Hồng Sinh': 'Đỏ thắng',
    'Red win': 'Đỏ thắng',
    'Black win': 'Đen thắng',
    'ngựa': 'Mã',
    'Ngựa': 'Mã',
    'đại bác': 'Pháo',
    'Đại bác': 'Pháo',
    'pháo binh': 'Pháo',
    'Pháo binh': 'Pháo',
    'súng thần công': 'Pháo',
    'súng': 'Pháo',
    'Súng': 'Pháo',
    'con tốt': 'Tốt',
    'quân tốt': 'Tốt',
    'lính': 'quân',
    'Lính': 'Quân',
    'vào game': 'nhập cục',
    'vào trò chơi': 'nhập cục',
    'lấy đà': 'giành thế',
    'Bỏ': 'Thí',
}


def has_chinese(text: str) -> bool:
    return any('\u3400' <= c <= '\u9fff' for c in text)


def cleanup_vietnamese(text: str) -> str:
    """Làm mượt các cụm dịch máy thường gặp trong dữ liệu cờ tướng."""
    if not text:
        return text

    result = text
    for source, target in POST_REPLACEMENTS.items():
        result = result.replace(source, target)

    result = result.replace('Class', 'nhóm')
    result = result.replace('take thế', 'giành thế')
    result = result.replace('địa', '')
    result = result.replace('无biết', 'chưa rõ')
    result = result.replace('未biết', 'chưa rõ')
    result = result.replace('mới疆', 'Tân Cương')
    result = result.replace('trên海', 'Thượng Hải')
    result = result.replace('河bắc', 'Hà Bắc')
    result = result.replace('湖nam', 'Hồ Nam')
    result = result.replace('山đông', 'Sơn Đông')
    result = result.replace('福xây', 'Phúc Kiến')
    result = result.replace('nặng庆', 'Trùng Khánh')
    result = result.replace('厦门', 'Hạ Môn')
    result = result.replace('江苏', 'Giang Tô')
    result = result.replace('浙江', 'Chiết Giang')
    result = result.replace('广东', 'Quảng Đông')
    result = result.replace('黑龙江', 'Hắc Long Giang')
    result = result.replace('吉林', 'Cát Lâm')
    result = result.replace('北京', 'Bắc Kinh')

    result = re.sub(r'\bthứ\s*(\d+)\s*cục\b', r'Bài \1', result, flags=re.IGNORECASE)
    result = re.sub(r'\bBài\s*(\d+)\s*cục\b', r'Bài \1', result, flags=re.IGNORECASE)
    result = re.sub(r'\bBài\s*(\d+)\s*địa\b', r'Bài \1', result, flags=re.IGNORECASE)
    result = re.sub(r'\bđỏ\s*thắng\b', 'Đỏ thắng', result, flags=re.IGNORECASE)
    result = re.sub(r'\bđen\s*thắng\b', 'Đen thắng', result, flags=re.IGNORECASE)
    result = re.sub(r'\b未\s*biết\b', 'chưa rõ', result, flags=re.IGNORECASE)
    result = re.sub(r'\bchưa\s*rõ\s*kết\s*quả\b', 'chưa rõ kết quả', result, flags=re.IGNORECASE)
    piece_names = r'(Xe|Mã|Pháo|Tốt|Sĩ|Tượng|Tướng)'
    result = re.sub(rf'([a-zà-ỹ])(?={piece_names}\b)', r'\1 ', result)
    result = re.sub(rf'{piece_names}(?=[a-zà-ỹ])', r'\1 ', result)
    result = re.sub(r'\s+', ' ', result)
    result = re.sub(r'\s+([,.;:!?])', r'\1', result)
    result = re.sub(r'\s*-\s*', ' - ', result)
    return result.strip()


def looks_bad_translation(text: str) -> bool:
    lowered = text.lower()
    bad_fragments = [
        'take',
        'unknown',
        'kind',
        'class',
        'gấptiến',
        'tiếntấn',
        'đúngphản',
        'đúnghành',
        'đúngtrực',
        'đúngtốt',
        'đúngtrái',
        'đúngphải',
        'trunggấp',
        'trungđúng',
        'sautrái',
        'tốigiữa',
        'tranhhy',
        'lạnước',
        'dướinước',
        'thưa có thể',
        'nhập phương',
        'cung cấp mã',
        'phản ứng mã',
        'thứ1',
        'thứ2',
        'thứ3',
    ]
    return any(fragment in lowered for fragment in bad_fragments)


def sanitize_user_facing_text(lesson: dict, index: int) -> None:
    """Ensure visible lesson copy is Vietnamese even when source translation fails."""
    category = cleanup_vietnamese(str(lesson.get('category') or 'Bài học'))
    display_number = index + 1

    if (
        has_chinese(str(lesson.get('title', '')))
        or looks_bad_translation(str(lesson.get('title', '')))
        or len(str(lesson.get('title', '')).strip()) < 3
    ):
        lesson['title'] = f"Bài {display_number}: {category}"
    else:
        lesson['title'] = cleanup_vietnamese(lesson['title'])

    lesson.pop('summary', None)
    lesson.pop('principles', None)
    lesson.pop('tags', None)

    for line_index, line in enumerate(lesson.get('lines', []), start=1):
        line_title = cleanup_vietnamese(str(line.get('title') or 'Biến chính'))
        if has_chinese(line_title) or looks_bad_translation(line_title):
            line_title = 'Biến chính' if line_index == 1 else f'Biến {line_index}'
        line['title'] = line_title
        line.pop('description', None)

        for move_index, move in enumerate(line.get('moves', []), start=1):
            title = cleanup_vietnamese(str(move.get('title') or ''))
            if has_chinese(title) or looks_bad_translation(title) or not title:
                title = f"Nước đi {move_index}"
            move['title'] = title

            comment = cleanup_vietnamese(str(move.get('comment') or ''))
            if has_chinese(comment) or looks_bad_translation(comment):
                comment = "Nước đi tiếp tục mạch chính, cần chú ý nhịp phối hợp quân và thế chủ động."
            move['comment'] = comment

    choice = lesson.get('choice') or {}
    prompt = cleanup_vietnamese(str(choice.get('prompt') or ''))
    if has_chinese(prompt) or looks_bad_translation(prompt) or not prompt:
        choice['prompt'] = "Bạn đánh giá nước đi nào tốt nhất trong ván cờ này?"
    else:
        choice['prompt'] = prompt

    for option in choice.get('options', []):
        label = cleanup_vietnamese(str(option.get('label') or ''))
        if has_chinese(label) or looks_bad_translation(label) or not label:
            option['label'] = "Theo biến chính"
        else:
            option['label'] = label

        feedback = cleanup_vietnamese(str(option.get('feedback') or ''))
        if has_chinese(feedback) or looks_bad_translation(feedback) or not feedback:
            option['feedback'] = "Đây là lựa chọn đi theo mạch phân tích chính của bài."
        else:
            option['feedback'] = feedback


def translate_text(text: str, translator=None) -> str:
    """Dịch text sang tiếng Việt, ưu tiên dịch theo cụm rồi chuẩn hóa thuật ngữ."""
    if not text or not text.strip():
        return text

    if not has_chinese(text):
        return cleanup_vietnamese(text)

    if text in TRANSLATION_CACHE:
        return TRANSLATION_CACHE[text]

    if translator is not None:
        try:
            translated = translator.translate(text)
            result = cleanup_vietnamese(translated)
            TRANSLATION_CACHE[text] = result
            return result
        except Exception as error:
            print(f"    fallback dictionary translation for {text!r}: {error}")

    result = text
    sorted_terms = sorted(CHESS_DICT.keys(), key=len, reverse=True)
    for term in sorted_terms:
        if term in result:
            result = result.replace(term, CHESS_DICT[term])

    result = cleanup_vietnamese(result)
    TRANSLATION_CACHE[text] = result
    return result


def collect_chinese_texts(value) -> set[str]:
    """Collect user-facing strings that still need translation."""
    texts: set[str] = set()

    def visit(node, key=''):
        if isinstance(node, dict):
            for child_key, child_value in node.items():
                visit(child_value, child_key)
        elif isinstance(node, list):
            for child in node:
                visit(child, key)
        elif isinstance(node, str) and key in TRANSLATABLE_LESSON_FIELDS and has_chinese(node):
            texts.add(node)

    visit(value)
    return texts


def translate_many(texts: list[str]) -> None:
    """Populate translation cache using batched Google Translate requests."""
    if not texts or requests is None:
        return

    separator = '\n###\n'
    batch: list[str] = []
    batch_size = 80
    translated_count = 0
    failed_count = 0

    def flush() -> None:
        nonlocal batch, translated_count, failed_count
        if not batch:
            return

        joined = separator.join(batch)
        try:
            response = requests.get(
                'https://translate.googleapis.com/translate_a/single',
                params={
                    'client': 'gtx',
                    'sl': 'zh-CN',
                    'tl': 'vi',
                    'dt': 't',
                    'q': joined,
                },
                timeout=10,
            )
            response.raise_for_status()
            translated = ''.join(part[0] for part in response.json()[0] if part[0])
            parts = translated.split(separator)
            if len(parts) != len(batch):
                raise ValueError(f'expected {len(batch)} translations, got {len(parts)}')

            for source, target in zip(batch, parts):
                TRANSLATION_CACHE[source] = cleanup_vietnamese(target)
            translated_count += len(batch)
        except Exception as error:
            error_name = type(error).__name__
            print(f"    batch translation fallback for {len(batch)} texts: {error_name}")
            failed_count += len(batch)

        batch = []
        print(f"    translated cache: {translated_count}/{len(texts)}; fallback pending: {failed_count}")

    for text in texts:
        batch.append(text)
        if len(batch) >= batch_size or len(separator.join(batch)) > 4500:
            flush()

    flush()


def translate_lesson(lesson: dict, translator=None, index: int = 0) -> dict:
    """Dịch một lesson."""
    # Dịch title
    if lesson.get('title'):
        lesson['title'] = translate_text(lesson['title'], translator)
    
    # Dịch lines
    for line in lesson.get('lines', []):
        if line.get('title'):
            line['title'] = translate_text(line['title'], translator)

        # Dịch moves comments
        for move in line.get('moves', []):
            if move.get('title'):
                move['title'] = translate_text(move['title'], translator)

            if move.get('comment'):
                move['comment'] = translate_text(move['comment'], translator)
    
    # Dịch choice
    if lesson.get('choice'):
        if lesson['choice'].get('prompt'):
            lesson['choice']['prompt'] = translate_text(lesson['choice']['prompt'], translator)
        
        for option in lesson['choice'].get('options', []):
            if option.get('label'):
                option['label'] = translate_text(option['label'], translator)

            if option.get('feedback'):
                option['feedback'] = translate_text(option['feedback'], translator)
    
    sanitize_user_facing_text(lesson, index)
    return lesson


def main():
    lessons_path = Path('apps/api/data/lessons/lessons.json')
    
    with open(lessons_path, 'r', encoding='utf-8') as f:
        lessons = json.load(f)
    
    untranslated = sorted(collect_chinese_texts(lessons), key=len)
    print(f"Pre-translating {len(untranslated)} unique Chinese user-facing strings...")
    translate_many(untranslated)

    translator = None
    print(f"Translating {len(lessons)} lessons...")
    print("Using pre-translated cache plus dictionary fallback.")
    
    for i, lesson in enumerate(lessons):
        if i % 50 == 0:
            print(f"  [{i}/{len(lessons)}] {lesson.get('id', '?')}")
        
        translate_lesson(lesson, translator, i)
    
    # Backup
    backup_path = lessons_path.with_suffix('.json.backup')
    with open(backup_path, 'w', encoding='utf-8') as f:
        json.dump(lessons, f, ensure_ascii=False, indent=2)
    
    with open(lessons_path, 'w', encoding='utf-8') as f:
        json.dump(lessons, f, ensure_ascii=False, indent=2)
    
    print(f"Done! Saved to {lessons_path}")
    print(f"Backup at {backup_path}")


if __name__ == '__main__':
    main()
