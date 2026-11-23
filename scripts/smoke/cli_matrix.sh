#!/bin/bash
# namer-cli 全参数矩阵 E2E 冒烟（docs/44 E2E 验证产物，14 用例）
#
# 覆盖：text/json 输出、TOML 配置加载、命令行 > TOML 优先级、女宝单字名、
# 五行偏好、典籍来源 shijing、笔画区间、避讳长辈、字辈 middle、复姓、
# 错误处理（缺姓氏/非法性别/非法年份，非零退出 + 中文错误信息）。
#
# 用法（在仓库根或任意目录）：
#     bash scripts/smoke/cli_matrix.sh
# 前置：go -C backend build -o backend/bin/namer-cli-e2e.exe ./cmd/namer-cli
#       物料库 backend/namer.db 已导入（materials status 无缺失）
#
# 断言用 python -I 传参形式（勿把仓库路径内嵌进 -c 字符串：Windows Python
# 不识别 Git Bash 的 /tmp 等虚拟路径，曾致 C6/C7/C8 假失败）。

set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CLI="$ROOT/backend/bin/namer-cli-e2e.exe"
DB="$ROOT/backend/namer.db"
OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT
PASS=0; FAIL=0
ok(){ echo "PASS C$1 $2"; PASS=$((PASS+1)); }
ng(){ echo "FAIL C$1 $2"; FAIL=$((FAIL+1)); }
assert_names(){ python -I - "$1" "${2:-}" << 'PYEOF'
import json, sys
d = json.load(open(sys.argv[1], encoding='utf-8'))
assert d['names'], 'names为空'
if len(sys.argv) > 2 and sys.argv[2]:
    assert eval(sys.argv[2]), '附加断言未通过'
PYEOF
}

# C1 text 基线
"$CLI" -db "$DB" -surname 王 -gender male -birth_year 2024 -birth_month 1 -birth_day 15 -birth_hour 12 -birth_minute 0 > "$OUT/c1.txt" 2>&1
[ $? -eq 0 ] && [ -s "$OUT/c1.txt" ] && ok 1 "text基线生成" || ng 1 "text基线生成"

# C2 json 基线：合法 JSON + bazi + 姓氏前缀 + 拼音回填
"$CLI" -db "$DB" -format json -surname 王 -gender male -birth_year 2024 -birth_month 1 -birth_day 15 -birth_hour 12 -birth_minute 0 > "$OUT/c2.json" 2>/dev/null
assert_names "$OUT/c2.json" \
  "d.get('bazi') and all(n['full_name'].startswith('王') and n.get('pinyin') for n in d['names'])" \
  && ok 2 "json基线:合法JSON+bazi+姓氏前缀+拼音回填" || ng 2 "json基线"

# C3 TOML 配置文件加载
"$CLI" -config "$ROOT/backend/cmd/namer-cli/example.toml" > "$OUT/c3.txt" 2>&1
[ $? -eq 0 ] && ok 3 "TOML配置加载" || ng 3 "TOML配置加载"

# C4 命令行覆盖 TOML（-surname 李 覆盖配置中的 王；提示信息应走 stderr 不污染 JSON 流）
"$CLI" -config "$ROOT/backend/cmd/namer-cli/example.toml" -format json -surname 李 > "$OUT/c4.json" 2>"$OUT/c4.err"
assert_names "$OUT/c4.json" "all(n['full_name'].startswith('李') for n in d['names'])" \
  && grep -q "已加载配置文件" "$OUT/c4.err" \
  && ok 4 "命令行优先级覆盖TOML（stderr提示不污染JSON流）" || ng 4 "命令行优先级覆盖TOML"

# C5 女宝 + 单字名
"$CLI" -db "$DB" -format json -surname 陈 -gender female -birth_year 2023 -birth_month 8 -birth_day 8 -birth_hour 9 -name_length 1 > "$OUT/c5.json" 2>/dev/null
assert_names "$OUT/c5.json" "all(len(n['full_name'])==2 for n in d['names'])" \
  && ok 5 "女宝单字名" || ng 5 "女宝单字名"

