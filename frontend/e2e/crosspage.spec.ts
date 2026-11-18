/**
 * 跨页链路 E2E（@playwright/test，docs/47 S5）
 *
 * 覆盖：收藏跨页（生成→收藏→收藏页→删除→空态）、对比跨页（选 2→对比页）。
 * 收藏用例自建自删，不向共享后端实例遗留数据。
 */
import { test, expect } from '@playwright/test';

test('收藏链路：生成 → 收藏候选 → 收藏页出现 → 取消收藏', async ({ page }) => {
  // 1. 首页生成
  await page.goto('/');
  await expect(page.getByRole('heading', { name: '宝宝起名' })).toBeVisible();
  await page.getByPlaceholder('请输入姓氏').fill('李');
  await page.getByRole('button', { name: '开始取名' }).click();
  await expect(page.getByText(/^李/).first()).toBeVisible({ timeout: 90_000 });

  // 2. 收藏第一个候选（aria-label：收藏 李XX）
  const favBtn = page.getByLabel(/^收藏 李/).first();
  const fullName = (await favBtn.getAttribute('aria-label'))!.replace('收藏 ', '');
  await favBtn.click();

  // 3. 收藏页出现该名字
  await page.goto('/#/favorites');
  await expect(page.getByRole('heading', { name: '我的收藏' })).toBeVisible();
  await expect(page.getByText(fullName).first()).toBeVisible({ timeout: 30_000 });

  // 4. 取消收藏（收藏页删除按钮 aria-label 为「删除收藏」，title 同名）
  await page.getByRole('button', { name: '删除收藏' }).first().click();
  await expect(page.getByText(fullName).first()).toHaveCount(0, { timeout: 30_000 });
});

test('导航入口：分发版默认隐藏物料管理', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: '宝宝起名' })).toBeVisible();
  // VITE_SHOW_MATERIALS 未注入（默认构建），物料入口不应出现在导航栏
  await expect(page.getByRole('button', { name: '物料管理' })).toHaveCount(0);
});

test('对比链路：生成 → 选 2 个加入对比 → 对比页综合分析出现', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: '宝宝起名' })).toBeVisible();
  await page.getByPlaceholder('请输入姓氏').fill('李');
  await page.getByRole('button', { name: '开始取名' }).click();
  await expect(page.getByText(/^李/).first()).toBeVisible({ timeout: 90_000 });

  // 选两个候选加入对比
  await page.getByLabel(/^将 李\S+ 加入对比$/).first().click();
  await page.getByLabel(/^将 李\S+ 加入对比$/).nth(1).click();

  // 点击「开始对比 2 个」（常驻托盘按钮，选中后文本带数量）
  await page.getByRole('button', { name: '开始对比 2 个' }).click();
  await expect(page.getByRole('heading', { name: '名字对比' })).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText('综合分析')).toBeVisible({ timeout: 30_000 });
});
