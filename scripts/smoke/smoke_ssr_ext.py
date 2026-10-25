# -*- coding: utf-8 -*-
"""SSR 新增能力 E2E（零 JS 差距补齐回归）：
结果页筛选/筛选态跨换一批保持/对比(空态+1名+2名+无会话)/JSON查看器/
暗色模式切换/完整报告/历史回看/统计按名查询/黄历万年历+时辰吉凶/Flash 横幅。
urllib + UTF-8 表单（规避 Git Bash cp936 中文 GBK 坑），结果写 smoke_ssr_ext.txt。
"""
import http.cookiejar
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request

BASE = os.environ.get("SMOKE_BASE", "http://127.0.0.1:8080")
REPO_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
OUT_DIR = os.path.join(REPO_ROOT, ".workbuddy", "tmp")
os.makedirs(OUT_DIR, exist_ok=True)
OUT = os.path.join(OUT_DIR, "smoke_ssr_ext.txt")
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


def fresh_noredir():
    j = http.cookiejar.CookieJar()
    return urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(j), NoRedirect
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
    """POST 不跟随重定向，返回 (状态码, 响应体, Location 头)。
    doseq=True：compare 等多值字段必须展开为重复键，否则列表会被 str() 成单值。"""
    body = urllib.parse.urlencode(data, doseq=True).encode("utf-8")
    req = urllib.request.Request(
        BASE + path, data=body, method="POST",
        headers={"Content-Type": "application/x-www-form-urlencoded; charset=utf-8"},
    )
    try:
        r = noredir.open(req, timeout=120)
        return r.status, r.read().decode("utf-8", errors="replace"), r.headers.get("Location", "")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", errors="replace"), e.headers.get("Location", "")


def no_render_fail(html):
    return "模板渲染失败" not in html


# ---------- 0. 建立会话：生成 ----------
form = {
    "surname": "林", "gender": "男",
    "birth_year": "2019", "birth_month": "3", "birth_day": "8",
    "birth_hour": "9", "birth_minute": "15",
    "birth_location": "杭州", "name_length": "2",
}
st, _, loc = post("/generate", form)
check("POST /generate 303", st == 303, "got %s" % st)
st, html = get("/result")
check("GET /result 200", st == 200)
check("结果页无渲染失败", no_render_fail(html))

# ---------- 1. 结果页筛选 ----------
fq = "gender=female&wuxing=%E9%87%91&min_strokes=8&max_strokes=20&min_tier=2"
st, html = get("/result?" + fq)
check("GET /result?筛选 200", st == 200)
check("筛选页无渲染失败", no_render_fail(html))
check("筛选后计数展示", "筛选后" in html)
check("性别选项保持选中", 'value="female" selected' in html)
check("五行选项保持选中", 'value="金" selected' in html)
check("含清除筛选入口", 'href="/result"' in html)

# ---------- 2. 筛选态跨换一批保持 ----------
mgen = re.search(r'name="generation_id" value="([^"]+)"', html)
check("结果页含 generation_id", mgen is not None)
if mgen:
    st, _, loc = post("/explore", {"generation_id": mgen.group(1), "rq": urllib.parse.unquote(fq)})
    check("POST /explore(带 rq) 303", st == 303, "got %s" % st)
    check("重定向保留筛选串(ASCII 部分)", "gender=female" in loc and "min_strokes=8" in loc, loc)
    st, html = get("/result?" + fq)
    check("换一批后筛选仍生效", "筛选后" in html and no_render_fail(html))

# ---------- 3. 对比 ----------
st, html = get("/compare")
check("GET /compare 空态引导 200", st == 200)
check("空态含勾选引导文案", "请先选择至少 2 个名字" in html)

# checkbox 要从结果页抓（对比页是空态，没有勾选框）
st, rhtml = get("/result")
mnames = re.findall(r'name="compare" value="([^"]+)" form="cmpForm"', rhtml)
check("结果页含对比 checkbox", len(mnames) >= 2, "勾选 %d 个" % len(mnames))

