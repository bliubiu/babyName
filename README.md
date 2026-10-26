# 🍼 起名 (Name)

***

## 📖 项目简介

起名是一个免费的在线起名工具，旨在帮助新手父母为宝宝起一个寓意美好、符合中国传统玄学的名字。

**核心功能：**

- 🔮 八字五行分析 - 计算生辰八字、五行分布、喜用神
- 🧘 纳音五行 - 60甲子纳音查询
- 🐭 生肖属相 - 12生肖及喜忌
- 📜 易经64卦 - 姓名卦象解读
- ⭐ 智能评分 - 100分制综合评分
- ❤️ 名字收藏 - 收藏喜欢的名字
- � 名字对比 - 多名字横向对比
- 📖 诗词典故 - 名字关联诗词出处
- 🔍 重名率查询 - 名字重名率统计
- 📷 图片导出 - 一键导出名字推荐

***

## 🏗️ 技术架构

### 技术栈

| 层级     | 技术                                             |
| -------- | ------------------------------------------------ |
| 前端     | Go `html/template` 服务端渲染（SSR 全栈，无 React） |
| 样式     | Tailwind CSS（构建产物自托管，运行期无 CDN）      |
| 交互     | 服务端表单 POST + PRG 重定向，必要时少量原生 JS   |
| 后端     | Go 1.25 + Gin                                    |
| 存储     | SQLite（默认，WAL）/ 内存存储（仅显式配置）       |
| 架构     | DDD（领域驱动设计）+ TDD（测试驱动开发）         |

### 目录结构

```
name/
├── backend/                         # Go后端
│   ├── cmd/server/
│   │   └── main.go                 # 入口文件
│   ├── internal/
│   │   ├── application/            # 应用层
│   │   │   ├── handlers/           # HTTP处理器
│   │   │   └── services/           # 业务服务
│   │   ├── domain/                  # 领域层
│   │   │   ├── bazi/              # 八字计算
│   │   │   ├── yijing/            # 易经64卦
│   │   │   ├── zodiac/             # 生肖属相
│   │   │   ├── name/              # 名字生成
│   │   │   ├── namestat/          # 名字统计
│   │   │   └── classics/           # 诗词典故
│   │   └── infrastructure/          # 基础设施层
│   │       ├── database/
│   │       │   ├── memory/          # 内存存储（仅显式配置）
│   │       │   └── sqlite/          # SQLite 存储（默认）
│   │       └── web/                 # SSR 渲染层（templates/、static/）
│   ├── go.mod
│   └── application.yml              # 本地配置
│
├── data/                            # 字库与经典语料（运行时数据）
├── scripts/
│   ├── build.ps1                   # 一键构建（SSR 全栈）
│   ├── tailwind.config.js          # Tailwind 自托管构建配置
│   └── tailwind.input.css
├── nginx/                           # 反向代理配置（Docker 部署）
├── docker-compose.yml               # 生产编排（nginx + backend）
└── docs/                            # 项目文档
    ├── 00-页面原型设计及相关素材知识.md
    ├── 01-需求规格说明书.md
    ├── 02-API文档.md
    ├── 02-原型图设计.md
    ├── 03-UI设计规范.md
    ├── 03-开发文档.md
    ├── 04-功能增强方案.md
    ├── 04-用户文档.md
    ├── 05-开发文档.md
    ├── 06-API.md
    ├── 07-部署使用手册.md
    └── 08-API相对路径与动态端口适配.md
```

***

## 🚀 快速开始

### 方式一：直接运行EXE（推荐）

运行单体二进制可执行文件启动所有服务。【Web 静态页面内嵌；运行时数据（汉字库、频率统计等）位于二进制同级的 `data/` 目录，一键构建已自动复制】

```bash
# 构建单文件产物（dist/namer.exe + application.yml + data/）
.\scripts\build.ps1

# 进入产物目录运行
cd dist
.\namer.exe -mode api -port 8080

# 访问 http://localhost:8080
```

**参数选项：**

