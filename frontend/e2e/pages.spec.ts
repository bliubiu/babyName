/**
 * 全业务页面 E2E（@playwright/test，docs/47 S5）
 *
 * 覆盖：导航可达、测名链路、黄历、重名率、历史页与 JSON 查看器。
 */
import { test, expect } from '@playwright/test';

test('导航栏可达性：首页 → 测名 → 黄历', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  // 点击导航「测名」进入评估页
  await page.getByLabel('测名').click();
  await expect(page.getByPlaceholder('如：王')).toBeVisible({ timeout: 15_000 });

  // 点击导航「黄历」进入黄历页
  await page.getByLabel('查看今日黄历').click();
  await expect(page.getByRole('heading', { name: '今日黄历' })).toBeVisible({ timeout: 15_000 });
});

test('测名链路：填姓名 → 开始测名 → 评分结果出现', async ({ page }) => {
  await page.goto('/#/evaluate');

  await page.getByPlaceholder('如：王').fill('王');
  await page.getByPlaceholder('如：浩然').fill('浩然');
  await page.getByRole('button', { name: '开始测名' }).click();

  // 评分出现：名帖印章（score-seal-lg）章内「综合评分」小字 + 数字——
  // IAB playwright 表面无 page.getByLabelText，以章内文本为锚
  await expect(page.getByText('综合评分').first()).toBeVisible({ timeout: 60_000 });
  await expect(page.getByText('八维评分依据')).toBeVisible({ timeout: 60_000 });
});

test('黄历页：今日黄历标题与宜忌内容渲染', async ({ page }) => {
  await page.goto('/#/huangli');
  await expect(page.getByRole('heading', { name: '今日黄历' })).toBeVisible();
  await expect(page.getByText('宜', { exact: true }).first()).toBeVisible({ timeout: 15_000 });
  await expect(page.getByText('忌', { exact: true }).first()).toBeVisible({ timeout: 15_000 });
});

test('万年历节气科普：点击节气日弹出知识卡，可关闭', async ({ page }) => {
  await page.goto('/#/huangli');
  await expect(page.getByRole('heading', { name: '今日黄历' })).toBeVisible();

  await page.getByRole('button', { name: '万年历' }).click();
  // 选 1 月：每年 1 月必含小寒、大寒两个节气，避免依赖当前月份
  await page.locator('select').nth(1).selectOption('1');

  const termCell = page.locator('[aria-label*="查看节气科普"]').first();
  await expect(termCell).toBeVisible({ timeout: 30_000 });
  await termCell.click();

  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  await expect(dialog.getByText('节气三候')).toBeVisible();
  await expect(dialog.getByText('节气故事')).toBeVisible();

  await dialog.getByRole('button', { name: '关闭节气科普' }).click();
  await expect(dialog).toHaveCount(0);
});

test('重名率查询：填名字 → 查询 → 重名人数与热门名字榜出现', async ({ page }) => {
  await page.goto('/#/stat');
  await expect(page.getByRole('heading', { name: '重名率查询' })).toBeVisible();

  await page.getByLabel('输入名字查询重名率').fill('梓涵');
  await page.getByRole('button', { name: '查询' }).click();

  await expect(page.getByText('重名人数').first()).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText('热门名字榜')).toBeVisible({ timeout: 30_000 });
});

test('历史页与 JSON 查看器页面可达', async ({ page }) => {
  await page.goto('/#/history');
  await expect(page.getByRole('heading', { name: '历史记录' })).toBeVisible();

  await page.goto('/#/json-viewer');
  // JSON 查看器以 iframe 承载 public/json4matter.html（iframe 本体不进 ARIA 树，
  // 以主导航渲染确认页面可达）
  await expect(page.getByRole('navigation', { name: '主导航' })).toBeVisible({ timeout: 15_000 });
});
