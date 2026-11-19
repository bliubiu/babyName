# DESIGN.md — 起名应用设计系统（单一真相源）

> 方法论参照：[impeccable](https://github.com/pbakaus/impeccable)（design vocabulary /
> critique / audit）与 [hallmark](https://github.com/Nutlope/hallmark)（Floor / Reflex /
> Finish 三层门槛）。本文件捕捉**现有**视觉系统，后续迭代（人或 AI）以此为基线，
> 与品牌不一致的改动必须先改这里再动代码。

## 1. 产品语境

- **受众**：新手父母；对"命理可信度"敏感，对"玄学套路"警惕
- **核心任务**：填宝宝信息 → 起名 → **选名**（收藏/对比）。页面组织围绕这条主线，
  每个页面的视觉权重应服务该页在主链路中的角色
- **气质**：文人纸卷——宣纸质感、水墨层次、印章点缀；克制、雅致，不做热闹促销感

## 2. 色板（CSS 变量，明暗双套自动适配）

| Token | 亮色 RGB | 暗色 RGB | 语义 |
|---|---|---|---|
| warm-white | 245 240 232 | 17 17 17 | 页面底色（宣纸/夜墨） |
| warm-white-light | 253 248 243 | 28 28 30 | 卡片底 |
| paper / paper-edge | 232 223 208 / 212 197 169 | 42 42 44 / 90 90 90 | 纸缘、分隔 |
| ink / ink-light | 26 26 26 / 74 74 74 | 232 228 220 / 168 168 168 | 正文 / 次要文字 |
| crimson (+light/dark) | 139 35 35 | 196 77 77 | 主行动色（标题/主按钮） |
| jade (+light/dark) | 61 122 106 | 94 168 152 | 辅助强调（拼音/链接/吉） |
| gold (+light/dark) | 189 154 90 | 206 176 122 | 分数/点缀 |
| seal-red | 204 42 42 | 224 74 74 | 印章红（收藏心/印章元素） |
| wuxing-金木水火土 | — | — | 五行标签专用五色 |

规则：新增颜色先问能否用上表；不引入新的设计 token（globals.css 头部注释约定）。

## 3. 字排

- 字体族：标题 `font-serif`（Noto Serif SC → STSong/SimSun 兜底）；正文/表单
  `font-sans`（Noto Sans SC → PingFang/YaHei）
- 标题：`text-3xl/4xl + tracking-widest`（书法字距）；正文行高 `1.7`（body 级）
- 拼音/粤拼：`text-xs + font-mono`（粤拼可省字距）
- 分数：`font-serif text-2xl font-bold`（列表第一扫视信号，见 NameCard）
- 字级阶梯：12(xs) / 14(sm) / 16(base) / 20-24(名字) / 30-36(页标题)，勿越级混用

## 4. 动效

- 动画族（tailwind keyframes，全部 `both` fill-mode）：fade-in-up（入场，可配
  animationDelay 逐卡 stagger）、seal-press（印章落）、scroll-unfurl（卷轴展开）、
  ink-spread/ink-bleed（墨晕）、heart-pop（收藏回弹 0.32s）、shimmer（骨架屏）
- 缓动约定：入场 `ease-out`；交互反馈 `duration-200~300`；**不做循环装饰动画**
- `prefers-reduced-motion: reduce` 由 globals.css 通配块整体归零（勿逐个组件写
  motion-reduce）
- hallmark 减法原则：新增动效先问"这个动效替用户看到了什么"，答不上来就不加

## 5. 组件模式

| 模式 | 类/文件 | 要点 |
|---|---|---|
| 主卡 | `.card` / `.consultation-sheet` | rounded-2xl + 双层柔和阴影 + corner-decor 角饰 |
| 主/次按钮 | `.btn-primary` / `.btn-secondary` | active:scale-95 触觉反馈 + min-w 防抖动 |
| 表单字段 | `.home-field` 族 | 44px 最小触控高、focus ring crimson、错误文案 `.home-field-error` |
| 导航 | red-ribbon 条 + aria-label 全覆盖 | 入口显隐走 featureFlags（VITE_SHOW_MATERIALS / VITE_SHOW_JSON） |
| 印章 | `.home-hero-mark`（"如意"） | seal-red 形制，内容可换；结果页评分徽章复用该形制 |
| 骨架屏 | `Skeleton.tsx` | 列表/卡片级，shimmer 驱动 |
| 进度 | `.home-progress` 族 | stage 文案 + percent，aria-live="polite" |
| 空态 | 卡片内大字 + 引导文案 | 印记式（大字号 + ink-light），不放插画 |
| 暗色 | CSS 变量整体翻转（`.dark` 重定义 :root 值） | 组件层勿散写 dark: 工具类，变量已覆盖 |

## 6. Floor 底线（永不发布，hallmark 对应）

1. **对比度**：正文 ≥4.5:1，大字（≥18.66px bold 或 ≥24px）≥3:1。**透明度档位
   即对比度档位**——2026-11-20 审计后建立的映射：
   - 辅助文字一律 `text-ink-light`（7.5:1），禁用 `/40` `/30` `/50` `/70`
   - jade 系辅助文字一律 `text-jade-dark/90`（≈6.5:1），禁用 `/50`~`/80`
   - 分数旁注（"分"字 10px）用 `text-ink-light/90`
   - 新组合先算对比度再落码（lum 公式见 CHANGELOG 2026.11.20 审计记录）
2. **触控目标 ≥44px**（min-h-[44px] 已成惯例）
3. **焦点可见**：focus-visible ring（crimson/30~40），outline-none 必须伴随 ring
4. **prefers-reduced-motion 生效**（通配块已覆盖，勿拆）
5. **移动端布局不坏**：新组件必须过 390px 宽度目检（两列折叠、进度条、托盘）
6. **不造假**：文案不假装成功（表单失败必须真实报错）；数据为空必须显示空态而非占位图

## 7. Reflex 对照（易掉入的套路，本项目处置）

| 套路（hallmark 清单） | 本项目处置 |
|---|---|
| 每个标题上方 eyebrow 小标签 | 首页 `home-kicker` 保留但**题签化**（引首章语义），不泛化到每个标题 |
| 标题里唯一的彩色词 | 标题整词同色（crimson），不做单字变色 |
| 全是空框/全是照片的卡片墙 | 候选名卡信息分层（名/拼音/评分/命理 chips），无空壳卡 |
| 每页结尾挂邮件订阅 | 无任何营销挂件，结果页以对比/收藏收尾 |
| 表单假装发送成功 | 生成失败走真实错误链（home-error + toast + 重试） |
| 卡片嵌套卡片 | 卡内分区用留白 + 分隔线，不用嵌套容器 |
