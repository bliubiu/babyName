/**
 * 起名 UI 主链路 E2E（tester-army/e2e，docs/35）
 *
 * 运行前置：
 * 1. 后端：go -C backend run ./cmd/server   （监听 127.0.0.1:8080）
 * 2. 前端：cd frontend && pnpm dev          （监听 127.0.0.1:3000）
 * 运行：pnpm exec e2e run tests/naming-ui.e2e.ts
 *
 * 未配置模型：纯确定性断言（locator + expect），零模型调用。
 */
import { test } from '@e2e-dev/web';
import { expect } from 'e2e';

test('起名主链路：首页填表 → 生成 → 结果页出现候选名', async ({ app, screen }) => {
  await app.open('/');

  // 首页标题（HomeContent h1）
  await expect(screen.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  // 填姓氏（NameForm 姓氏输入框）
  const surname = screen.getByPlaceholder('请输入姓氏');
  await expect(surname).toBeVisible();
  await surname.fill('李');

  // 生辰使用表单默认值，直接提交
  await screen.getByRole('button', { name: '开始取名' }).click();

  // 生成成功后跳转结果页（生成秒级，用例 timeout 180s）。
  // e2e 的 App 无 waitForURL；改为等结果页真实内容（"李"开头候选名文本）出现。
  // 说明：getByText 只匹配文本节点、不匹配 input 值，首页不会命中，能准确等到结果页。
  const firstCandidate = screen.getByText(/^李/).first();
  await expect(firstCandidate).toBeVisible({ timeout: 90000 });
});
