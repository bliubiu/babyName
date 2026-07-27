"""
诗词曲类 JSON 统一格式转换器

统一格式：
[
  {
    "title": "标题",
    "author": "作者",
    "dynasty": "唐",
    "type": "诗/词/曲/辞赋/乐府",
    "book": "来源集子（可选）",
    "source": "来源分类",
    "tags": ["标签"],
    "paragraphs": ["诗句1", "诗句2"]
  }
]

处理文件：
  data/shici.json       (822) - 已有格式，规范化
  data/cifu.json        (47)  - content→paragraphs
  data/yuefu.json       (203) - content→paragraphs
  data/yuanqu.json      (11057) - dynasty 英→中
  蒙学/tangshisanbaishou.json (320) - 提取并合并到 shici.json
"""
import json
import re
from pathlib import Path

DATA_DIR = Path(__file__).resolve().parent.parent / "data"
MX_DIR = DATA_DIR / "蒙学"

def save_json(path, data):
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"  ✓ {path.relative_to(DATA_DIR)}")

def split_content_to_paragraphs(text):
    """将连续文本按句分割为段落数组"""
    # 先按换行或全角空格分割
    lines = re.split(r'[　\s]+', text.strip())
    # 过滤空行
    lines = [l.strip() for l in lines if l.strip()]
    if not lines:
        # 按句号分句
        lines = [s.strip() + "。" for s in re.split(r'[。！？；]', text) if s.strip()]
    return lines if lines else [text.strip()]

def main():
    print("=" * 50)
    print("诗词曲类 JSON 统一格式转换")
    print("=" * 50)

    # 1. 加载现有 shici.json（目标文件）
    shici_path = DATA_DIR / "shici.json"
    shici_data = json.loads(shici_path.read_text(encoding="utf-8"))
    existing_titles = set(x["title"] for x in shici_data)
    print(f"\n[1] shici.json 现有: {len(shici_data)} 首")

    # ======== cifu.json (47篇辞赋) ========
    cifu = json.loads((DATA_DIR / "cifu.json").read_text(encoding="utf-8"))
    cifu_converted = []
    for item in cifu:
        content = item.get("content", "")
        paragraphs = split_content_to_paragraphs(content)
        entry = {
            "title": item.get("title", ""),
            "author": item.get("author", ""),
            "dynasty": item.get("dynasty", "").replace("代", ""),
            "type": "辞赋",
            "book": item.get("book", ""),
            "source": item.get("book", "辞赋"),
            "tags": [item.get("dynasty", ""), item.get("book", "辞赋")],
            "paragraphs": paragraphs,
        }
        if entry["title"] not in existing_titles:
            cifu_converted.append(entry)
    print(f"\n[2] cifu.json: {len(cifu)} 篇 → {len(cifu_converted)} 篇新增")

    # ======== yuefu.json (203首乐府) ========
    yuefu = json.loads((DATA_DIR / "yuefu.json").read_text(encoding="utf-8"))
    yuefu_converted = []
    for item in yuefu:
        content = item.get("content", "")
        paragraphs = split_content_to_paragraphs(content)
        entry = {
            "title": item.get("title", ""),
            "author": item.get("author", ""),
            "dynasty": item.get("dynasty", "").replace("代", ""),
            "type": "乐府",
            "book": item.get("book", ""),
            "source": item.get("book", "乐府"),
            "tags": [item.get("dynasty", ""), item.get("book", "乐府")],
            "paragraphs": paragraphs,
        }
        if entry["title"] not in existing_titles:
            yuefu_converted.append(entry)
    print(f"\n[3] yuefu.json: {len(yuefu)} 首 → {len(yuefu_converted)} 首新增")

    # ======== yuanqu.json (11057首元曲) ========
    dynasty_map = {"yuan": "元", "ming": "明", "qing": "清"}
    yuanqu = json.loads((DATA_DIR / "yuanqu.json").read_text(encoding="utf-8"))
    yuanqu_converted = []
    for item in yuanqu:
        entry = {
            "title": item.get("title", ""),
            "author": item.get("author", ""),
            "dynasty": dynasty_map.get(item.get("dynasty", ""), item.get("dynasty", "")),
            "type": "元曲",
            "source": "元曲",
            "tags": ["元曲", item.get("dynasty", "")],
            "paragraphs": item.get("paragraphs", []),
        }
        if entry["title"] not in existing_titles:
            yuanqu_converted.append(entry)
    print(f"\n[4] yuanqu.json: {len(yuanqu)} 首 → {len(yuanqu_converted)} 首新增")

    # ======== 蒙学/tangshisanbaishou.json (320首唐诗) ========
    tangshi_path = MX_DIR / "tangshisanbaishou.json"
    if tangshi_path.exists():
        tangshi = json.loads(tangshi_path.read_text(encoding="utf-8"))
        tangshi_converted = []
        for item in tangshi.get("content", []):
            # chapter 格式 "五言絕句·行宮"
            chapter = item.get("chapter", "")
            parts = chapter.split("·") if "·" in chapter else [chapter]
            poem_type = parts[0] if len(parts) > 1 else ""
            poem_title = parts[-1]
            # 去重
            if poem_title in existing_titles or poem_title in [x.get("title","") for x in tangshi_converted]:
                continue
            entry = {
                "title": poem_title,
                "author": item.get("author", "").replace("唐代：", "").replace(" ", ""),
                "dynasty": "唐",
                "type": poem_type,
                "source": "唐诗三百首",
                "book": "唐诗三百首",
                "tags": ["唐诗三百首", poem_type],
                "paragraphs": item.get("paragraphs", []),
            }
            tangshi_converted.append(entry)

        print(f"\n[5] 蒙学/tangshisanbaishou.json: {len(tangshi['content'])} 首原 → {len(tangshi_converted)} 首新增")
    else:
        tangshi_converted = []
        print(f"\n[5] 蒙学/tangshisanbaishou.json: 不存在，跳过")

    # ======== 合并 shici.json ========
    all_new = cifu_converted + yuefu_converted + yuanqu_converted + tangshi_converted
    merged = shici_data + all_new

    # 按 type 统计
    from collections import Counter
    type_counter = Counter(x["type"] for x in merged)
    print(f"\n[6] 合并后 shici.json:")
    print(f"  原有: {len(shici_data)} 首")
    print(f"  新增: {len(all_new)} 首")
    print(f"  总计: {len(merged)} 首")
    print(f"  类型分布:")
    for t, cnt in sorted(type_counter.items(), key=lambda x: -x[1]):
        print(f"    {t}: {cnt}")

    # 写回
    save_json(shici_path, merged)

    # ======== 清理空文件（cifu.json, yuefu.json, yuanqu.json 内容已合并到 shici.json） ========
    # 转换为空数组标记，保留文件存在
    for name in ["cifu.json", "yuefu.json", "yuanqu.json"]:
        path = DATA_DIR / name
        save_json(path, [])
        print(f"  → {name} 已清空（数据已合并到 shici.json）")

    # 从蒙学删除 tangshisanbaishou.json（已合并）
    if tangshi_path.exists():
        tangshi_path.unlink()
        print(f"  → 蒙学/tangshisanbaishou.json 已删除")

    print("\n转换完成!")

if __name__ == "__main__":
    main()