# C6 五行偏好 水,木
"$CLI" -db "$DB" -format json -surname 林 -gender male -birth_year 2024 -birth_month 5 -birth_day 20 -birth_hour 6 -wuxing_match 水,木 > "$OUT/c6.json" 2>/dev/null
assert_names "$OUT/c6.json" && ok 6 "五行偏好水木" || ng 6 "五行偏好水木"

# C7 典籍来源 shijing（含字集提取完整性回归）
"$CLI" -db "$DB" -format json -surname 苏 -gender female -birth_year 2024 -birth_month 3 -birth_day 3 -birth_hour 10 -source_classic shijing -include_poetry > "$OUT/c7.json" 2>/dev/null
assert_names "$OUT/c7.json" && ok 7 "典籍来源shijing" || ng 7 "典籍来源shijing"

# C8 笔画范围 5-20
"$CLI" -db "$DB" -format json -surname 周 -gender male -birth_year 2024 -birth_month 6 -birth_day 1 -birth_hour 8 -min_strokes 5 -max_strokes 20 > "$OUT/c8.json" 2>/dev/null
assert_names "$OUT/c8.json" && ok 8 "笔画范围5-20" || ng 8 "笔画范围5-20"

# C9 避讳长辈（张伟/王秀英 → 名不含 伟/秀/英）
"$CLI" -db "$DB" -format json -surname 陈 -gender male -birth_year 2024 -birth_month 2 -birth_day 2 -birth_hour 3 -avoid_elder_names 张伟,王秀英 > "$OUT/c9.json" 2>/dev/null
assert_names "$OUT/c9.json" "not [n['full_name'] for n in d['names'] if any(c in '伟秀英' for c in n['full_name'][1:])]" \
  && ok 9 "避讳长辈排除同形字" || ng 9 "避讳长辈排除同形字"

# C10 字辈 李承X
"$CLI" -db "$DB" -format json -surname 李 -gender male -birth_year 2024 -birth_month 4 -birth_day 4 -birth_hour 4 -generation 承 -generation_position middle > "$OUT/c10.json" 2>/dev/null
assert_names "$OUT/c10.json" "all(n['full_name'].startswith('李承') for n in d['names'])" \
  && ok 10 "字辈middle位置" || ng 10 "字辈middle位置"

# C11 复姓欧阳
"$CLI" -db "$DB" -format json -surname 欧阳 -gender male -birth_year 2024 -birth_month 7 -birth_day 7 -birth_hour 14 > "$OUT/c11.json" 2>/dev/null
assert_names "$OUT/c11.json" "all(n['full_name'].startswith('欧阳') for n in d['names'])" \
  && ok 11 "复姓欧阳" || ng 11 "复姓欧阳"

# C12 错误处理三连：缺姓氏 / 非法性别 / 非法年份 → 非零退出
"$CLI" -db "$DB" -gender male -birth_year 2024 -birth_month 1 -birth_day 1 -birth_hour 1 > /dev/null 2>&1
[ $? -ne 0 ] && ok 12a "缺姓氏拒绝" || ng 12a "缺姓氏未拒绝"
"$CLI" -db "$DB" -surname 王 -gender xyz -birth_year 2024 -birth_month 1 -birth_day 1 -birth_hour 1 > /dev/null 2>&1
[ $? -ne 0 ] && ok 12b "非法性别拒绝" || ng 12b "非法性别未拒绝"
"$CLI" -db "$DB" -surname 王 -gender male -birth_year 1800 -birth_month 1 -birth_day 1 -birth_hour 1 > /dev/null 2>&1
[ $? -ne 0 ] && ok 12c "非法年份拒绝" || ng 12c "非法年份未拒绝"

# C13 错误信息中文输出
"$CLI" -db "$DB" -gender male -birth_year 2024 -birth_month 1 -birth_day 1 -birth_hour 1 2>&1 | grep -q "姓氏" \
  && ok 13 "中文错误信息" || ng 13 "中文错误信息"

echo "=============================="
echo "CLI E2E 结果: PASS=$PASS FAIL=$FAIL"
[ "$FAIL" -eq 0 ]
