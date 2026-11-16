/**
 * 结果页动作区 E2E（@playwright/test，docs/47 S5）
 *
 * 覆盖 Playwright 基础 14 例未触达的链路：换一批（ExploreNames）、导出 PDF
 * （GenerateReportPDF）、分享快照往返（CreateShare/GetShare + /share 落地页）、
 * 历史落库与删除（SaveHistory/GetHistory/DeleteHistory）。
 *
 * 这些动作是桌面端绑定层「参数形状」与「路由形态」的高危区：
 * Wails 绑定直接透传参数，不再有 HTTP handler 解包 {data:...} 信封；
 * 资产服务器也不给未知路径做 SPA fallback，分享链接必须走 hash 路由。
 */
import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';

/** 首页填表 → 生成 → 结果页出现候选名 */
async function gotoResult(page: Page): Promise<void> {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: '宝宝起名' })).toBeVisible();
  await page.getByPlaceholder('请输入姓氏').fill('李');
  await page.getByRole('button', { name: '开始取名' }).click();
  await expect(page.getByText(/^李/).first()).toBeVisible({ timeout: 90_000 });
}

/** 当前榜单候选全名（取收藏按钮 aria-label，避免匹配到标题/说明文本） */
function candidateLabels(page: Page): Promise<string[]> {
  return page.getByLabel(/^收藏 李/).evaluateAll(
    (els) => els.map((el) => el.getAttribute('aria-label')).filter((v): v is string => !!v),
  );
}

test('结果页「换一批」：与推荐榜零交集，可返回推荐榜', async ({ page }) => {
  await gotoResult(page);

  const before = await candidateLabels(page);
  expect(before.length).toBeGreaterThan(0);

  await page.getByRole('button', { name: '换一批' }).click();
  await expect(page.getByRole('heading', { name: '换一批' })).toBeVisible({ timeout: 60_000 });

  const after = await candidateLabels(page);
  expect(after.length).toBeGreaterThan(0);
  expect(after.filter((name) => before.includes(name))).toEqual([]);

  await page.getByRole('button', { name: '返回推荐榜' }).click();
  await expect(page.getByRole('heading', { name: '推荐名字' })).toBeVisible();
});

test('结果页导出 PDF：下载到有效的 PDF 文件', async ({ page }) => {
  await gotoResult(page);

  const [download] = await Promise.all([
    page.waitForEvent('download', { timeout: 60_000 }),
    page.getByRole('button', { name: '导出 PDF' }).click(),
  ]);

  expect(download.suggestedFilename()).toMatch(/^起名报告-.*\.pdf$/);
  const filePath = await download.path();
  expect(filePath).toBeTruthy();
  const bytes = readFileSync(filePath!);
  expect(bytes.subarray(0, 5).toString('ascii')).toBe('%PDF-');
  // 含真实结果数据的报告远大于空壳（字体子集 + 名字列表）
  expect(bytes.length).toBeGreaterThan(4 * 1024);
});

test('结果页分享：创建快照 → 打开分享链接 → 读回同一份结果', async ({ page, context }) => {
  // 不绑定 origin：server build（8080）与 v2 浏览器通道（34115）都能复用本用例
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await gotoResult(page);

  await page.getByRole('button', { name: '分享' }).click();

  // 成功、剪贴板降级、失败提示任一出现，说明绑定调用已返回
  await expect(
    page.getByText(/分享链接已复制|请手动复制|复制失败|创建分享失败|分享内容不合法/).first(),
  ).toBeVisible({ timeout: 30_000 });

  let url = '';
  try {
    url = await page.evaluate(() => navigator.clipboard.readText());
  } catch {
    // 浏览器通道下剪贴板可能不可用，回退到 toast 里暴露的绝对链接
  }
  if (!/share\?id=[a-f0-9]{32}/.test(url)) {
    const body = (await page.locator('body').textContent()) ?? '';
    url = (body.match(/https?:\/\/\S*share\?id=[a-f0-9]{32}/) ?? [''])[0];
  }
  expect(url, '创建分享失败时不会出现 /share?id= 链接').toMatch(/share\?id=[a-f0-9]{32}/);

  // 直接打开后端返回的链接：hash 路由缺失 / 参数包装错误都会在这里暴露
  await page.goto(url);
  await expect(page.getByRole('heading', { name: '推荐名字' })).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText(/共 \d+ 个名字/)).toBeVisible();
});

test('历史记录：生成自动落库，可删除该条', async ({ page }) => {
  await gotoResult(page);

  await page.goto('/#/history');
  await expect(page.getByRole('heading', { name: '历史记录' })).toBeVisible();
  // 首页提交时 saveHistory 已写入；至少一条记录且不是空态
  await expect(page.getByText('暂无历史记录')).toHaveCount(0);

  // 删除按钮为纯图标（IconTrash），按记录卡片内的按钮定位
  const deleteButtons = page.locator('div.card button');
  await expect(deleteButtons.first()).toBeVisible({ timeout: 30_000 });
  const countBefore = await deleteButtons.count();
  await deleteButtons.first().click();
  await expect(page.getByText('删除成功')).toBeVisible({ timeout: 30_000 });
  await expect(deleteButtons).toHaveCount(countBefore - 1, { timeout: 30_000 });
});
