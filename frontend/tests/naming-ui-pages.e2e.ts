/**
 * GUI 全业务页面 E2E（tester-army/e2e，docs/35）
 *
 * 运行前置：
 * 1. 后端：go -C backend run ./cmd/server   （监听 127.0.0.1:8080，物料库已导入）
 * 2. 前端：cd frontend && pnpm dev          （监听 127.0.0.1:3000）
 * 运行：pnpm exec e2e run tests/naming-ui-pages.e2e.ts
 *
 * 未配置模型：纯确定性断言（locator + expect），零模型调用。
 *
 * 覆盖场景（全业务页面巡检）：
 * 1. 导航栏可达性：首页 → 测名 → 黄历 依次点击可达
 * 2. 测名链路：填姓氏+名字 → 开始测名 → 评分与八维依据出现
 * 3. 黄历页：今日黄历标题与宜忌内容渲染
 * 4. 重名率查询：填名字 → 查询 → 重名人数出现；热门名字榜可见
 * 5. 收藏链路（跨页）：首页生成 → 结果页收藏候选 → 收藏页出现 → 取消收藏
 * 6. 对比链路（跨页）：生成 → 选 2 个加入对比 → 对比页出现综合分析
 * 7. 历史页与 JSON 查看器页面可达
 */
import { test } from '@e2e-dev/web';
import { expect } from 'e2e';

test('导航栏可达性：首页 → 测名 → 黄历', async ({ app, screen }) => {
  await app.open('/');
  await expect(screen.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  // 点击导航「测名」进入评估页
  await screen.getByLabel('测名').click();
  await expect(screen.getByPlaceholder('如：王')).toBeVisible({ timeout: 15000 });

  // 点击导航「黄历」进入黄历页
  await screen.getByLabel('查看今日黄历').click();
  await expect(screen.getByRole('heading', { name: '今日黄历' })).toBeVisible({ timeout: 15000 });
});

test('测名链路：填姓名 → 开始测名 → 评分结果出现', async ({ app, screen }) => {
  await app.open('/evaluate');

  await screen.getByPlaceholder('如：王').fill('王');
  await screen.getByPlaceholder('如：浩然').fill('浩然');
  await screen.getByRole('button', { name: '开始测名' }).click();

  // 评分出现（X.X 分 形式）且八维评分依据渲染
  await expect(screen.getByText(/\d+\.\d+ 分/).first()).toBeVisible({ timeout: 60000 });
  await expect(screen.getByText('八维评分依据')).toBeVisible({ timeout: 60000 });
});

test('黄历页：今日黄历标题与宜忌内容渲染', async ({ app, screen }) => {
  await app.open('/huangli');
  await expect(screen.getByRole('heading', { name: '今日黄历' })).toBeVisible();
  // 宜/忌区块渲染（黄历数据来自后端 /api/v1/huangli）
  await expect(screen.getByText('宜', { exact: true }).first()).toBeVisible({ timeout: 15000 });
  await expect(screen.getByText('忌', { exact: true }).first()).toBeVisible({ timeout: 15000 });
});

test('重名率查询：填名字 → 查询 → 重名人数与热门名字榜出现', async ({ app, screen }) => {
  await app.open('/stat');
  await expect(screen.getByRole('heading', { name: '重名率查询' })).toBeVisible();

  await screen.getByLabel('输入名字查询重名率').fill('梓涵');
  await screen.getByRole('button', { name: '查询' }).click();

  await expect(screen.getByText('重名人数').first()).toBeVisible({ timeout: 30000 });
  await expect(screen.getByText('热门名字榜')).toBeVisible({ timeout: 30000 });
});

test('收藏链路：生成 → 收藏候选 → 收藏页出现 → 取消收藏', async ({ app, screen }) => {
  // 1. 首页生成
  await app.open('/');
  await expect(screen.getByRole('heading', { name: '宝宝起名' })).toBeVisible();
  await screen.getByPlaceholder('请输入姓氏').fill('李');
  await screen.getByRole('button', { name: '开始取名' }).click();
  await expect(screen.getByText(/^李/).first()).toBeVisible({ timeout: 90000 });

  // 2. 收藏第一个候选（aria-label：收藏 李XX）
  const favBtn = screen.getByLabel(/^收藏 李/).first();
  const fullName = (await favBtn.getAttribute('aria-label'))!.replace('收藏 ', '');
  await favBtn.click();

  // 3. 收藏页出现该名字
  await app.open('/favorites');
  await expect(screen.getByRole('heading', { name: '我的收藏' })).toBeVisible();
  await expect(screen.getByText(fullName).first()).toBeVisible({ timeout: 30000 });

  // 4. 取消收藏（收藏页删除按钮 aria-label 为「删除收藏」，title 同名）
  await screen.getByRole('button', { name: '删除收藏' }).first().click();
  await expect(screen.getByText(fullName).first()).toHaveCount(0, { timeout: 30000 });
});

test('对比链路：生成 → 选 2 个加入对比 → 对比页综合分析出现', async ({ app, screen }) => {
  await app.open('/');
  await expect(screen.getByRole('heading', { name: '宝宝起名' })).toBeVisible();
  await screen.getByPlaceholder('请输入姓氏').fill('李');
  await screen.getByRole('button', { name: '开始取名' }).click();
  await expect(screen.getByText(/^李/).first()).toBeVisible({ timeout: 90000 });

  // 选两个候选加入对比
  await screen.getByLabel(/^将 李\S+ 加入对比$/).first().click();
  await screen.getByLabel(/^将 李\S+ 加入对比$/).nth(1).click();

  // 点击「开始对比 2 个」（常驻托盘按钮，选中后文本带数量，db464d6 起形态）
  await screen.getByRole('button', { name: '开始对比 2 个' }).click();
  await expect(screen.getByRole('heading', { name: '名字对比' })).toBeVisible({ timeout: 30000 });
  await expect(screen.getByText('综合分析')).toBeVisible({ timeout: 30000 });
});

test('历史页与 JSON 查看器页面可达', async ({ app, screen }) => {
  await app.open('/history');
  await expect(screen.getByRole('heading', { name: '历史记录' })).toBeVisible();

  await app.open('/json-viewer');
  // JSON 查看器以 iframe 承载 public/json4matter.html（iframe 本体不进 ARIA 树，
  // 以主导航渲染确认页面可达）
  await expect(screen.getByRole('navigation', { name: '主导航' })).toBeVisible({ timeout: 15000 });
});
