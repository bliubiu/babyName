#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
从 data/raw/ 下的两份清洗清单生成 data/forbidden_combos.json

输入:
    data/raw/清洗清单-门禁虚字.txt    （595 行 / 双字组合）
    data/raw/清洗清单-荒谬组合.txt    （367 行 / 双字组合）

输出:
    data/forbidden_combos.json        （962 条双字组合 JSON 数组）

被 internal/domain/fate/semantic_filter.go 的 LoadForbiddenCombosFromJSON 加载，
与 forbiddenCombos 硬编码常量合并参与 IsBadCombo 判定。

背景:
    仓库内 data/raw/清洗清单-*.txt 是历史清洗成果（合计 962 条荒谬双字组合），
    对应引擎注释中"混入仲尼/若兮/七政/与砺 等 962 条垃圾"被剔除的部分。
    该脚本把它们从 .txt 重组为运行时加载的 JSON，使 IsBadCombo 可消费。

运行:
    python scripts/build_forbidden_combos.py
"""

import json
import os
import sys


def main() -> int:
    # 脚本位于 backend/scripts/，相对路径基于 backend/ 根
    here = os.path.dirname(os.path.abspath(__file__))
    backend_root = os.path.abspath(os.path.join(here, ".."))
    raw_dir = os.path.join(backend_root, "data", "raw")
    out_path = os.path.join(backend_root, "data", "forbidden_combos.json")

    sources = [
        os.path.join(raw_dir, "清洗清单-门禁虚字.txt"),
        os.path.join(raw_dir, "清洗清单-荒谬组合.txt"),
    ]

    combos: set[str] = set()
    for path in sources:
        if not os.path.isfile(path):
            print(f"跳过缺失文件: {path}", file=sys.stderr)
            continue
        with open(path, encoding="utf-8") as fp:
            for line in fp:
                w = line.strip()
                # 只收双字组合；空行/其他行忽略
                if len(w) == 2:
                    combos.add(w)

    combos_sorted = sorted(combos)
    os.makedirs(os.path.dirname(out_path), exist_ok=True)
    with open(out_path, "w", encoding="utf-8") as fp:
        json.dump(combos_sorted, fp, ensure_ascii=False, indent=2)

    print(f"已生成 {len(combos_sorted)} 条禁忌组合 → {out_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())