```bash
-port 8080        # 指定端口（默认8080）
-host localhost   # 指定主机（默认localhost）
-mode api         # 运行模式：api（SSR 无静态目录）或 all（历史兼容）
-db namer.db      # SQLite 数据库文件路径（覆盖配置文件）
-log logs         # 日志目录
-config application.yml  # 配置文件路径
```

**示例：**

```bash
# 使用 SQLite 持久化存储（默认）
.\namer.exe -mode api -db namer.db

# 使用端口8083
.\namer.exe -mode api -port 8083
```

### 方式二：从源码运行

```bash
cd backend
go run ./cmd/server
# 服务运行在 http://localhost:8080（SSR 页面与 API 同源）
```

打开浏览器访问 <http://localhost:8080>。

***

## 📡 API接口

### 名字生成

**POST** `/api/v1/names/generate`

```json
请求：
{
  "surname": "王",
  "gender": "male",
  "birth_year": 2024,
  "birth_month": 1,
  "birth_day": 15,
  "birth_hour": 12,
  "birth_location": "北京"
}

响应：
{
  "success": true,
  "data": {
    "bazi": {
      "bazi": {
        "year": "癸卯",
        "month": "乙丑",
        "day": "戊子",
        "hour": "戊午"
      },
      "wuxing": { "jin": 2, "mu": 2, "shui": 2, "huo": 2, "tu": 2 },
      "xiyongshen": ["水", "金"],
      "rishou": "戊",
      "rishou_wuxing": "土",
      "nayin": "海中金",
      "day_master": "土"
    },
    "nayin": "海中金",
    "zodiac": "兔",
    "hexagram": { ... },
    "names": [
      { "surname": "王", "given_name": "磊", "score": 92.5, ... }
    ]
  }
}
```

### 历史记录

| 方法     | 路径                    | 说明     |
| ------ | --------------------- | ------ |
| GET    | `/api/v1/history`     | 获取历史记录 |
| POST   | `/api/v1/history`     | 保存历史记录 |
| DELETE | `/api/v1/history/:id` | 删除历史记录 |

### 收藏管理

| 方法     | 路径                        | 说明      |
| ------ | ------------------------- | ------- |
| GET    | `/api/v1/favorites`       | 获取收藏列表  |
| POST   | `/api/v1/favorites`       | 添加收藏    |
| DELETE | `/api/v1/favorites/:id`   | 删除收藏    |
| GET    | `/api/v1/favorites/check` | 检查是否已收藏 |

### 名字统计

| 方法  | 路径                                | 说明       |
| --- | --------------------------------- | -------- |
| GET | `/api/v1/namestat/:name`          | 获取名字重名率  |
| GET | `/api/v1/namestat`                | 获取常用名字排行 |

### 其他接口

| 方法   | 路径                            | 说明       |
| ---- | ----------------------------- | -------- |
| POST | `/api/v1/bazi/analyze`        | 八字分析     |
| GET  | `/api/v1/yijing/hexagram/:id` | 获取卦象     |
| GET  | `/api/v1/yijing/hexagram`     | 获取全部卦象   |
| GET  | `/api/v1/zodiac/:animal`      | 获取生肖信息   |
| GET  | `/api/v1/zodiac`              | 获取全部生肖   |
| POST | `/api/v1/report/pdf`          | 生成PDF报告  |
| POST | `/api/v1/report/html`         | 生成HTML报告 |

***

## 🎨 设计规范

### 色彩系统

| 颜色名称 | 色值        | 用途       |
| ---- | --------- | -------- |
| 温润米白 | `#FDF8F3` | 主背景      |
| 暖白   | `#F5F0E8` | 卡片背景     |
| 墨黑   | `#2C2C2C` | 主要文字     |
| 深红   | `#8B2323` | 强调/按钮/心形 |
| 金色   | `#C9A962` | 高亮/评分    |
| 黛青   | `#4A6670` | 次要文字     |

### 字体

- 标题：Noto Serif SC（思源宋体）
- 正文：Noto Sans SC（思源黑体）

***

## 📋 功能说明

### 八字分析

