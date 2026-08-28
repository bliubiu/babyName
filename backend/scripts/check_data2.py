# -*- coding: utf-8 -*-
import json, os
from collections import Counter

base = r'D:\19-Training\learngo\name\backend\data\raw'

# hanzi.json 完整分析
with open(os.path.join(base, 'hanzi.json'), 'r', encoding='utf8') as f:
    h = json.load(f)

print(f'hanzi.json: {len(h)} 条')
print(f'字段: {list(h[0].keys())}')
print()

# radical 覆盖
rad_count = sum(1 for x in h if x.get('radical') and x['radical'].strip())
print(f'有部首信息: {rad_count}/{len(h)}')

# wuxing 覆盖
wx_count = sum(1 for x in h if x.get('wuxing') and x['wuxing'].strip())
wx_filled = sum(1 for x in h if x.get('wuxing') and x['wuxing'] in ('金','木','水','火','土'))
print(f'有五行标注: {wx_count}/{len(h)}')
print(f'五行有效: {wx_filled}/{len(h)}')
print()

# 现有五行分布
wx_dist = Counter(x['wuxing'] for x in h if x.get('wuxing') in ('金','木','水','火','土'))
for wx, n in wx_dist.most_common():
    print(f'  {wx}: {n}')

print()
# 统计部首
rad_dist = Counter(x['radical'] for x in h if x.get('radical'))
print(f'不同部首数: {len(rad_dist)}')
print(f'部首前20: {rad_dist.most_common(20)}')

# 样本
print()
print('样本前3条:')
for x in h[:3]:
    print(json.dumps(x, ensure_ascii=False))
