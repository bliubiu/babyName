import json
from pathlib import Path

DATA = Path(r'D:\19-Training\learngo\name\backend\data')
MX = DATA / '蒙学'

# Check tangshisanbaishou
t = json.loads((MX / 'tangshisanbaishou.json').read_text(encoding='utf-8'))
print(f'tangshisanbaishou.json: {len(t["content"])} 条')
print(f'  格式: title={t["title"]}')
# Check first few to understand structure
for c in t['content'][:3]:
    print(f'  条目: chapter={c.get("chapter","?")} source={c.get("source","")} author={c.get("author","")}')

# Check shici.json for overlap
s = json.loads((DATA / 'shici.json').read_text(encoding='utf-8'))
shici_titles = set(x['title'] for x in s)
tangshi_titles = set(c.get('chapter','') for c in t['content'])
overlap = shici_titles & tangshi_titles
print(f'\nshici.json: {len(s)} 首')
print(f'tangshisanbaishou.json (after flatten): {len(t["content"])} 首')
print(f'重复: {len(overlap)} 首')
if overlap:
    print(f'  例: {list(overlap)[:3]}')