if len(mnames) >= 1:
    st, _, loc = post("/compare", {"compare": mnames[0]})
    check("POST /compare 1 名 → 303 拦截", st == 303, "got %s" % st)
    check("拦截文案=请先勾选 2-4 个", "请先勾选" in urllib.parse.unquote_plus(loc), loc)

if len(mnames) >= 2:
    st, html2, _ = post("/compare", {"compare": [mnames[0], mnames[1]]})
    check("POST /compare 2 名 → 200", st == 200, "got %s" % st)
    check("对比表含综合评分", "综合评分" in html2)
    check("对比表含综合分析", "综合分析" in html2)
    check("对比表含音韵律动/三才/生肖", all(k in html2 for k in ("音韵律动", "天地人三才", "生肖适配")))
    check("对比页渲染两个名字", mnames[0] in html2 and mnames[1] in html2)

# 无会话对比（全新 cookie；列表字段必须 doseq=True，否则 urlencode 会把列表 str() 成单值）
op = fresh_noredir()
body = urllib.parse.urlencode({"compare": ["甲", "乙"]}, doseq=True).encode("utf-8")
req = urllib.request.Request(BASE + "/compare", data=body, method="POST",
                             headers={"Content-Type": "application/x-www-form-urlencoded; charset=utf-8"})
try:
    r = op.open(req, timeout=30)
    code, loc = r.status, r.headers.get("Location", "")
except urllib.error.HTTPError as e:
    code, loc = e.code, e.headers.get("Location", "")
check("无会话 POST /compare → 303", code == 303, "got %s" % code)
check("无会话引导回首页", "请先生成名字" in urllib.parse.unquote_plus(loc), loc)

# ---------- 4. JSON 查看器 ----------
st, html = get("/json-viewer")
check("GET /json-viewer 200", st == 200)
check("iframe 指向内嵌查看器", 'src="/json4matter.html"' in html)
check("查看器页无渲染失败", no_render_fail(html))
try:
    r = opener.open(BASE + "/json4matter.html", timeout=20)
    ctype = r.headers.get("Content-Type", "")
    asset = r.read().decode("utf-8", errors="replace")
    code = r.status
except urllib.error.HTTPError as e:
    code, ctype, asset = e.code, e.headers.get("Content-Type", ""), ""
check("GET /json4matter.html 200", code == 200, "got %s" % code)
check("静态资源 Content-Type text/html", "text/html" in ctype, ctype)
check("查看器为可交互单文件(含脚本)", "<script" in asset and "json4matter" in asset.lower() or "JSON" in asset)

# ---------- 5. 暗色模式 ----------
st, _, loc = post("/theme", {"theme": "dark", "redirect": "/stat"})
check("POST /theme(dark) 303", st == 303, "got %s" % st)
check("主题切换回跳 redirect", loc == "/stat", loc)
st, html = get("/stat")
check("暗色 cookie 生效(html.dark)", '<html lang="zh-CN" class="dark">' in html)
check("暗色覆盖样式存在", ".dark{" in html)
st, _, loc = post("/theme", {"theme": "light", "redirect": "/"})
check("POST /theme(light) 303", st == 303, "got %s" % st)
st, html = get("/")
check("浅色恢复(无 dark class)", '<html lang="zh-CN" class="dark">' not in html)

# ---------- 6. 完整报告 ----------
st, html = get("/report")
check("GET /report(会话) 200", st == 200)
check("报告含八字分析段", "一、八字分析" in html)
check("报告含候选名详情段", "二、候选名详情" in html)
check("报告含综合解读段", "三、综合解读" in html)
check("解读五段齐全", all(k in html for k in ("八字格局", "五行平衡", "喜用神补益", "生肖纳音", "名字质量")))
check("报告无渲染失败", no_render_fail(html))
op = fresh_noredir()
try:
    r = op.open(BASE + "/report", timeout=30)
    code, loc = r.status, r.headers.get("Location", "")
except urllib.error.HTTPError as e:
    code, loc = e.code, e.headers.get("Location", "")
check("无会话 GET /report → 303", code == 303, "got %s" % code)
check("无会话报告引导回首页", "请先生成名字" in urllib.parse.unquote_plus(loc), loc)

