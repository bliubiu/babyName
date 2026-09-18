# -*- coding: utf-8 -*-
"""安全修正 hexagram.go 中 64 卦 upper/lower 数字与 symbol"""
import re

MEIHUA = [
    "乾", "履", "同人", "无妄", "姤", "讼", "遁", "否",
    "夬", "兑", "革", "随", "大过", "困", "咸", "萃",
    "大有", "睽", "离", "噬嗑", "鼎", "未济", "旅", "晋",
    "大壮", "归妹", "丰", "震", "恒", "解", "小过", "豫",
    "小畜", "中孚", "家人", "益", "巽", "涣", "渐", "观",
    "需", "节", "既济", "屯", "井", "坎", "蹇", "比",
    "大畜", "损", "贲", "颐", "蛊", "蒙", "艮", "剥",
    "泰", "临", "明夷", "复", "升", "师", "谦", "坤",
]
SYM = {
    1: "☰", 2: "☱", 3: "☲", 4: "☳",
    5: "☴", 6: "☵", 7: "☶", 8: "☷",
}

def expected(name):
    i = MEIHUA.index(name)
    u, l = i // 8 + 1, i % 8 + 1
    return u, l, SYM[u] + SYM[l]

path = r"D:\19-Training\learngo\name\backend\internal\domain\yijing\hexagram.go"
with open(path, encoding="utf-8") as f:
    src = f.read()

# Match: {12, "否", 12, "☷☰", 8, 1,
pat = re.compile(r'\{(\d+), "([^"]+)", (\d+), "([^"]*)", (\d+), (\d+),')
changes = []

def repl(m):
    gid, name, num, sym, up, lo = m.group(1), m.group(2), m.group(3), m.group(4), int(m.group(5)), int(m.group(6))
    if name not in MEIHUA:
        return m.group(0)
    eu, el, es = expected(name)
    if up == eu and lo == el and sym == es:
        return m.group(0)
    changes.append((name, f"{up}/{lo}/{sym}", f"{eu}/{el}/{es}"))
    return '{%s, "%s", %s, "%s", %d, %d,' % (gid, name, num, es, eu, el)

out = pat.sub(repl, src)
with open(path, "w", encoding="utf-8") as f:
    f.write(out)

print("changed", len(changes))
for c in changes:
    print(c)
