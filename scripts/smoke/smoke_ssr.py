# -*- coding: utf-8 -*-
"""SSR 链路冒烟：验证 attachSurname 移除后全名仍正确（走服务层回填）。"""
import json
import os
import urllib.request
import urllib.parse
import http.cookiejar
import re
import sys

BASE = os.environ.get("SMOKE_BASE", "http://127.0.0.1:8080")
cj = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))

def get(path):
    with opener.open(BASE + path, timeout=60) as r:
        return r.status, r.read().decode("utf-8", errors="replace")

def post(path, data):
    body = urllib.parse.urlencode(data).encode("utf-8")
    req = urllib.request.Request(BASE + path, data=body, method="POST")
    with opener.open(req, timeout=60) as r:
        return r.status, r.read().decode("utf-8", errors="replace")

out = []
try:
    st, home = get("/")
    out.append("GET / -> %d, len=%d, form=%s" % (st, len(home), "/generate" in home))

    form = {
        "surname": "李",
        "gender": "female",
        "birth_year": "2023", "birth_month": "8", "birth_day": "20",
        "birth_hour": "10", "birth_minute": "30",
        "birth_location": "北京",
        "name_length": "2",
        "exclude_rare": "1",
        "enable_cantonese": "1",
    }
    st, _ = post("/generate", form)
    out.append("POST /generate -> %d" % st)

    st, res = get("/result")
    out.append("GET /result -> %d, len=%d" % (st, len(res)))

    # 关键断言1：不应出现模板渲染失败
    out.append("render_fail_absent=%s" % ("模板渲染失败" not in res))
    # 关键断言2：八字/五行区块在
    for kw in ["八字五行", "四柱", "生肖", "喜用神", "日主"]:
        out.append("has[%s]=%s" % (kw, kw in res))
    # 关键断言3：候选名 h3 形如「李XX」——证明服务层 FullName 回填生效
    m = re.findall(r'<h3 class="name-big">([^<]+?)\s*<span', res)
    out.append("candidate_names=%s" % json.dumps(m[:5], ensure_ascii=False))
    out.append("fullname_surname_prefix_ok=%s" % (
        len(m) > 0 and all(n.startswith("李") for n in m)))
    # 关键断言4：换一批表单带 generation_id
    out.append("explore_gen_id=%s" % ('name="generation_id"' in res and 'value="g' in res))
    # 关键断言5：收藏表单隐藏字段带姓氏
    m2 = re.findall(r'name="surname" value="([^"]*)"', res)
    out.append("fav_surname_hidden=%s" % json.dumps(sorted(set(m2))[:3], ensure_ascii=False))

    # 换一批
    gm = re.search(r'name="generation_id" value="([^"]+)"', res)
    if gm:
        st, _ = post("/explore", {"generation_id": gm.group(1)})
        st2, res2 = get("/result")
        m3 = re.findall(r'<h3 class="name-big">([^<]+?)\s*<span', res2)
        out.append("POST /explore -> %d ; after GET /result -> %d ; names=%s" % (
            st, st2, json.dumps(m3[:3], ensure_ascii=False)))
        out.append("explore_fullname_ok=%s" % (len(m3) > 0 and all(n.startswith("李") for n in m3)))

    # 收藏第一个候选
    sm = re.search(r'name="given_name" value="([^"]+)"', res)
    if sm:
        st, _ = post("/favorites", {
            "surname": "李", "given_name": sm.group(1),
            "pinyin": "li", "gender": "female",
            "score": "90", "source": "生成",
        })
        out.append("POST /favorites -> %d" % st)
        st, fav = get("/favorites")
        out.append("GET /favorites -> %d, len=%d, has_entry=%s" % (
            st, len(fav), "取消收藏" in fav))

    st, hist = get("/history")
    out.append("GET /history -> %d, len=%d, has_entry=%s" % (
        st, len(hist), "个名字" in hist))

except Exception as e:
    out.append("EXCEPTION: %r" % e)

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
OUT_DIR = os.path.join(REPO_ROOT, ".workbuddy", "tmp")
os.makedirs(OUT_DIR, exist_ok=True)
OUT = os.path.join(OUT_DIR, "smoke_ssr.txt")
joined = "\n".join(out)
# 关键断言：无异常 + 渲染未失败 + 候选名带姓前缀（服务层 FullName 回填生效）
critical = all(("%s=True" % k) in joined for k in ("render_fail_absent", "fullname_surname_prefix_ok"))
passed = critical and ("EXCEPTION" not in joined)
out.append("RESULT: %s" % ("PASS" if passed else "FAIL"))
with open(OUT, "w", encoding="utf-8") as f:
    f.write("\n".join(out))
print("\n".join(out))
print("输出: %s" % OUT)
sys.exit(0 if passed else 1)
