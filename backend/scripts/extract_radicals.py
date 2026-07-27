# -*- coding: utf-8 -*-
import re
from collections import Counter, defaultdict

data = open(r'D:\19-Training\learngo\name\backend\internal\domain\hanzi\hanzi.go', 'r', encoding='utf8').read()

pattern = r'"(.+?)".*?Radical:\s*"(.+?)".*?Wuxing:\s*"(.+?)"'
matches = re.findall(pattern, data, re.DOTALL)
print(f'Total chars: {len(matches)}')

rad_counts = Counter()
rad_wuxing = defaultdict(lambda: Counter())
for char, rad, wx in matches:
    rad_counts[rad] += 1
    rad_wuxing[rad][wx] += 1

radicals_sorted = sorted(rad_counts.items(), key=lambda x: -x[1])
for rad, cnt in radicals_sorted:
    wx_dist = dict(rad_wuxing[rad])
    wx_str = ', '.join(f'{k}={v}' for k, v in sorted(wx_dist.items()))
    print(f'{rad}: {cnt} chars | {wx_str}')

print('\n--- ALL CHARACTERS WUXING COUNTS ---')
all_wx = Counter()
for char, rad, wx in matches:
    all_wx[wx] += 1
for wx, cnt in sorted(all_wx.items()):
    print(f'{wx}: {cnt}')
