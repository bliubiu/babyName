#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""webUI 业务功能端到端验证脚本
以 webUI 实际调用的 HTTP 接口验证核心业务能力：
  1. /api/v1/names/generate        起名输出 TopN 推荐名字
  2. /api/v1/favorites POST/GET/DELETE  收藏 增/查/删 全链路
  3. /api/v1/names/evaluate         测名（与生成同源同分）
不依赖第三方库，使用标准库 urllib；直连 127.0.0.1 绕过代理。
"""
import json
import os
import sys
import urllib.request
import urllib.error

# 直连本地，绕过代理
os.environ["no_proxy"] = "127.0.0.1,localhost"
os.environ["NO_PROXY"] = "127.0.0.1,localhost"

BASE = os.environ.get("NAMER_BASE", "http://127.0.0.1:8080")
results = []  # (名称, 是否通过, 详情)


def call(method, path, body=None, raw_response=False):
    url = BASE + path
    data = None
    headers = {"Content-Type": "application/json"}
    if body is not None:
        data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            raw = resp.read().decode("utf-8")
            code = resp.getcode()
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", "replace")
        code = e.code
    except Exception as e:  # noqa: BLE001
        return None, -1, str(e)
    if raw_response:
        return raw, code, raw
    try:
        return json.loads(raw), code, raw
    except Exception:  # noqa: BLE001
        return raw, code, raw


def rec(name, ok, detail):
    results.append((name, ok, detail))
    mark = "✓ PASS" if ok else "✗ FAIL"
    print(f"[{mark}] {name}: {detail}")
    return ok


def main():
    print("=" * 96)
    print(f"webUI 业务功能端到端验证  ->  {BASE}")
    print("=" * 96)

    # ---------- 1. 起名 generate TopN ----------
    gen_body = {
        "surname": "王", "gender": "male",
        "birth_year": 2024, "birth_month": 5, "birth_day": 20,
        "birth_hour": 10, "birth_minute": 0,
    }
    gen, code, raw = call("POST", "/api/v1/names/generate", gen_body)
    ok = False
    detail = ""
    top_name = None
    if code != 200:
        detail = f"HTTP={code}"
    elif not isinstance(gen, dict):
        detail = "响应非 JSON 对象"
    else:
        data = gen.get("data", gen)
        names = data.get("names", []) if isinstance(data, dict) else []
        if not names:
            detail = "未返回任何名字"
        else:
            ok = True
            scored = [n for n in names if isinstance(n, dict) and "total_score" in n]
            bad = [n for n in scored if not (0 <= float(n["total_score"]) <= 100)]
            top = names[0]
            top_name = {
                "surname": gen_body["surname"],
                "given_name": top.get("given_name", ""),
                "full_name": top.get("full_name", ""),
                "score": float(top.get("total_score", 0)),
            }
            detail = (f"HTTP=200 返回 {len(names)} 个名字; "
                      f"Top1={top.get('full_name','')} 评分={top.get('total_score')}; "
                      f"评分越界数={len(bad)}")
            print("    Top10: " + " ".join(
                f"{n.get('full_name','')}({n.get('total_score')})"
                for n in names[:10] if isinstance(n, dict)))
    rec("起名输出TopN推荐名字", ok, detail)

    # ---------- 2. 收藏 create ----------
    fav_ok = False
    fav_detail = ""
    fav_id = None
    if top_name:
        fav_body = {
            "surname": top_name["surname"],
            "given_name": top_name["given_name"],
            "pinyin": "",
            "gender": "male",
            "score": top_name["score"],
            "source": "generate",
            "notes": "e2e自动验证",
        }
        fr, fcode, _ = call("POST", "/api/v1/favorites", fav_body)
        if fcode != 200:
            fav_detail = f"创建收藏 HTTP={fcode}"
        elif isinstance(fr, dict):
            fav_id = fr.get("id") or (fr.get("data", {}) or {}).get("id")
            if fav_id:
                fav_ok = True
                fav_detail = f"HTTP=200 返回 id={fav_id}"
            else:
                fav_detail = f"HTTP=200 但响应缺少 id: {json.dumps(fr, ensure_ascii=False)[:120]}"
        else:
            fav_detail = f"响应非预期: {str(fr)[:120]}"
    else:
        fav_detail = "跳过（无可用名字）"
    rec("收藏-创建(POST /favorites)", fav_ok, fav_detail)

    # ---------- 3. 收藏 list ----------
    list_ok = False
    list_detail = ""
    if fav_id:
        ls, lcode, _ = call("GET", "/api/v1/favorites")
        if lcode != 200:
            list_detail = f"查询收藏 HTTP={lcode}"
        else:
            # 响应信封: {code, success, data:{data:[...], records:[...], total, page, limit}}
            ls_inner = ls.get("data") if isinstance(ls.get("data"), dict) else ls
            recs = []
            if isinstance(ls_inner, dict):
                recs = (ls_inner.get("data") or ls_inner.get("records") or [])
            list_detail = f"信封解析后列表条数={len(recs)}"
            found = any((r.get("id") == fav_id) for r in recs if isinstance(r, dict))
            if found:
                list_ok = True
                list_detail = f"HTTP=200 列表含 {len(recs)} 条，命中本次收藏 id={fav_id}"
            else:
                list_detail = f"HTTP=200 列表 {len(recs)} 条但未包含 id={fav_id}"
    else:
        list_detail = "跳过（创建失败）"
    rec("收藏-查询(GET /favorites)", list_ok, list_detail)

    # ---------- 4. 收藏 delete ----------
    del_ok = False
    del_detail = ""
    if fav_id:
        dr, dcode, _ = call("DELETE", f"/api/v1/favorites/{fav_id}")
        if dcode != 200:
            del_detail = f"删除收藏 HTTP={dcode}"
        else:
            # 再次查询确认已删除
            ls2, _, _ = call("GET", "/api/v1/favorites")
            ls2_inner = ls2.get("data") if isinstance(ls2.get("data"), dict) else ls2
            recs2 = []
            if isinstance(ls2_inner, dict):
                recs2 = ls2_inner.get("data") or ls2_inner.get("records") or []
            gone = not any((r.get("id") == fav_id) for r in recs2 if isinstance(r, dict))
            if gone:
                del_ok = True
                del_detail = f"HTTP=200 删除成功，复查列表已无 id={fav_id}"
            else:
                del_detail = f"HTTP=200 但复查列表仍含 id={fav_id}"
    else:
        del_detail = "跳过（创建失败）"
    rec("收藏-删除(DELETE /favorites/:id)", del_ok, del_detail)

    # ---------- 5. 测名 evaluate 同源同分 ----------
    ev_ok = False
    ev_detail = ""
    if top_name:
        ev_body = {
            "surname": top_name["surname"],
            "given_name": top_name["given_name"],
            "gender": "male",
            "birth_year": 2024, "birth_month": 5, "birth_day": 20,
            "birth_hour": 10, "birth_minute": 0,
        }
        ev, ecode, _ = call("POST", "/api/v1/names/evaluate", ev_body)
        if ecode != 200:
            ev_detail = f"测名 HTTP={ecode}"
        elif isinstance(ev, dict):
            ev_data = ev.get("data", ev)
            name_obj = ev_data.get("name") if isinstance(ev_data, dict) else None
            ev_score = None
            if isinstance(name_obj, dict):
                ev_score = name_obj.get("total_score")
            elif isinstance(ev_data, dict):
                ev_score = ev_data.get("total_score")
            if ev_score is None:
                ev_detail = "测名响应缺少 total_score"
            elif abs(float(ev_score) - top_name["score"]) > 0.01:
                ev_detail = (f"测名分 {ev_score} 与生成分 {top_name['score']} 不一致")
            else:
                ev_ok = True
                ev_detail = f"测名分 {ev_score} == 生成分 {top_name['score']}（同源同分）"
        else:
            ev_detail = f"响应非预期: {str(ev)[:120]}"
    else:
        ev_detail = "跳过（无可用名字）"
    rec("测名同源同分(evaluate)", ev_ok, ev_detail)

    # ---------- 汇总 ----------
    print("=" * 96)
    passed = sum(1 for _, ok, _ in results if ok)
    total = len(results)
    print(f"汇总: {passed}/{total} 项通过")
    allok = passed == total
    print("最终结果:", "全部通过 ✓" if allok else "存在失败项 ✗")
    print("=" * 96)
    sys.exit(0 if allok else 1)


if __name__ == "__main__":
    main()
