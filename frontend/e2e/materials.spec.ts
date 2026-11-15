/**
 * 物料管理页 E2E（@playwright/test，docs/47 S5）
 *
 * 覆盖 S3.5 物料管理页：状态表格渲染 + 手动重导（主体条目指纹短路跳过、
 * 零失败；classics 目录指纹语义导致重物化，见 docs/47 S5 遗留）。
 */
import { test, expect } from '@playwright/test';

test('物料页：状态表格渲染（物料/版本/行数）', async ({ page }) => {
  await page.goto('/#/materials');

  await expect(page.getByRole('heading', { name: '物料管理' })).toBeVisible();

  // 首跑播种后 materials_state 必有 kangxi / namer 等条目
  await expect(page.getByRole('cell', { name: 'kangxi' })).toBeVisible({ timeout: 30_000 });
  await expect(page.getByRole('cell', { name: 'namer' })).toBeVisible();

  // 行数为千分位数字格式（如 63,696）
  await expect(page.getByText(/^\d{1,3}(,\d{3})+$/).first()).toBeVisible();
});

test('物料页：手动重导幂等——主体条目指纹跳过且零失败', async ({ page }) => {
  await page.goto('/#/materials');
  await expect(page.getByRole('cell', { name: 'kangxi' })).toBeVisible({ timeout: 30_000 });

  // 重导（装配时物料目录=仓库根 data/）：绝大多数条目指纹未变秒级跳过；
  // classics 因目录指纹语义每次重物化（已知行为，docs/47 S5 遗留），耗时十秒级
  await page.getByRole('button', { name: '重新导入物料' }).click();

  // 汇总提示：零失败（写入/跳过数不锁死，避免与 classics 物化行为耦合）
  await expect(page.getByText(/物料导入完成：写入 \d+ 项、跳过 \d+ 项、失败 0 项/)).toBeVisible({
    timeout: 120_000,
  });

  // 明细：至少一条「指纹未变，跳过」，且无失败项错误
  await expect(page.getByText('指纹未变，跳过').first()).toBeVisible();
  await expect(page.getByText(/校验失败|导入失败|脏数据/)).toHaveCount(0);
});
