# -*- coding: utf-8 -*-
"""删除链路冒烟：取消收藏 / 删除历史。"""
import http.cookiejar
import os
import re
import sys
import urllib.parse
import urllib.request

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
    st, fav = get("/favorites")
    ids = re.findall(r'name="id" value="([^"]+)"', fav)
    out.append("favorites_page=%d entries=%s" % (st, len(ids)))
    if ids:
        st, _ = post("/favorites/delete", {"id": ids[0], "page": "1"})
        st2, fav2 = get("/favorites")
        ids2 = re.findall(r'name="id" value="([^"]+)"', fav2)
        out.append("after delete: status=%d page=%d entries=%d removed=%s" % (
            st, st2, len(ids2), ids[0] not in ids2))

    st, hist = get("/history")
    hids = re.findall(r'name="id" value="([^"]+)"', hist)
    out.append("history_page=%d entries=%s" % (st, len(hids)))
    if hids:
        st, _ = post("/history/delete", {"id": hids[0], "page": "1"})
        st2, hist2 = get("/history")
        hids2 = re.findall(r'name="id" value="([^"]+)"', hist2)
        out.append("after delete: status=%d page=%d entries=%d removed=%s" % (
            st, st2, len(hids2), hids[0] not in hids2))
        out.append("history_render_ok=%s" % ("模板渲染失败" not in hist2))
except Exception as e:
    out.append("EXCEPTION: %r" % e)

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
OUT_DIR = os.path.join(REPO_ROOT, ".workbuddy", "tmp")
os.makedirs(OUT_DIR, exist_ok=True)
OUT = os.path.join(OUT_DIR, "smoke_delete.txt")
joined = "\n".join(out)
passed = "EXCEPTION" not in joined
out.append("RESULT: %s" % ("PASS" if passed else "FAIL"))
with open(OUT, "w", encoding="utf-8") as f:
    f.write("\n".join(out))
print("\n".join(out))
print("输出: %s" % OUT)
sys.exit(0 if passed else 1)
