import type { E2EConfig } from 'e2e';
import { web } from '@e2e-dev/web';

// tester-army/e2e 配置（docs/35）
// - 被测前端：Next.js dev server（127.0.0.1:3000，不用 localhost —— 本机 IPv6 解析坑）
// - 后端服务由测试外部启动（go -C backend run ./cmd/server，默认 127.0.0.1:8080）
// - 未配置模型 provider：全部用例只走 app/screen/expect 确定性断言，零模型调用
export default {
  targets: [
    {
      name: 'web',
      engine: web({ browser: 'chromium' }),
      app: {
        url: 'http://127.0.0.1:3000',
      },
    },
  ],
  // 起名生成是秒级全量计算 + 异步轮询，单个用例放宽到 3 分钟
  timeout: 180000,
  actionTimeout: 30000,
  assertionTimeout: 10000,
  workers: 1,
} satisfies E2EConfig;
