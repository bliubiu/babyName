# API接口文档

> 起名 - 后端API接口定义
>
> ★ 本文档与后端实际路由逐一核对（2026.11.08，docs/44 E2E 验证）：
> 接口清单以 `backend/cmd/server/main.go` 的 `setupRouter` 注册为准。
> 八字分析、易经卦象、生肖属相由**前端本地实现**（见第 9 节），后端无对应 HTTP 路由。

**Base URL:**
- 开发环境：`/api` (相对路径，自动适配当前页面端口)
- 生产环境：`/api` (相对路径，由Nginx代理到后端)
- 环境变量：`NEXT_PUBLIC_API_URL` (可选，用于服务器端渲染)

**API版本：** v1
**完整路径：** `/api/v1/*`

---

## API端口动态适配机制

### 相对路径配置

前端使用相对路径 `/api` 访问后端API，实现动态端口适配：

```typescript
const getApiBaseUrl = (): string => {
  if (typeof window !== 'undefined') {
    return '/api';
  }
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';
};
```

### 部署场景适配

| 部署场景 | 前端访问地址 | API访问方式 | 后端端口 |
|---------|------------|-----------|---------|
| 开发环境 | http://localhost:3000 | Next.js Rewrite代理 | 8080 |
| 生产环境(all模式) | http://localhost:8080 | 相对路径直接访问 | 8080 |
| 生产环境(Nginx) | http://your-domain.com | Nginx代理到后端 | 8080 |
| 自定义端口 | http://localhost:PORT | 相对路径自动适配 | PORT |

---

## 会话标识（访客隔离）

历史记录与收藏按**服务端会话**隔离归属。会话标识的解析优先级（`middleware.EnsureSessionID`）：

1. `X-Session-ID` 请求头（API 客户端显式携带）
2. `nm_sid` Cookie（浏览器自动携带，服务端首次响应种下，HttpOnly）
3. 两者均无时服务端新建并 `SetCookie`

**注意事项：**

- 合法格式为 16–64 位小写 hex（`services.NormalizeSessionID`），非法值一律视为无会话并重新生成，不会回落空串。
- 前端 `lib/api.ts` 统一在 `fetchWithRetry` 注入 `X-Session-ID`（localStorage 持久化，key `nm_session_id`）。此设计消除「并发首访各请求无 Cookie → 服务端各发新 session → 浏览器仅保留最后一个」的竞态（docs/44 缺陷 3）。
- CORS 已放行 `X-Session-ID` 请求头，第三方客户端可直接使用。

---

## 目录

