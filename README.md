# 🍼 宝宝起名大师 (NameMaster)

***

## 📖 项目简介

宝宝起名大师是一个免费的在线起名工具，旨在帮助新手父母为宝宝起一个寓意美好、符合中国传统玄学的名字。

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

| 层级   | 技术                                 |
| ---- | ---------------------------------- |
| 前端   | Next.js 14 + React 18 + TypeScript |
| 样式   | Tailwind CSS                       |
| 状态管理 | Zustand                            |
| 后端   | Go + Gin                           |
| 存储   | 内存存储（默认）/ PostgreSQL（可选）           |
| 架构   | DDD（领域驱动设计）                        |

### 目录结构

```
namemaster/
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
│   │       └── database/memory/    # 内存存储
│   ├── go.mod
│   └── namemaster.exe              # 编译后的可执行文件
│
├── frontend/                        # Next.js前端
│   ├── src/
│   │   ├── app/                    # 页面
│   │   │   ├── page.tsx           # 首页
│   │   │   ├── result/            # 结果页
│   │   │   ├── history/           # 历史记录页
│   │   │   ├── favorites/         # 收藏页
│   │   │   ├── compare/           # 对比页
│   │   │   └── stat/              # 重名率查询页
│   │   ├── components/            # 组件
│   │   ├── lib/                   # 工具库
│   │   └── types/                 # TypeScript类型
│   ├── package.json
│   └── tailwind.config.ts
│
└── docs/                           # 项目文档
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

```bash
# 进入后端目录
cd backend

# 运行服务（默认端口8080）
.\namemaster.exe -mode all

# 访问 http://localhost:8080
```

**参数选项：**

```bash
-port 8080    # 指定端口（默认8080）
-host localhost  # 指定主机（默认localhost）
-mode all       # 运行模式：backend或all
```

**端口动态适配：**

- 前端使用相对路径 `/api`，自动适配后端端口
- 无需修改前端代码，支持任意端口配置
- 详细说明请参考 [API相对路径与动态端口适配](./docs/08-API相对路径与动态端口适配.md)

**示例：**

```bash
# 使用端口8083
.\namemaster.exe -mode all -port 8083
# 访问 http://localhost:8083

# 使用端口8084
.\namemaster.exe -mode all -port 8084
# 访问 http://localhost:8084
```

### 方式二：前后端分离

**1. 启动后端**

```bash
cd backend
go run cmd/server/main.go
# 后端运行在 http://localhost:8080
```

**2. 启动前端**

```bash
cd frontend
npm install
npm run dev
# 前端运行在 http://localhost:3000
```

**3. 访问应用**
打开浏览器访问 <http://localhost:3000>

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
| GET | `/api/v1/namestat/:name/province` | 获取省份重名率  |

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

- 输入名字查询全国重名率
- 可选择省份查询地区分布
- 显示重名程度（极高/较高/一般/较低）

### 图片导出

- 在结果页点击"导出"按钮
- 将名字推荐导出为PNG图片
- 方便分享给家人朋友

***

## 🤝 贡献指南

欢迎提交Issue和Pull Request！

1. Fork 本仓库  github.com/bliubiao/name
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 打开Pull Request

***

## 📄 许可证

MIT License