- 计算年柱、月柱、日柱、时柱
- 统计五行分布（金木水火土）
- 判断日主强弱
- 计算喜用神

### 纳音五行

- 根据出生年份查询纳音
- 如：2024年 → "海中金"
- 分析名字与年命的关系

### 生肖属相

- 根据出生年份确定生肖
- 提供相合/相冲生肖
- 列出喜忌字符

### 易经64卦

- 根据姓名笔画起卦
- 解读卦辞、象辞
- 事业/财运/健康建议

### 名字评分

| 评分维度  | 权重  |
| ----- | --- |
| 八字匹配度 | 30% |
| 纳音五行  | 20% |
| 生肖适配  | 15% |
| 易经卦象  | 20% |
| 读音韵律  | 10% |
| 寓意内涵  | 5%  |

### 名字收藏

- 点击名字旁边的 ❤️ 即可收藏
- 在收藏页面管理已收藏的名字
- 支持删除收藏

### 名字对比

- 勾选要对比的名字（最多4个）
- 点击"对比"按钮进入对比页面
- 横向对比评分、五行、笔画等指标

### 诗词典故

- 选择"诗词经典"来源时，名字会关联诗词出处
- 显示诗句所属的诗词篇章
- 名字详解中展示诗词典故

### 重名率查询

- 输入名字查询全国重名率（基于真实人名频率语料库）
- 显示重名人数、占比与频率排名
- 显示重名程度（极高/较高/一般/较低/未收录）

### 图片导出

- 在结果页点击"导出"按钮
- 将名字推荐导出为PNG图片
- 方便分享给家人朋友

***

## 🔨 构建与部署

### 一键构建（推荐）

```bash
# 项目根目录执行
.\scripts\build.ps1
```

此脚本会：
1. 编译 Go 后端（SSR 模板与 Tailwind 已内嵌进二进制）
2. 复制 `application.yml` 与仓库根 `data/` 字库到 `dist/`
3. 生成 `start.bat` / `start.ps1`

### 手动构建

```bash
cd backend
go build -o ../dist/namer.exe ./cmd/server/
```

构建后需把 `backend/application.yml` 与仓库根 `data/` 复制到二进制同级目录。

### 部署

**二进制 + 数据目录部署（推荐）：**

```bash
.\scripts\build.ps1

cd dist
.\namer.exe -mode api
```

运行时数据从二进制同级的 `data/` 目录加载（汉字库、人名频率、命名质量门禁字表、经典文献等）。
部署时请保持 `namer.exe`、`application.yml` 与 `data/` 同级；字库缺失时服务会**拒绝启动**，
不会在空字库上继续提供 100% 失败的业务。

收藏、历史等用户数据默认写入同目录 `namer.db`（WAL 模式）。
SQLite 打开失败会拒绝启动；如需临时内存存储，需显式设置
`NAME_DATABASE_DRIVER=memory`（重启即丢数据）。

### 环境变量

配置项统一使用 `NAME_` 前缀（如 `NAME_DATABASE_PATH`、`NAME_SERVER_PORT`）。

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `NAME_DATABASE_DRIVER` | 存储驱动：`sqlite` / `memory` | `sqlite` |
| `NAME_DATABASE_PATH` | SQLite 文件路径 | `namer.db` |
| `NAME_SERVER_HOST` | 监听地址 | `localhost` |
| `NAME_SERVER_PORT` | 监听端口 | `8080` |
| `NAME_MODE` | `api`（SSR，无静态目录）/ `all` | `api` |
| `NAME_RATE_LIMIT_CAPACITY` | 限流容量 | `100` |
| `NAME_RATE_LIMIT_RATE` | 限流速率 | `10` |
| `DATA_DIR` | 字库/数据目录 | 启动目录下 `data/` |
| `CONFIG_DIR` | 配置文件搜索目录 | 可执行文件目录 |

***

## 🤝 贡献指南

欢迎提交Issue和Pull Request！

1. Fork 本仓库  github.com/bliubiao/name
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 打开Pull Request

***