1. [名字生成](#1-名字生成)（同步 / 异步 / 换一批 / 测名）
2. [历史记录](#2-历史记录)
3. [名字收藏](#3-名字收藏)
4. [名字统计](#4-名字统计)
5. [报告生成](#5-报告生成)
6. [黄历农历](#6-黄历农历)
7. [规范汉字分组](#7-规范汉字分组)
8. [健康检查与指标](#8-健康检查与指标)
9. [前端本地实现的功能域](#9-前端本地实现的功能域)（八字 / 卦象 / 生肖）

**接口速查表（与实际路由一一对应）：**

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/names/generate | 同步生成名字 |
| POST | /api/v1/names/generate/async | 提交异步生成任务 |
| GET | /api/v1/names/task/:id | 查询异步任务状态 |
| POST | /api/v1/names/generate/explore | 探索模式换一批 |
| POST | /api/v1/names/evaluate | 测名评估 |
| GET | /api/v1/history | 查询历史记录（分页） |
| POST | /api/v1/history | 保存历史记录 |
| DELETE | /api/v1/history/:id | 删除历史记录 |
| GET | /api/v1/favorites | 查询收藏列表（分页） |
| POST | /api/v1/favorites | 添加收藏 |
| DELETE | /api/v1/favorites/:id | 删除收藏 |
| POST | /api/v1/report/pdf | 生成PDF报告 |
| GET | /api/v1/namestat/:name | 名字重名率 |
| GET | /api/v1/namestat | 常用名字排行 |
| GET | /api/v1/huangli | 黄历信息 |
| GET | /api/v1/lunar | 农历信息 |
| GET | /api/v1/lunar/month | 农历月历 |
| GET | /api/v1/characters/groups | 规范汉字偏旁分组 |
| GET | /healthz | 服务健康检查 |
| GET | /readyz | 服务就绪检查 |
| GET | /metrics | 门禁指标快照 |

---

## 1. 名字生成

### 生成名字（同步）

**POST** `/api/v1/names/generate`

根据宝宝的出生信息生成推荐名字列表（含八字排盘、卦象、紫微、喜用神与逐名评分出典）。

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| surname | string | 是 | 姓氏 |
| generation | string | 否 | 字辈 |
| generation_position | string | 否 | 字辈位置：middle（名中）/ tail（名尾） |
| gender | string | 是 | 性别：male / female |
| birth_year | int | 是 | 出生年份 |
| birth_month | int | 是 | 出生月份 |
| birth_day | int | 是 | 出生日期 |
| birth_hour | int | 是 | 出生时辰（0-23） |
| birth_minute | int | 否 | 出生分钟（0-59） |
| birth_location | string | 否 | 出生地点（真太阳时校正） |
| birth_type | string | 否 | 出生类型 |
| name_type | string | 否 | 名字类型：single / double |
| preferences | []string | 否 | 偏好设置 |
| name_length | int | 否 | 名字长度（不含姓，1 单字 / 2 双字），默认2 |
| exclude_rare | bool | 否 | 是否排除生僻字 |
| wuxing_match | []string | 否 | 五行偏好（如 ["水","木"]） |
| source_classic | string | 否 | 经典来源（shijing/chuci/lunyu 等） |
| min_strokes | int | 否 | 最小笔画数 |
| max_strokes | int | 否 | 最大笔画数 |
| include_poetry | bool | 否 | 是否包含诗词字库 |
| include_classic | bool | 否 | 是否包含经典字库 |
| meaning_keywords | []string | 否 | 寓意关键词 |
| pinyin_initial | string | 否 | 拼音首字母 |
| avoid_elder_names | []string | 否 | 避讳长辈姓名（排除同形/同音字） |

**请求示例：**
```json
{
  "surname": "王",
  "gender": "male",
  "birth_year": 2024,
  "birth_month": 1,
  "birth_day": 15,
  "birth_hour": 12,
  "birth_location": "北京"
}
```

**响应 `data` 顶层字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| bazi | object | 八字排盘（四柱干支、五行计数、喜用神、日主、纳音等） |
| nayin | string | 年柱纳音 |
| zodiac | string | 生肖 |
| hexagram | object | 姓名卦（卦名/卦辞/爻辞） |
| hexagram_match | object | 卦象匹配（match/score/analysis） |
| hexagram_detail | object | 本/互/变三卦详解 |
| hexagram_interpretation | string | 卦象解读文字 |
| tiyong | object | 体用断卦 |
| ziwei | object | 紫微斗数（十四主星、四化、辅星等） |
| names | array | 候选名列表（默认 50 个） |
| generation_id | string | 生成会话标识（供 `/names/generate/explore` 换一批使用） |

**候选名元素（names[]）核心字段**（完整结构以实际响应为准，共 46 个字段）：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 名字编号 |
| surname / generation / given_name / full_name | string | 姓氏 / 字辈 / 名 / 全名 |
| pinyin | string | 拼音（自字库回填） |
| meaning | string | 寓意 |
| wuxing / nayin / strokes / gender | 混合 | 字属性：五行 / 纳音 / 笔画 / 适用性别 |
| bazi_score | number | 八字匹配分 |
| wuxing_analysis / meaning_detail / yinyun / sancai_analysis | string | 各维度依据文字 |
| poetry_source / poetry_chapter / poetry_sentence / poetry_author / poetry_dynasty / poetry_full_text | string | 诗词出典 |
| reasons | []string | 推荐理由 |
| total_score | number | 综合评分（100 分制） |
| wuxing_score / yinyun_score / meaning_score / sancai_score / zodiac_score / nayin_score / novelty_score / bigram_score / frequency_score | number | 八维分项 |
| hexagram / hexagram_meaning / hexagram_detail / tiyong | 混合 | 名字卦象 |
| score_detail | array | 八维评分明细 |
| jyutping | string | 粤拼 |
| risks | array | 风险体检清单（谐音/生僻字/多音字等） |

**响应示例（节选）：**
```json
{
  "code": 200,
  "success": true,
  "data": {
    "bazi": {
      "bazi": { "year": "甲辰", "month": "乙丑", "day": "戊子", "hour": "戊午" },
      "xiyongshen": ["木", "水", "金"]
    },
    "zodiac": "龙",
    "names": [
      {
        "id": 1,
        "surname": "王",
        "given_name": "珀晨",
        "full_name": "王珀晨",
        "pinyin": "pò chén",
        "total_score": 85.2
      }
    ],
    "generation_id": "g1794146486359784900-e698c30b"
  }
}
```

### 提交异步生成任务

**POST** `/api/v1/names/generate/async`

请求参数与同步生成完全相同。立即返回 `task_id`，不阻塞等待引擎完成；通过 `GET /names/task/:id` 轮询。并发任务已满时返回 503「当前生成任务过多，请稍后重试」。

**响应：**
```json
{
  "code": 200,
  "success": true,
  "data": { "task_id": "t1794147618696297800-11" }
}
```

### 查询异步任务状态

**GET** `/api/v1/names/task/:id`

任务不存在或已过期时返回 404。`status` 取值：`pending` / `running` / `success` / `failed`。

**响应：**
```json
{
  "code": 200,
  "success": true,
  "data": {
    "task_id": "t1794147618696297800-11",
    "status": "running",
    "stage": "八字排盘",
    "percent": 83
  }
}
```

- `status=failed` 时附带 `error` 字段（失败原因）。
- `status=success` 时附带 `result` 字段（完整生成结果，**取走即弃**：再次查询该任务返回 404）。

### 探索模式换一批

**POST** `/api/v1/names/generate/explore`

从指定生成会话换一批候选名，与上一轮结果零交集。会话不存在或已淘汰（内存 FIFO）时返回 400「生成会话不存在或已过期，请重新生成」。

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| generation_id | string | 是 | 生成响应返回的会话标识 |
| count | int | 否 | 换一批数量（默认 10，上限 30） |

**响应：**
```json
{
  "code": 200,
  "success": true,
  "data": {
    "generation_id": "g1794146486359784900-e698c30b",
    "names": [ ... ],
    "remaining": 940
  }
}
```

### 测名评估

**POST** `/api/v1/names/evaluate`

对给定姓名做与生成链路**同源同分**的八维评估（评分口径与 /names/generate 一致，e2e 巡检校验）。缺少 `given_name` 等必填参数时返回 400。

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| surname | string | 是 | 姓氏 |
| given_name | string | 是 | 名字 |
| gender | string | 是 | 性别：male / female |
| birth_year | int | 是 | 出生年份 |
| birth_month | int | 是 | 出生月份 |
| birth_day | int | 是 | 出生日期 |
| birth_hour | int | 否 | 出生时辰 |
| birth_minute | int | 否 | 出生分钟 |
| birth_location | string | 否 | 出生地（真太阳时校正） |
| birth_longitude | float | 否 | 显式经度（度，东经为正；>0 时优先于地点查表） |

**响应 `data` 核心字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| full_name | string | 全名 |
| name | object | 与 /generate 的 names[] 元素**同构**（含 score_detail / 出典 / 各维度分数），前端 NameCard/NameDetail 组件可直接复用 |
| bazi | object | 八字排盘（与 /generate 同源） |
| zodiac | string | 生肖 |
| risks | array | 风险体检清单（谐音/生僻字/多音字/负面联想/门禁字/撞名热度/户籍友好度） |

---

## 2. 历史记录

历史记录按服务端会话隔离（见「会话标识」）。前端在生成成功后会自动调用保存。

### 获取历史记录

**GET** `/api/v1/history`

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码（正整数，默认 1） |
| limit | int | 否 | 每页数量（默认 100，上限 100，超出返回 400 并告知真实生效值） |

**响应：**
```json
{
  "code": 200,
  "success": true,
  "data": {
    "records": [
      {
        "id": "uuid-1",
        "surname": "王",
        "gender": "male",
        "birth_date": "2024-01-15",
        "birth_time": "12:00",
        "birth_location": "北京",
        "results": { "names": [ ... ] },
        "created_at": "2024-01-15T12:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 100
  }
}
```

### 保存历史记录

**POST** `/api/v1/history`

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| surname | string | 是 | 姓氏 |
| gender | string | 是 | 性别 |
| birth_date | string | 是 | 出生日期 |
| birth_time | string | 是 | 出生时间 |
| birth_location | string | 否 | 出生地点 |
| results | object | 是 | 生成结果（JSON 原样存储） |

归属由服务端会话决定（客户端提交的 SessionID 字段被忽略）。

**响应：**
```json
{
  "code": 200,
  "success": true,
  "data": { "id": "uuid-1" }
}
```

### 删除历史记录

**DELETE** `/api/v1/history/:id`

**响应：**
```json
{
  "code": 200,
  "success": true,
  "data": { "message": "删除成功" }
}
```

---

## 3. 名字收藏

收藏按服务端会话隔离（见「会话标识」）。

### 获取收藏列表

**GET** `/api/v1/favorites`

**查询参数：** 与历史记录相同（`page` / `limit`，默认 100、上限 100）。

**响应：**
```json
{
  "code": 200,
  "success": true,
  "data": {
    "data": [
      {
        "id": "uuid-string",
        "surname": "王",
        "given_name": "磊",
        "pinyin": "lěi",
        "gender": "male",
        "score": 92.5,
        "source": "诗词经典",
        "notes": "",
        "created_at": "2024-01-15T10:30:00Z"
      }
    ],
    "records": [ ... 同 data，兼容字段 ... ],
    "total": 1,
    "page": 1,
    "limit": 100
  }
}
```

### 添加收藏

**POST** `/api/v1/favorites`

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| surname | string | 是 | 姓氏 |
| given_name | string | 是 | 名字 |
| pinyin | string | 是 | 拼音 |
| gender | string | 是 | 性别 |
| score | number | 是 | 评分 |
| source | string | 否 | 来源 |
| notes | string | 否 | 备注 |

**响应：**
```json
{
  "code": 200,
  "success": true,
  "data": { "id": "uuid-string" }
}
```

### 删除收藏

**DELETE** `/api/v1/favorites/:id`

根据ID删除收藏。**响应：** `data: { "message": "删除成功" }`（结构同历史删除）。

---

## 4. 名字统计

### 获取名字重名率

**GET** `/api/v1/namestat/:name`

获取指定名字的重名率统计（单字名查单字频率表，双字名查双字组合频率表；未收录返回 found=false 而非伪造数据）。

**响应：**
```json
{
  "code": 200,
  "success": true,
  "data": {
    "name": "伟",
    "count": 15567,
    "rate": 1.6253787538658475,
    "rank": 8
  }
}
```

### 获取常用名字排行

**GET** `/api/v1/namestat?limit=20`

获取常用名字排行榜。

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| limit | int | 返回数量，默认20，最大100 |

**响应：** NameStat 数组（结构同上）。

---

## 5. 报告生成

### 生成PDF报告

**POST** `/api/v1/report/pdf`

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| data | object | 是 | 完整的分析数据（原样传给报告引擎） |

**请求示例：**
```json
{
  "data": {
    "surname": "王",
    "gender": "male",
    "names": [ { "full_name": "王珀晨", "total_score": 85.2 } ]
  }
}
```

**响应：** `application/pdf` 二进制文件流（HTTP 200）；生成失败返回 500 JSON。

---

## 6. 黄历农历

### 获取黄历信息

**GET** `/api/v1/huangli`

根据公历日期获取黄历信息。

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| year | int | 是 | 年份 |
| month | int | 是 | 月份（1-12） |
| day | int | 是 | 日期（1-31） |

**响应 `data` 字段（实测 30 个字段，节选）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| date | string | 公历日期（YYYY-MM-DD） |
| year / month / day | string | 年/月/日干支（如 丙午 / 己亥 / 丙戌） |
| yi / ji | []string | 宜 / 忌事项列表 |
| jiShen / xiongSha | []string | 吉神 / 凶煞 |
| pengZhu | string | 彭祖百忌 |
| jieShen / fu | string | 解神 / 福 |
| chong / sha | string | 冲煞 |
| yearWuxing / monthWuxing / dayWuxing | string | 年/月/日纳音五行（如 天河水） |
| liuYao / zhiShen / taiShen / jianChu / xingXiu | string | 六曜 / 值神 / 胎神 / 建除 / 星宿 |
| xiShen / fuShen / caiShen / yangGui / yinGui | string | 喜神 / 福神 / 财神 / 阳贵 / 阴贵方位 |
| hours | array | 逐时吉凶 |

**响应示例（节选）：**
```json
{
  "code": 200,
  "success": true,
  "data": {
    "date": "2026-11-08",
    "year": "丙午",
    "month": "己亥",
    "day": "丙戌",
    "yi": ["嫁娶", "纳采", "订盟", "祭祀", "祈福", "求嗣"],
    "ji": ["开市", "立券", "交易", "挂匾", "开仓"],
    "yearWuxing": "天河水",
    "monthWuxing": "平地木",
    "dayWuxing": "屋上土",
    "pengZhu": "天牢",
    "jieShen": "天喜",
    "fu": "天福"
  }
}
```

### 获取农历信息

**GET** `/api/v1/lunar`

根据公历日期时间获取农历信息与四柱干支。

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| year | int | 是 | 年份 |
| month | int | 是 | 月份 |
| day | int | 是 | 日期 |
| hour | int | 否 | 小时，默认0 |
| minute | int | 否 | 分钟，默认0 |

**响应 `data` 字段（实测 26 个字段，节选）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| year / month / day | int | 公历日期回显 |
| isLeap | bool | 是否闰月 |
| lunarMonthNumber / lunarDayNumber | int | 农历月 / 日序号 |
| yearGan / yearZhi / monthGan / monthZhi / dayGan / dayZhi / hourGan / hourZhi | string | 四柱天干地支 |
| nayin / yearNayin / monthNayin / dayNayin | string | 纳音 |
| lunarYearName / lunarMonthName / lunarDayName | string | 农历年/月/日名称（如 丙午年 / 九月 / 初三） |
| dateStr / lunarDate / lunarMonth | string | 格式化日期字符串 |
| solarTerm | string | 节气 |

### 获取农历月历

**GET** `/api/v1/lunar/month`

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| year | int | 是 | 年份 |
| month | int | 是 | 月份 |

**响应：**
```json
{
  "code": 200,
  "success": true,
  "data": {
    "year": 2026,
    "month": 11,
    "days": [
      { "day": 1, "solarTerm": "", "lunarFestival": "" }
    ]
  }
}
```

---

## 7. 规范汉字分组

### 获取规范汉字偏旁分组

**GET** `/api/v1/characters/groups`

返回《通用规范汉字表》按偏旁部首的分组数据（数据源：namer.json 顶层 charGroups，启动时载入内存）。前端「展开偏旁选字」使用。

**响应：** `data` 为分组数组：

| 字段 | 类型 | 说明 |
|------|------|------|
| radical | string | 偏旁部首（如 氵） |
| name | string | 分组名称 |
| meaning | string | 部首含义 |
| chars | []string | 组内汉字列表 |

实测共 29 组。

---

## 8. 健康检查与指标

### 服务健康检查

**GET** `/healthz`

检查服务是否正常运行（不检查数据库）。

**响应：**
```json
{ "status": "ok", "timestamp": 1715856000 }
```

### 服务就绪检查

**GET** `/readyz`

检查服务是否就绪（含数据库连接检查；异常时 HTTP 503）。

**响应：**
```json
// 正常
{ "db": "connected", "status": "ready" }

// 异常
{ "status": "not ready", "db": "disconnected" }
```

### 门禁指标快照

**GET** `/metrics`

返回质量门禁命中审计与复审闭环指标快照（docs/33 M4/M6，`fate.GateMetricsSnapshot()`），供监控采集。

---

## 9. 前端本地实现的功能域

以下功能域**无后端 HTTP 路由**，全部由前端在浏览器本地计算（离线可用），文档保留其说明以澄清架构归属。此前版本曾误列为后端 API（`/bazi/analyze`、`/yijing/*`、`/zodiac/*`），实际从未实现，已从接口清单移除。

### 八字分析

- 实现：`frontend/src/lib/` 本地排盘 + 生成接口响应内嵌的 `bazi` 字段（后端领域层 `internal/domain/bazi` 计算，随 /generate、/evaluate、/lunar 返回）。
- 前端消费：结果页/测名页的 BaziAnalysis 组件、BirthdayPicker 时辰选择（`shichen.ts`）。

### 易经卦象

- 实现：`frontend/src/lib/hexagram.ts`（64 卦本地数据与计算）+ HexagramDisplay / HexagramLines / HexagramDetailDisplay 组件。
- 后端仅在生成/测名响应中内嵌 `hexagram` 系列字段（服务端领域层 `internal/domain/fate` 计算）。

### 生肖属相

- 实现：前端本地（`internal/domain/fate` 服务端随生成响应返回 `zodiac` 字段；生肖喜忌文案由前端组件渲染）。

> 若未来需要独立的后端查询接口（如 `/api/v1/yijing/hexagram/:id`），物料库 `yijing` 表已具备 64 卦数据，需另行立项实现路由与 handler；在落地前请勿在客户端调用上述路径。

---

## 错误响应

所有接口的统一响应包装（`internal/application/response`）：

**成功：**
```json
{
  "code": 200,
  "success": true,
  "data": { ... }
}
```

**失败：**
```json
{
  "code": 400,
  "success": false,
  "message": "请求参数格式错误"
}
```

> 注意：失败响应使用 `message` 字段（`data` 缺省不输出）；服务层错误细节只落服务端日志，不回传客户端（docs/43 P1-10）。

**HTTP状态码：**

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误（含 page/limit 非法、任务/会话不存在） |
| 404 | 资源不存在（如异步任务已过期或取走即弃） |
| 429 | 请求过于频繁（限流） |
| 500 | 服务器内部错误 |
| 503 | 当前生成任务过多（异步提交被拒，稍后重试） |

---

## 速率限制

- 每个IP使用令牌桶算法限流，默认容量100个令牌，每秒生成10个令牌
- 超出限制将返回429状态码
- 可通过命令行参数 `-rate-capacity` 和 `-rate-rate` 调整

---

## 版本控制

API版本通过URL路径控制，当前版本为v1，路径格式为`/api/v1/...`。

---

## 相关文档

- [开发文档](./03-开发文档.md)
- [部署使用手册](./05-部署使用手册.md)
- [API相对路径与动态端口适配](./08-API相对路径与动态端口适配.md)
- [CLI/GUI 全业务功能端到端测试验证报告](./44-CLI与GUI全业务功能端到端测试验证报告.md)（本文档重写依据）
