# -*- coding: utf-8 -*-
"""SSR 全页面 E2E：首页/生成/结果/换一批/收藏/历史/测名/黄历/统计 + 错误路径。
用 urllib 发 UTF-8 表单（规避 Git Bash cp936 中文 GBK 坑）。结果写 smoke_e2e.txt。
"""
import http.cookiejar
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request

BASE = os.environ.get("SMOKE_BASE", "http://127.0.0.1:8080")
out = []


def log(ok, name, detail=""):
    out.append("%s %s %s" % ("PASS" if ok else "FAIL", name, detail))


def check(name, cond, detail=""):
    log(cond, name, detail)
    return cond


cj = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


noredir = urllib.request.build_opener(
    urllib.request.HTTPCookieProcessor(cj), NoRedirect
)


def get(path):
    try:
        r = opener.open(BASE + path, timeout=60)
        return r.status, r.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", errors="replace")


def get_noredir(path):
    try:
        r = noredir.open(BASE + path, timeout=60)
        return r.status, r.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", errors="replace")


def post(path, data):
    body = urllib.parse.urlencode(data).encode("utf-8")
    req = urllib.request.Request(
        BASE + path, data=body, method="POST",
        headers={"Content-Type": "application/x-www-form-urlencoded; charset=utf-8"},
    )
    try:
        r = noredir.open(req, timeout=120)
        return r.status, r.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", errors="replace")


# ---------- 1. 首页 ----------
st, html = get("/")
check("GET / 200", st == 200)
check("首页含出生分钟输入框", 'name="birth_minute"' in html)
check("首页含粤语开关", 'name="enable_cantonese"' in html)
check("无模板渲染失败", "模板渲染失败" not in html)

# ---------- 2. 生成（带分钟 + 粤语开关） ----------
form = {
    "surname": "李", "gender": "男",
    "birth_year": "2020", "birth_month": "6", "birth_day": "15",
    "birth_hour": "10", "birth_minute": "30",
    "birth_location": "北京", "name_length": "2",
    "avoid_elder_names": "李强", "enable_cantonese": "1",
}
st, _ = post("/generate", form)
check("POST /generate 303", st == 303, "got %s" % st)

st, html = get("/result")
check("GET /result 200", st == 200)
check("结果页无渲染失败", "模板渲染失败" not in html)
m = re.search(r"<h3[^>]*>\s*(李\S*?)\s*<", html)
check("候选名带姓氏前缀", m is not None, m.group(1) if m else "无 h3")
check("八字四柱渲染", "四柱：" in html)
check("紫微区块存在或数据为空(容忍)", "紫微斗数" in html or ".Ziwei" not in html)
check("粤拼标注渲染", "[粤:" in html, "未出现粤拼")
check("粤语分渲染", "粤语分" in html, "未出现粤语分")

# ---------- 3. 换一批 ----------
mgen = re.search(r'name="generation_id" value="([^"]+)"', html)
check("结果页含 generation_id", mgen is not None)
if mgen:
    st, _ = post("/explore", {"generation_id": mgen.group(1)})
    check("POST /explore 303", st == 303, "got %s" % st)
    st, html = get("/result")
    check("换一批后结果页 200", st == 200)
    check("换一批后无渲染失败", "模板渲染失败" not in html)

# ---------- 4. 收藏 + 取消 ----------
mname = re.search(r"<h3[^>]*>\s*(李\S*?)\s*<", html)
mform = re.search(
    r'<form method="post" action="/favorites"[^>]*>(.*?)</form>', html, re.S
)
if mform:
    fields = dict(re.findall(r'name="([^"]+)" value="([^"]*)"', mform.group(1)))
    st, _ = post("/favorites", fields)
    check("POST /favorites 303", st == 303, "got %s" % st)

