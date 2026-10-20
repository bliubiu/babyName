/**
 * 起名接口契约 E2E（tester-army/e2e，docs/35）
 *
 * 运行前置：go -C backend run ./cmd/server（监听 127.0.0.1:8080）
 * 运行：pnpm exec e2e run tests/naming-api.e2e.ts
 *
 * 未配置模型：纯确定性断言，零模型调用。
 */
import { test, expect } from 'e2e';

// 本机一律 127.0.0.1，不用 localhost（IPv6 ::1 解析坑，docs/35 §6）
const BACKEND = 'http://127.0.0.1:8080';

test('起名接口契约：POST /api/v1/names/generate 返回候选名且性别回显', async () => {
  const response = await fetch(`${BACKEND}/api/v1/names/generate`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      surname: '李',
      gender: 'male',
      birth_year: 2024,
      birth_month: 1,
      birth_day: 15,
      birth_hour: 12,
    }),
  });

  expect(response.status).toBe(200);
  expect(response.headers.get('content-type')).toContain('application/json');

  const payload = (await response.json()) as {
    success?: boolean;
    data?: { names?: Array<{ full_name?: string; gender?: string }> };
  };

  const names = payload.data?.names ?? [];
  expect(names.length).toBeGreaterThan(0);

  // 每个候选名：以请求姓氏开头，性别回显与请求一致（回归：Gender 未映射曾全显「女」）
  for (const name of names) {
    expect(name.full_name).toMatch(/^李/);
    expect(name.gender).toBe('male');
  }
});
