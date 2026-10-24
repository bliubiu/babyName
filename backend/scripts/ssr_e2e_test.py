#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""SSR 全流程端到端验证脚本

以真实浏览器动线验证 SSR 层（零 JS、表单提交 + cookie 会话）：
  首页 → 生成(PR G) → 结果页 → 换一批 → 收藏 → 收藏列表 → 删除
  → 历史 → 测名 → 黄历 → 统计，外加表单校验负面用例。

依赖：仅标准库（urllib + cookiejar）；通过环境变量 NAMER_BASE 指定服务地址。
"""
import html as html_mod
import json
import os
import re
import sys
import urllib.parse
import urllib.request
import http.cookiejar

os.environ.setdefault("no_proxy", "127.0.0.1,localhost")
os.environ.setdefault("NO_PROXY", "127.0.0.1,localhost")

BASE = os.environ.get("NAMER_BASE", "http://127.0.0.1:8090")
results = []

# 全局共享同一个 cookie jar：call_no_redirect 每次新建一次性 jar 会让
# /generate 发下的 nm_sid 随 jar 丢弃，结果页将永远收不到会话（历史缺陷）。
JAR = http.cookiejar.CookieJar()
_opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(JAR))


def call(method, path, form=None):
    """SSR 表单/页面请求；返回 (状态码, 响应体, 响应对象)。跟随重定向。"""
    url = BASE + path
    data = None
    headers = {}
    if form is not None:
        data = urllib.parse.urlencode(form).encode("utf-8")
        headers["Content-Type"] = "application/x-www-form-urlencoded"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with _opener.open(req, timeout=180) as resp:
            return resp.getcode(), resp.read().decode("utf-8", "replace"), resp
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", "replace"), e
    except Exception as e:  # noqa: BLE001
        return -1, str(e), None


def call_no_redirect(method, path, form=None):
    """不跟随重定向，返回 (状态码, Location, Set-Cookie)。"""
    url = BASE + path
    data = None
    headers = {}
    if form is not None:
        data = urllib.parse.urlencode(form).encode("utf-8")
        headers["Content-Type"] = "application/x-www-form-urlencoded"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)

    class NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self, *a, **k):  # noqa: ANN002, ANN003
            return None

    try:
        op = urllib.request.build_opener(NoRedirect,
                                         urllib.request.HTTPCookieProcessor(JAR))
        resp = op.open(req, timeout=180)
        return resp.getcode(), resp.headers.get("Location", ""), resp.headers.get("Set-Cookie", "")
    except urllib.error.HTTPError as e:
        return e.code, e.headers.get("Location", ""), e.headers.get("Set-Cookie", "")


def rec(name, ok, detail):
    results.append((name, ok, detail))
    mark = "✓ PASS" if ok else "✗ FAIL"
    print(f"[{mark}] {name}: {detail}")


H3_RE = re.compile(r'<h3 class="name-big">\s*([^<]+?)\s*<')  # SSR 结果卡标题标记
GENID_RE = re.compile(r'name="generation_id" value="([^"]+)"')


def extract_names(page):
    return [html_mod.unescape(m.strip()) for m in H3_RE.findall(page)]


def main():
    print("=" * 96)
    print(f"SSR 全流程端到端验证  ->  {BASE}")
    print("=" * 96)

    # ---------- 1. 首页 ----------
    code, page, _ = call("GET", "/")
    rec("首页渲染", code == 200 and "起名" in page, f"HTTP={code}")

    # ---------- 2. 生成（PRG + cookie 会话） ----------
    form = {
        "surname": "王", "gender": "男",
        "birth_year": "2024", "birth_month": "5", "birth_day": "20",
        "birth_hour": "10", "birth_minute": "0",
    }
    code, loc, set_cookie = call_no_redirect("POST", "/generate", form)
    ok = code == 303 and loc.startswith("/result") and "nm_sid" in set_cookie
    rec("生成 PRG(POST /generate → 303 /result + nm_sid)", ok,
        f"HTTP={code} Location={loc} Set-Cookie含nm_sid={'nm_sid' in set_cookie}")
    if not ok:
        finish()

    # ---------- 3. 结果页渲染 ----------
    code, page, _ = call("GET", "/result")
    names_a = extract_names(page)
    gen_id_m = GENID_RE.search(page)
    rec("结果页渲染 + 会话名字", code == 200 and bool(names_a),
        f"HTTP={code} 名字数={len(names_a)} 抽样={names_a[:3]}")

    # ---------- 4. 换一批 ----------
    if not gen_id_m:
        rec("换一批(POST /explore)", False, "结果页缺少 generation_id")
        finish()
    code, loc, _ = call_no_redirect("POST", "/explore",
                                    {"generation_id": gen_id_m.group(1)})
    code2, page2, _ = call("GET", "/result")
    names_b = extract_names(page2)
    ok = code == 303 and code2 == 200 and bool(names_b) and names_b != names_a
    rec("换一批(POST /explore → 刷新结果页)", ok,
        f"HTTP={code} 新名字数={len(names_b)} 与旧批相同={names_b == names_a}")

    # ---------- 5. 收藏（服务层 score 防线正例，≥80 触发自学习） ----------
    full = names_b[0]
    given = full[len("王"):] if full.startswith("王") else full
    form = {"surname": "王", "given_name": given, "pinyin": "",
            "gender": "male", "score": "85", "source": "e2e"}
    code, loc, _ = call_no_redirect("POST", "/favorites", form)
    ok = code == 303 and loc.startswith("/result")
    rec("收藏(POST /favorites score=85)", ok, f"HTTP={code} Location={loc} 名={full}")

    # ---------- 6. 收藏列表（SSR 页） ----------
    code, fav_page, _ = call("GET", "/favorites")
    in_page = given in fav_page
    rec("收藏列表(GET /favorites 含新收藏)", code == 200 and in_page,
        f"HTTP={code} 页面含{given}={in_page}")

    # ---------- 7. 拿收藏 id（JSON API 与 SSR 共库） ----------
    code, raw, _ = call("GET", "/api/v1/favorites")
    fav_id = None
    try:
        env = json.loads(raw)
        data = env.get("data") if isinstance(env.get("data"), dict) else env
        recs = (data.get("data") or data.get("records") or []) if isinstance(data, dict) else []
        for r in recs:
            if isinstance(r, dict) and r.get("given_name") == given:
                fav_id = r.get("id")
                break
    except Exception:  # noqa: BLE001
        pass
    rec("收藏 id 解析(GET /api/v1/favorites)", bool(fav_id),
        f"HTTP={code} id={fav_id}")

    # ---------- 8. 删除收藏（SSR 表单口径） ----------
    if fav_id:
        code, loc, _ = call_no_redirect("POST", "/favorites/delete", {"id": fav_id, "page": "1"})
        code2, raw2, _ = call("GET", "/api/v1/favorites")
        gone = fav_id not in raw2
        rec("删除收藏(POST /favorites/delete → 复查已删)", code == 303 and gone,
            f"HTTP={code} 复查已删={gone}")
    else:
        rec("删除收藏(POST /favorites/delete)", False, "无 id，跳过执行")

    # ---------- 9. 历史页 ----------
    code, page, _ = call("GET", "/history")
    rec("历史页(GET /history)", code == 200 and "王" in page, f"HTTP={code}")

    # ---------- 10. 测名（与生成同一条评分链） ----------
    form = {
        "surname": "王", "given_name": given or "涵", "gender": "女",
        "birth_year": "2024", "birth_month": "5", "birth_day": "20",
        "birth_hour": "10", "birth_minute": "0",
    }
    code, page, _ = call("POST", "/evaluate", form)
    has_score = bool(re.search(r"综合.*评分|total|评分", page))
    rec("测名(POST /evaluate 渲染评分)", code == 200 and has_score,
        f"HTTP={code} 含评分文案={has_score}")

    # ---------- 11. 黄历 / 统计页 ----------
    code, page, _ = call("GET", "/huangli")
    rec("黄历页(GET /huangli)", code == 200, f"HTTP={code}")
    code, page, _ = call("GET", "/stat")
    rec("统计页(GET /stat)", code == 200, f"HTTP={code}")

    # ---------- 12. 负面：收藏评分越界（服务防线回归） ----------
    form = {"surname": "王", "given_name": given or "涵", "score": "999"}
    code, loc, _ = call_no_redirect("POST", "/favorites", form)
    import urllib.parse as _up
    err_msg = html_mod.unescape(_up.parse_qs(_up.urlparse(loc).query).get("error", [""])[0]) if loc else ""
    ok = code == 303 and err_msg == "评分必须在0-100之间"
    rec("负面-收藏评分越界拦截", ok, f"HTTP={code} error={err_msg}")

    # ---------- 13. 负面：非法性别被表单校验拦截 ----------
    form = dict(form_ := {"surname": "王", "gender": "外星人",
                          "birth_year": "2024", "birth_month": "5", "birth_day": "20"})
    code, loc, _ = call_no_redirect("POST", "/generate", form)
    err_msg = html_mod.unescape(_up.parse_qs(_up.urlparse(loc).query).get("error", [""])[0]) if loc else ""
    ok = code == 303 and err_msg != ""
    rec("负面-生成非法性别拦截", ok, f"HTTP={code} error={err_msg}")

    finish()


def finish():
    print("=" * 96)
    passed = sum(1 for _, ok, _ in results if ok)
    total = len(results)
    print(f"汇总: {passed}/{total} 项通过")
    print("最终结果:", "全部通过 ✓" if passed == total else "存在失败项 ✗")
    print("=" * 96)
    sys.exit(0 if passed == total else 1)


if __name__ == "__main__":
    main()