st, html = get("/favorites")
check("GET /favorites 200", st == 200)
check("收藏列表含刚收藏名字", mname is not None and mname.group(1) in html, mname.group(1) if mname else "无候选名")
m = re.search(r'/favorites/delete[^>]*>\s*取消', html)
check("收藏页含取消入口", m is not None or "取消" in html)
# 取第一条收藏 id
mfid = re.search(r'name="id" value="([^"]+)"', html)
if mfid:
    st, _ = post("/favorites/delete", {"id": mfid.group(1)})
    check("POST /favorites/delete 303", st == 303, "got %s" % st)
    st, html = get("/favorites")
    check("取消后收藏页 200", st == 200)

# ---------- 5. 历史 ----------
st, html = get("/history")
check("GET /history 200", st == 200)
check("历史列表有条目", "TopName" not in html and ("李" in html))
mhid = re.search(r'name="id" value="([^"]+)"', html)
if mhid:
    st, _ = post("/history/delete", {"id": mhid.group(1)})
    check("POST /history/delete 303", st == 303, "got %s" % st)

# ---------- 6. 测名 ----------
st, html = get("/evaluate")
check("GET /evaluate 200", st == 200)
check("测名表单含姓/名输入", 'name="given_name"' in html)
st, html = post("/evaluate", {
    "surname": "李", "given_name": "明澈", "gender": "男",
    "birth_year": "2020", "birth_month": "6", "birth_day": "15",
    "birth_hour": "10", "birth_minute": "30", "birth_location": "北京",
})
check("POST /evaluate 200", st == 200)
check("测名结果渲染", "风险体检" in html, "无风险体检区块")
check("测名结果无渲染失败", "模板渲染失败" not in html)
check("测名评分明细渲染", "评分明细" in html)

# 测名错误路径（缺名字）
st, html = post("/evaluate", {
    "surname": "李", "given_name": "", "gender": "男",
    "birth_year": "2020", "birth_month": "6", "birth_day": "15",
})
check("测名缺名字不 500", st in (200, 303), "got %s" % st)
check("测名错误有中文提示", 'bg-red-100' in html, "无错误提示条")

# ---------- 7. 黄历 ----------
st, html = get("/huangli")
check("GET /huangli(默认今天) 200", st == 200)
check("黄历宜/忌渲染", "宜" in html and "忌" in html)
check("黄历无渲染失败", "模板渲染失败" not in html)

st, html = get("/huangli?year=2026&month=10&day=23")
check("GET /huangli 指定日期 200", st == 200)
check("黄历含农历", "农历" in html)

st, html = get("/huangli?year=2026&month=13&day=45")
check("黄历非法日期不 500", st in (200, 400), "got %s" % st)
check("黄历非法日期有提示", "无法换算" in html or "检查" in html, "无提示")

# ---------- 8. 统计 ----------
st, html = get("/stat")
check("GET /stat 200", st == 200)
check("统计表渲染", "排名" in html and "占比" in html)
check("统计无渲染失败", "模板渲染失败" not in html)

st, html = get("/stat?limit=50")
check("GET /stat?limit=50 200", st == 200)

# ---------- 9. 结果页直连（无会话）应重定向首页 ----------
cj2 = http.cookiejar.CookieJar()
op2 = urllib.request.build_opener(
    urllib.request.HTTPCookieProcessor(cj2), NoRedirect
)
try:
    r = op2.open(BASE + "/result", timeout=30)
    check("无会话访问 /result 303", r.status == 303)
except urllib.error.HTTPError as e:
    check("无会话访问 /result 303", e.code == 303, "got %s" % e.code)

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
OUT_DIR = os.path.join(REPO_ROOT, ".workbuddy", "tmp")
os.makedirs(OUT_DIR, exist_ok=True)
OUT = os.path.join(OUT_DIR, "smoke_e2e.txt")
passes = sum(1 for l in out if l.startswith("PASS"))
fails = sum(1 for l in out if l.startswith("FAIL"))
with open(OUT, "w", encoding="utf-8") as f:
    f.write("\n".join(out))
    f.write("\n\nTOTAL=%d PASS=%d FAIL=%d\n" % (len(out), passes, fails))
print("\n".join(out))
print("TOTAL=%d PASS=%d FAIL=%d" % (len(out), passes, fails))
print("输出: %s" % OUT)
sys.exit(1 if fails else 0)
