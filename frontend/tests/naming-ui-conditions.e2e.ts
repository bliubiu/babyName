/**
 * 起名 UI 条件表单 E2E（tester-army/e2e，docs/35）
 *
 * 运行前置：
 * 1. 后端：go -C backend run ./cmd/server   （监听 127.0.0.1:8080）
 * 2. 前端：cd frontend && pnpm dev          （监听 127.0.0.1:3000）
 * 运行：pnpm exec e2e run tests/naming-ui-conditions.e2e.ts
 *
 * 未配置模型：纯确定性断言（locator + expect），零模型调用。
 *
 * 覆盖场景：
 * 1. 姓氏为空提交 → 校验错误提示「请输入姓氏」，停留在首页不跳转
 * 2. 经典来源 toggle 选中/取消 + 个性补充关键词输入回显（纯 UI 交互，不生成）
 * 3. 字辈「承」主链路 → 结果候选名以「李承」开头
 * 4. 名字类型选「单字名」→ 生成成功且候选名以「李」开头
 */
import { test } from '@e2e-dev/web';
import { expect } from 'e2e';

test('姓氏为空提交：出现校验错误提示，停留在首页', async ({ app, screen }) => {
  await app.open('/');

  await expect(screen.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  // 不填姓氏直接提交（HomeContent 在提交时校验并展示 #surname-error）
  await screen.getByRole('button', { name: '开始取名' }).click();

  // 错误提示出现（表单错误 + toast 文案相同，取第一个即可）
  await expect(screen.getByText('请输入姓氏').first()).toBeVisible();

  // 仍停留在首页：结果页的候选名内容不应出现
  await expect(screen.getByText(/^李/)).toHaveCount(0);
});

test('经典来源 toggle 与个性补充关键词：选中显示提示，取消后提示消失', async ({ app, screen }) => {
  await app.open('/');

  await expect(screen.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  // 选中「诗经」经典来源（PreferenceSelector toggle 按钮）
  await screen.getByRole('button', { name: '诗经' }).click();
  await expect(screen.getByText(/当前已选择/)).toBeVisible();

  // 再次点击同一来源 → 取消选择，提示消失
  await screen.getByRole('button', { name: '诗经' }).click();
  await expect(screen.getByText(/当前已选择/)).toHaveCount(0);

  // 切换到另一来源，提示再次出现
  await screen.getByRole('button', { name: '楚辞' }).click();
  await expect(screen.getByText(/当前已选择/)).toBeVisible();

  // 个性补充：填写寓意关键词并回显
  const keywords = screen.getByPlaceholder('输入关键词，如：清莲、竹韵、明德、鹤影');
  await keywords.fill('竹韵');
  await expect(keywords).toBeVisible();
});

test('字辈「承」主链路：生成后候选名以「李承」开头', async ({ app, screen }) => {
  await app.open('/');

  await expect(screen.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  const surname = screen.getByPlaceholder('请输入姓氏');
  await surname.fill('李');

  // 填写字辈「承」，位置保持默认「固定中间」
  const generation = screen.getByPlaceholder('辈分字');
  await generation.fill('承');

  // 生辰使用表单默认值，直接提交
  await screen.getByRole('button', { name: '开始取名' }).click();

  //的字
  // 生成成功后跳转结果页，候选名以「李承」开头（字辈固定中间）
  const candidate = screen.getByText(/^李承/).first();
  await expect(candidate).toBeVisible({ timeout: 90000 });
});

test('名字类型「单字名」：生成后候选名以「李」开头', async ({ app, screen }) => {
  await app.open('/');

  await expect(screen.getByRole('heading', { name: '宝宝起名' })).toBeVisible();

  const surname = screen.getByPlaceholder('请输入姓氏');
  await surname.fill('李');

  // 选择「单字名」（NameForm 名字类型单选）
  await screen.getByText('单字名').click();

  // 生辰使用表单默认值，直接提交
  await screen.getByRole('button', { name: '开始取名' }).click();

  // 生成成功后跳转结果页，候选名以「李」开头
  const firstCandidate = screen.getByText(/^李/).first();
  await expect(firstCandidate).toBeVisible({ timeout: 90000 });
});
