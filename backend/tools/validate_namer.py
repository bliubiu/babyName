"""namer.json 数据校验脚本
对照 jaywcjlove/table-of-general-standard-chinese-characters 的官方数据
校验 8105 字集、拼音、多音字标记
"""
import json
import csv
import sys

def load_official_chars(url="https://raw.githubusercontent.com/jaywcjlove/table-of-general-standard-chinese-characters/main/data/characters.json"):
    """下载官方 8105 字表"""
    import urllib.request
    try:
        with urllib.request.urlopen(url, timeout=10) as f:
            return json.loads(f.read().decode('utf-8'))
    except Exception as e:
        print(f"⚠ 下载 official characters.json 失败: {e}")
        return None

def load_official_pinyin():
    """下载官方拼音数据"""
    import urllib.request
    url = "https://raw.githubusercontent.com/jaywcjlove/table-of-general-standard-chinese-characters/main/data/pinyin.json"
    try:
        with urllib.request.urlopen(url, timeout=10) as f:
            return json.loads(f.read().decode('utf-8'))
    except Exception as e:
        print(f"⚠ 下载 official pinyin.json 失败: {e}")
        return None

def load_namer(path="data/namer.json"):
    with open(path, 'r', encoding='utf-8') as f:
        return json.load(f)

def load_csv(path="data/gsc_pinyin.csv"):
    with open(path, 'r', encoding='utf-8') as f:
        return list(csv.DictReader(f))

def validate_charset(namer, official):
    """校验 8105 字集是否匹配"""
    namer_chars = {c['char'] for c in namer['chars']}
    official_set = set(official)
    
    # 1. namer 是否覆盖了所有官方字
    missing = official_set - namer_chars
    if missing:
        print(f"❌ namer.json 缺少 {len(missing)} 个官方字: {sorted(missing)[:20]}...")
    else:
        print(f"✅ namer.json 覆盖全部 {len(official_set)} 个官方字")
    
    # 2. namer 是否有多余字
    extra = namer_chars - official_set
    if extra:
        print(f"⚠ namer.json 有 {len(extra)} 个表外字: {sorted(extra)[:20]}...")
    else:
        print(f"✅ namer.json 无表外字")
    
    return namer_chars, official_set

def validate_pinyin(namer, pinyin_list):
    """校验拼音正确性"""
    namer_map = {c['char']: c for c in namer['chars']}
    
    errors = []
    poly_warnings = []
    for i, ch in enumerate(official_chars):
        if i >= len(pinyin_list):
            break
        official_py = pinyin_list[i]
        namer_entry = namer_map.get(ch)
        if not namer_entry:
            continue
        
        # official_py 可能是字符串（单音）或列表（多音）
        if isinstance(official_py, str):
            official_py_set = {official_py}
        else:
            official_py_set = set(official_py)
        
        # namer 的拼音（CSV 中多个拼音用逗号分隔）
        namer_py = namer_entry.get('pinyin', '')
        namer_py_parts = set(p.strip() for p in namer_py.split(',') if p.strip())
        
        # 检查是否有完全错误的拼音
        if namer_py_parts and not namer_py_parts.intersection(official_py_set):
            errors.append((ch, namer_py, official_py_set))
        
        # 检查多音字标记（IsPolyphonic）
        is_official_poly = isinstance(official_py, list) and len(official_py) > 1
        namer_entry = namer_map[ch]
        is_namer_poly = namer_entry.get('is_polyphonic', False)
        
        if is_official_poly and not is_namer_poly and len(official_py) > 1:
            poly_warnings.append((ch, official_py, namer_entry.get('is_polyphonic', False)))
    
    if errors:
        print(f"\n❌ 拼音不匹配 {len(errors)} 处:")
        for ch, npy, opy in errors[:15]:
            print(f"   {ch}: namer={npy}, official={opy}")
    else:
        print(f"✅ 拼音完全匹配")
    
    if poly_warnings:
        print(f"\n⚠ 可能遗漏多音字标记 {len(poly_warnings)} 处:")
        for ch, opy, _ in poly_warnings[:15]:
            print(f"   {ch}: 官方多音 {opy}, namer 未标记")

def validate_strokes(namer, csv_rows):
    """校验笔画数（CSV vs namer）"""
    csv_map = {r['word']: int(r['stroke_count']) for r in csv_rows}
    namer_map = {c['char']: c for c in namer['chars']}
    
    mismatches = []
    for ch, csv_strokes in csv_map.items():
        n = namer_map.get(ch)
        if n and n.get('strokes', 0) != csv_strokes:
            mismatches.append((ch, csv_strokes, n['strokes']))
    
    if mismatches:
        print(f"\n⚠ namer vs CSV 笔画不一致 {len(mismatches)} 处:")
        for ch, csv_s, namer_s in mismatches[:10]:
            print(f"   {ch}: CSV={csv_s}, namer={namer_s}")
    else:
        print(f"✅ 笔画数与 CSV 全部一致")

def validate_sequence(namer):
    """校验序号连续性（8105 字按序号排列）"""
    chars = namer['chars']
    prev_seq = 0
    gaps = []
    for c in chars:
        seq = c.get('seq', 0)
        if seq != prev_seq + 1:
            gaps.append((prev_seq, seq, c['char']))
        prev_seq = seq
    if gaps:
        print(f"⚠ 序号不连续 {len(gaps)} 处: {gaps[:5]}")
    else:
        print(f"✅ 序号连续 1→8105")


if __name__ == '__main__':
    print("=" * 60)
    print("namer.json 数据校验")
    print("=" * 60)
    
    # 加载本地数据
    namer = load_namer()
    csv_rows = load_csv()
    print(f"\n📂 namer.json: {namer.get('version', '?')}, {len(namer['chars'])} 字")
    print(f"📂 gsc_pinyin.csv: {len(csv_rows)} 条")
    
    # 下载官方数据校验
    official_chars = load_official_chars()
    official_pinyin = load_official_pinyin()
    
    if official_chars:
        print(f"\n--- 1. 字集校验 ---")
        validate_charset(namer, official_chars)
    
    if official_pinyin:
        print(f"\n--- 2. 拼音校验 ---")
        validate_pinyin(namer, official_pinyin)
    
    print(f"\n--- 3. 笔画数校验 ---")
    validate_strokes(namer, csv_rows)
    
    print(f"\n--- 4. 序号连续性 ---")
    validate_sequence(namer)
    
    print(f"\n{'='*60}")
    print("✅ 校验完成")
