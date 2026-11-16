/**
 * 起名主链路 E2E（@playwright/test，docs/47 S5）
 *
 * 运行：pnpm e2e（webServer 自动拉起 Wails v3 server build :8080，
 * 走与桌面 WebView 相同的 /wails/runtime HTTP IPC，docs/48 阶段 3）。
 *
 * 未配置模型：纯确定性断言（locator + expect），零模型调用。
 */
import { test, expect } from '@playwright/test';

test('起名主链路：首页填表 → 生成 → 结果页出现候选名', async ({ page }) => {
  await page.goto('/');

  // 首页标题（HomeContent h1）
  await expect(page.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  // 填姓氏（NameForm 姓氏输入框）
  const surname = page.getByPlaceholder('请输入姓氏');
  await expect(surname).toBeVisible();
  await surname.fill('李');

  // 生辰使用表单默认值，直接提交
  await page.getByRole('button', { name: '开始取名' }).click();

  // 生成成功后跳转结果页：等"李"开头候选名文本出现
  // （getByText 只匹配文本节点、不匹配 input 值，首页不会命中，能准确等到结果页）
  const firstCandidate = page.getByText(/^李/).first();
  await expect(firstCandidate).toBeVisible({ timeout: 90_000 });
});

test('姓氏为空提交：出现校验错误提示，停留在首页', async ({ page }) => {
  await page.goto('/');

  await expect(page.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  // 不填姓氏直接提交（HomeContent 在提交时校验并展示 #surname-error）
  await page.getByRole('button', { name: '开始取名' }).click();

  // 错误提示出现（表单错误 + toast 文案相同，取第一个即可）
  await expect(page.getByText('请输入姓氏').first()).toBeVisible();

  // 仍停留在首页：结果页的候选名内容不应出现
  await expect(page.getByText(/^李/)).toHaveCount(0);
});

test('经典来源 toggle：选中显示提示，取消后提示消失', async ({ page }) => {
  await page.goto('/');

  await expect(page.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  // 选中「诗经」经典来源（PreferenceSelector toggle 按钮）
  await page.getByRole('button', { name: '诗经' }).click();
  await expect(page.getByText(/当前已选择/)).toBeVisible();

  // 再次点击同一来源 → 取消选择，提示消失
  await page.getByRole('button', { name: '诗经' }).click();
  await expect(page.getByText(/当前已选择/)).toHaveCount(0);

  // 切换到另一来源，提示再次出现
  await page.getByRole('button', { name: '楚辞' }).click();
  await expect(page.getByText(/当前已选择/)).toBeVisible();
});

test('字辈「承」主链路：生成后候选名以「李承」开头', async ({ page }) => {
  await page.goto('/');

  await expect(page.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  await page.getByPlaceholder('请输入姓氏').fill('李');

  // 填写字辈「承」，位置保持默认「固定中间」
  await page.getByPlaceholder('辈分字').fill('承');

  await page.getByRole('button', { name: '开始取名' }).click();

  // 生成成功后跳转结果页，候选名以「李承」开头（字辈固定中间）
  const candidate = page.getByText(/^李承/).first();
  await expect(candidate).toBeVisible({ timeout: 90_000 });
});

test('名字类型「单字名」：生成后候选名以「李」开头', async ({ page }) => {
  await page.goto('/');

  await expect(page.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  await page.getByPlaceholder('请输入姓氏').fill('李');

  // 选择「单字名」（NameForm 名字类型单选）
  await page.getByText('单字名').click();

  await page.getByRole('button', { name: '开始取名' }).click();

  const firstCandidate = page.getByText(/^李/).first();
  await expect(firstCandidate).toBeVisible({ timeout: 90_000 });
});