# ---------- 7. 历史回看 ----------
st, html = get("/history")
check("GET /history 200", st == 200)
mview = re.search(r'href="(/history/view\?id=[^"]+)"', html)
check("历史列表含回看入口", mview is not None)
if mview:
    st, html = get(mview.group(1))
    check("GET /history/view 200", st == 200)
    check("回看横幅提示", "回看历史记录" in html)
    check("回看页隐去换一批表单", 'action="/explore"' not in html)
    check("回看页隐去对比勾选", "cmpForm" not in html)
    check("回看页无渲染失败", no_render_fail(html))
# GET 非法 id（不跟随重定向）
try:
    r = noredir.open(BASE + "/history/view?id=__not_exist__", timeout=30)
    code, loc = r.status, r.headers.get("Location", "")
except urllib.error.HTTPError as e:
    code, loc = e.code, e.headers.get("Location", "")
check("非法 id 回看 → 303 不 500", code == 303, "got %s" % code)
check("非法 id 回 history?error", "/history?error=" in loc, loc)
try:
    r = noredir.open(BASE + "/history/view", timeout=30)
    code, loc = r.status, r.headers.get("Location", "")
except urllib.error.HTTPError as e:
    code, loc = e.code, e.headers.get("Location", "")
check("缺 id 回看 → 303", code == 303, "got %s" % code)
check("缺 id 回 /history", loc == "/history", loc)

# ---------- 8. 统计按名查询 ----------
st, html = get("/stat?name=%E6%9E%97")  # 林
check("GET /stat?name=林 200", st == 200)
check("命中展示重名率", "重名率" in html and "重名度" in html)
check("无渲染失败", no_render_fail(html))
st, html = get("/stat?name=%E9%BE%8D%E7%BE%BD")  # 龙羽（大概率未收录）
check("未收录名不 500", st == 200, "got %s" % st)
check("未收录有提示", "语料库中未收录" in html, "无未收录提示")

# ---------- 9. 黄历 ----------
st, html = get("/huangli?view=month&year=2026&month=10")
check("万年历月视图 200", st == 200)
check("万年历文案", "万年历" in html)
check("月视图无渲染失败", no_render_fail(html))
# html/template 会把 href 里的 & 转义成 &amp;
norm = html.replace("&amp;", "&")
check("月视图含上月入口", "view=month&year=2026&month=9" in norm)
check("月视图含下月入口", "view=month&year=2026&month=11" in norm)
st, html = get("/huangli?view=month&year=2026&month=13")
check("非法月份不 500", st == 200, "got %s" % st)
check("非法月份有提示", "月份需在 1-12 之间" in html)
st, html = get("/huangli?year=2026&month=10&day=23")
check("日视图 200", st == 200)
check("时辰吉凶格", "时辰吉凶" in html)
check("时辰含时间区间", bool(re.search(r"\d{2}:00 ~ \d{2}:59", html)))
check("时辰含吉凶标记", "吉" in html and "凶" in html)
check("日视图无渲染失败", no_render_fail(html))

# ---------- 10. Flash 横幅 ----------
st, html = get("/?error=" + urllib.parse.quote("冒烟测试错误"))
check("GET /?error= 200", st == 200)
check("错误横幅渲染", "bg-red-100" in html and "冒烟测试错误" in html)
# Ok 横幅：收藏成功路径带 ok 参数
st, html = get("/result?ok=" + urllib.parse.quote("已加入收藏"))
check("GET /result?ok= 200", st == 200)
check("成功横幅渲染", "bg-emerald-100" in html and "已加入收藏" in html)

passes = sum(1 for l in out if l.startswith("PASS"))
fails = sum(1 for l in out if l.startswith("FAIL"))
with open(OUT, "w", encoding="utf-8") as f:
    f.write("\n".join(out))
    f.write("\n\nTOTAL=%d PASS=%d FAIL=%d\n" % (len(out), passes, fails))
print("\n".join(out))
print("TOTAL=%d PASS=%d FAIL=%d" % (len(out), passes, fails))
print("输出: %s" % OUT)
sys.exit(1 if fails else 0)
