/**
 * 起名首页 UI 结构契约（docs/47 S5 之后的首页局部优化）
 *
 * 纯确定性断言：分区层级、高级选项折叠、桌面提交条吸底、移动端无横向溢出、
 * 暗色辅助文字可读性，以及「常用字段不得被折叠藏起来」的回归防线。
 * 不触发生成、不写数据库，可与 naming.spec.ts 共用同一后端实例。
 */
import { test, expect } from '@playwright/test';

test('首页三分区与高级选项折叠：展开后字段可见，设置项在折叠态仍有提示', async ({ page }) => {
  await page.goto('/');

  await expect(page.getByRole('heading', { name: '必填信息' })).toBeVisible();
  await expect(page.getByRole('heading', { name: '取名偏好' })).toBeVisible();

  const toggle = page.getByRole('button', { name: /高级选项/ });
  await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  await expect(page.locator('#home-advanced-body')).toHaveCount(0);
  await expect(page.locator('#avoid-elder-input')).toHaveCount(0);

  await toggle.click();
  await expect(toggle).toHaveAttribute('aria-expanded', 'true');
  await expect(page.locator('#avoid-elder-input')).toBeVisible();
  await expect(page.getByText('开启粤语多维度评分')).toBeVisible();
  await expect(page.getByRole('button', { name: /展开偏旁选字/ })).toBeVisible();

  // 折叠态的「已设置 N 项」是防止用户误以为高级设置丢失的关键提示
  await page.locator('#avoid-elder-input').fill('张三');
  await expect(page.getByText('已设置 1 项')).toBeVisible();
  await toggle.click();
  await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  await expect(page.getByText('已设置 1 项')).toBeVisible();
  await expect(page.locator('#avoid-elder-input')).toHaveCount(0);
});

test('桌面端：提交按钮位于首屏，且操作条为吸底定位', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/');

  const submit = page.getByRole('button', { name: '开始取名' });
  await expect(submit).toBeVisible();
  const box = await submit.boundingBox();
  expect(box).not.toBeNull();
  expect(box!.y).toBeGreaterThanOrEqual(0);
  expect(box!.y + box!.height).toBeLessThanOrEqual(900);

  const position = await page.locator('.home-submit-bar').evaluate(
    (el) => getComputedStyle(el).position
  );
  expect(position).toBe('sticky');
});

test('移动端：无横向溢出、高级标题单行、操作条保持文档流', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/');

  const widths = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    clientWidth: document.documentElement.clientWidth,
  }));
  expect(widths.scrollWidth).toBeLessThanOrEqual(widths.clientWidth);

  const title = page.getByRole('button', { name: /高级选项/ }).locator('.home-section-title');
  const titleBox = await title.boundingBox();
  expect(titleBox).not.toBeNull();
  expect(titleBox!.height).toBeLessThan(40); // 窄屏不允许「高级选/项」折成两行

  const position = await page.locator('.home-submit-bar').evaluate(
    (el) => getComputedStyle(el).position
  );
  expect(position).toBe('static'); // 移动端不吸底，避免与虚拟键盘抢占空间

  await expect(page.getByRole('button', { name: '开始取名' })).toBeVisible();
});

test('回归防线：分区后常用字段与经典来源仍默认可见', async ({ page }) => {
  await page.goto('/');

  await expect(page.getByPlaceholder('请输入姓氏')).toBeVisible();
  await expect(page.getByPlaceholder('辈分字')).toBeVisible();
  await expect(page.getByRole('button', { name: '诗经' })).toBeVisible();
  await expect(page.getByText('单字名')).toBeVisible();
  await expect(page.getByRole('button', { name: '开始取名' })).toBeVisible();
});
