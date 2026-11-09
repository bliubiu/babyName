# -*- coding: utf-8 -*-
"""全业务 API 冒烟测试（按实际注册路由，docs/44 E2E 验证产物）。

覆盖 9 大类 23 用例：健康检查、生成（同步/异步/任务轮询/探索换一批/测名）、
历史（增查删）、收藏（增查删）、统计、PDF 报告、黄历/农历/农历月历、
规范汉字分组、错误处理契约。写操作测试数据自动清理，不污染用户数据。

用法（服务已启动时）：
    python scripts/smoke/api_smoke.py            # 默认打 127.0.0.1:8080
    NAMER_API_BASE=http://127.0.0.1:9000 python scripts/smoke/api_smoke.py

注意：Git Bash 终端下 curl -d 中文会以 GBK 传输导致 400，请统一用本脚本
（urllib UTF-8）或等价客户端，勿用 curl 直传中文 body。
"""
import json
import os
import urllib.error
import urllib.parse
import urllib.request

BASE = os.environ.get('NAMER_API_BASE', 'http://127.0.0.1:8080')
results = []


def call(method, path, body=None):
    data = json.dumps(body).encode('utf-8') if body is not None else None
    req = urllib.request.Request(BASE + urllib.parse.quote(path, safe='/:?&='),
                                 data=data, method=method,
                                 headers={'Content-Type': 'application/json'})
    try:
        r = urllib.request.urlopen(req, timeout=60)
        raw = r.read()
        try:
            return r.status, json.loads(raw.decode('utf-8'))
        except Exception:
            return r.status, {'_binary': True, '_head': raw[:8], '_len': len(raw)}
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read().decode('utf-8'))
        except Exception:
            return e.code, {}


def rec(name, ok, detail=''):
    results.append((name, ok, detail))
    print(('PASS' if ok else 'FAIL'), name, ('| ' + detail if detail else ''))


