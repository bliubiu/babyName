# -*- coding: utf-8 -*-
import json
from collections import Counter

base = r'D:\19-Training\learngo\name\backend\data'

with open(f'{base}/hanzi.json', 'r', encoding='utf8') as f:
    h = json.load(f)

# All radicals in hanzi.json
rad_dist = Counter(x['radical'] for x in h if x.get('radical'))
print(f'Total unique radicals: {len(rad_dist)}')

# Group radicals by wuxing majority vote for each radical
rad_wx = {}
for x in h:
    r = x['radical']
    w = x['wuxing']
    if r not in rad_wx:
        rad_wx[r] = Counter()
    rad_wx[r][w] += 1

# Check which radicals I need to map
my_radicals_str = """
木 艹 竹 禾 米 瓜 耒 豆 丰 甲 乙 干
火 灬 日 月 心 忄 光 龙 马 鸟 赤 丿 丶 丨 虫
土 山 石 田 玉 王 瓦 皿 缶 宀 广 厂 尸 囗 一 二 己 戊 鹿 大 又 幺 尢 工 巨 生 用 亠 凵 冂 十 辰
金 钅 刀 刂 斤 辛 戈 酉 人 亻 士 口 贝 小 寸 卜 歹 匕 几 力 女 儿 夕 厶 庚 勹 入 车
水 氵 冫 雨 风 鱼 子 亥 文 方 川 辶 廴 冖 无 不 黑 夂 夊
"""
my_rads = set(my_radicals_str.split())

all_rads = set(rad_dist.keys())
missing = all_rads - my_rads
print(f'\nMy map: {len(my_rads)} radicals')
print(f'hanzi.json: {len(all_rads)} radicals')
print(f'Missing from my map: {len(missing)}')
print()

# Show first 50 missing radicals with their majority wuxing
sorted_missing = sorted(missing)
print('First 50 missing radicals + majority wuxing:')
for r in sorted_missing[:50]:
    majority = rad_wx[r].most_common(1)[0][0]
    sec = rad_wx[r].most_common(2)
    conf = sec[0][1] / sum(rad_wx[r].values()) * 100 if sum(rad_wx[r].values()) > 0 else 0
    second_info = f' vs {sec[1][0]}={sec[1][1]}' if len(sec) > 1 else ''
    print(f'  {r} (uni: {ord(r):04X}) → 现有多数={majority} ({conf:.0f}%){second_info}  (总{sum(rad_wx[r].values())}字)')

# Also check how hanzi.json wuxing maps: are they consistent per radical?
print(f'\n\nConsistency check - radicals with >1 dominant wuxing:')
inconsistent = []
for r, c in rad_wx.items():
    top2 = c.most_common(2)
    if len(top2) >= 2 and top2[0][1] / (top2[0][1]+top2[1][1]) < 0.7:
        inconsistent.append((r, top2[0][0], top2[0][1], top2[1][0], top2[1][1], sum(c.values())))
print(f'Radicals with low wuxing agreement (<70%): {len(inconsistent)}')
for r, w1, n1, w2, n2, tot in sorted(inconsistent, key=lambda x: -x[2])[:20]:
    print(f'  {r}: {w1}={n1}, {w2}={n2} (总{tot})')
