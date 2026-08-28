# -*- coding: utf-8 -*-
import json, os

base = r'D:\19-Training\learngo\name\backend\data\raw'

# standard_chars.json
with open(os.path.join(base, 'standard_chars.json'), 'r', encoding='utf8') as f:
    d = json.load(f)
print(f'standard_chars.json: {len(d)} 条')
if d:
    print(f'字段: {list(d[0].keys())}')
    print(f'样本: {json.dumps(d[0], ensure_ascii=False)[:200]}')

# 看繁体部首字段
has_radical = sum(1 for x in d if x.get('radical') or x.get('radical_kangxi'))
print(f'有部首信息: {has_radical}/{len(d)}')

# hanzi.json
with open(os.path.join(base, 'hanzi.json'), 'r', encoding='utf8') as f:
    h = json.load(f)
print(f'\nhanzi.json: {len(h)} 条')
if h and isinstance(h, dict):
    keys = list(h.keys())[:5]
    print(f'键样本: {keys}')
    if isinstance(h[keys[0]], dict):
        print(f'字段: {list(h[keys[0]].keys())}')
elif isinstance(h, list):
    print(f'字段: {list(h[0].keys())}')