def main():
    # 1. 健康检查
    s, d = call('GET', '/healthz')
    rec('1a健康检查', s == 200 and d.get('status') == 'ok', str(d.get('status')))
    s, d = call('GET', '/readyz')
    rec('1b就绪检查', s == 200 and d.get('status') == 'ready', str(d.get('status')))

    # 2. 名字生成（同步/异步/探索/评估）
    gen = {'surname': '端到', 'gender': 'female', 'birth_year': 2025,
           'birth_month': 3, 'birth_day': 21, 'birth_hour': 14,
           'birth_minute': 30, 'birth_location': '杭州'}
    s, d = call('POST', '/api/v1/names/generate', gen)
    names = d.get('data', {}).get('names', [])
    gid = d.get('data', {}).get('generation_id', '')
    rec('2a同步生成', s == 200 and len(names) > 0, 'HTTP=%d 名字数=%d' % (s, len(names)))
    s, d = call('POST', '/api/v1/names/generate/async', gen)
    tid = d.get('data', {}).get('task_id', '')
    rec('2b异步生成', s == 200 and tid, 'HTTP=%d task=%s' % (s, str(tid)[:20]))
    s, d = call('GET', '/api/v1/names/task/' + str(tid))
    rec('2c任务状态查询', s == 200 and d.get('data', {}).get('status') in ('success', 'running'),
        'HTTP=%d status=%s' % (s, d.get('data', {}).get('status')))
    s, d = call('POST', '/api/v1/names/generate/explore', {'generation_id': gid, 'count': 10})
    rec('2d探索换一批', s == 200 and len(d.get('data', {}).get('names', [])) > 0, 'HTTP=%d' % s)
    s, d = call('POST', '/api/v1/names/evaluate',
                {'surname': '端到', 'given_name': '测试', 'gender': 'female',
                 'birth_year': 2025, 'birth_month': 3, 'birth_day': 21, 'birth_hour': 14})
    rec('2e测名评估', s == 200 and bool(d.get('data')), 'HTTP=%d' % s)

    # 3. 历史记录（增查删）
    hist_body = {'surname': '端到', 'gender': 'female', 'birth_date': '2025-03-21',
                 'birth_time': '14:30', 'results': {'names': [{'full_name': '端到测试'}]}}
    s, d = call('POST', '/api/v1/history', hist_body)
    hid = d.get('data', {}).get('id', '')
    rec('3a保存历史', s == 200 and bool(hid), 'HTTP=%d id=%s' % (s, str(hid)[:20]))
    s, d = call('GET', '/api/v1/history')
    cnt = len(d.get('data', [])) if isinstance(d.get('data'), list) else len(d.get('data', {}).get('records', []))
    rec('3b查询历史', s == 200, 'HTTP=%d 条数=%s' % (s, cnt))
    s, d = call('DELETE', '/api/v1/history/' + str(hid))
    rec('3c删除历史', s == 200, 'HTTP=%d' % s)

    # 4. 收藏（增查删）
    fav_body = {'surname': '端到', 'given_name': '测试', 'pinyin': 'duan dao ce shi',
                'gender': 'female', 'score': 88, 'notes': '端到端测试数据'}
    s, d = call('POST', '/api/v1/favorites', fav_body)
    fid = d.get('data', {}).get('id', '')
    rec('4a添加收藏', s == 200 and bool(fid), 'HTTP=%d' % s)
    s, d = call('GET', '/api/v1/favorites')
    rec('4b收藏列表', s == 200, 'HTTP=%d' % s)
    s, d = call('DELETE', '/api/v1/favorites/' + str(fid))
    rec('4c删除收藏', s == 200, 'HTTP=%d' % s)

    # 5. 名字统计
    s, d = call('GET', '/api/v1/namestat/伟')
    rec('5a重名率查询', s == 200, 'HTTP=%d data=%s' % (s, str(d.get('data'))[:60]))
    s, d = call('GET', '/api/v1/namestat?limit=20')
    top = d.get('data', [])
    rec('5b常用名排行', s == 200 and len(top) > 0, 'HTTP=%d 条数=%d' % (s, len(top)))

    # 6. 报告生成
    s, d = call('POST', '/api/v1/report/pdf',
                {'data': {'surname': '端到', 'gender': 'female', 'birth_date': '2025-03-21',
                          'birth_time': '14:30', 'names': [{'full_name': '端到测试', 'score': 88}]}})
    rec('6PDF报告', s == 200, 'HTTP=%d data=%s' % (s, str(d.get('data'))[:60]))

    # 7. 黄历农历
    s, d = call('GET', '/api/v1/huangli?year=2026&month=11&day=8')
    rec('7a黄历', s == 200 and bool(d.get('data')), 'HTTP=%d' % s)
    s, d = call('GET', '/api/v1/lunar?year=2026&month=11&day=8&hour=10&minute=30')
    rec('7b农历', s == 200 and bool(d.get('data')), 'HTTP=%d' % s)
    s, d = call('GET', '/api/v1/lunar/month?year=2026&month=11')
    rec('7c农历月历', s == 200, 'HTTP=%d' % s)

    # 8. 规范汉字分组
    s, d = call('GET', '/api/v1/characters/groups')
    groups = d.get('data', [])
    rec('8规范汉字分组', s == 200 and len(groups) > 0, 'HTTP=%d 分组数=%d' % (s, len(groups)))

    # 9. 错误处理契约
    s, d = call('POST', '/api/v1/names/generate',
                {'surname': '', 'gender': 'male', 'birth_year': 2024,
                 'birth_month': 1, 'birth_day': 1, 'birth_hour': 1})
    rec('9a空姓氏400', s == 400, 'HTTP=%d msg=%s' % (s, str(d.get('message'))[:40]))
    s, d = call('GET', '/api/v1/names/task/nonexistent')
    rec('9b不存在任务', s in (400, 404), 'HTTP=%d' % s)
    s, d = call('POST', '/api/v1/names/evaluate', {'surname': '端到'})
    rec('9c缺参400', s == 400, 'HTTP=%d' % s)

    fails = [r for r in results if not r[1]]
    print('=' * 30)
    print('API 冒烟: PASS=%d FAIL=%d' % (len(results) - len(fails), len(fails)))
    raise SystemExit(1 if fails else 0)


if __name__ == '__main__':
    main()
