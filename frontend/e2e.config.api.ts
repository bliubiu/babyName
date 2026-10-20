import type { E2EConfig } from 'e2e';

/**
 * API 契约专用配置（docs/35 §4）
 *
 * 无引擎 target：纯 fetch 契约测试，完全不启动浏览器。
 * 依据 config.mdx：
 *   - 无引擎 target 必须显式给 platform（第 287 行）
 *   - 只有「app + 无引擎」才是 INVALID_CONFIG；无 app 无引擎合法
 *   - api-testing.mdx：无引擎 target 没有 app.baseUrl，URL 改由测试内硬编码/环境变量读取
 *
 * 被测后端由测试外部启动：go -C backend run ./cmd/server（127.0.0.1:8080）
 * 运行：pnpm exec e2e run --config e2e.config.api.ts tests/naming-api.e2e.ts
 */
export default {
  targets: [
    {
      name: 'api',
      platform: 'api',
    },
  ],
  timeout: 60000,
  workers: 1,
} satisfies E2EConfig;
