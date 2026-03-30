# API接口文档

> 宝宝起名大师 - 后端API接口定义

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

## 目录

1. [名字生成](#1-名字生成)
2. [八字分析](#2-八字分析)
3. [易经卦象](#3-易经卦象)
4. [生肖属相](#4-生肖属相)
5. [历史记录](#5-历史记录)
6. [名字收藏](#6-名字收藏)
7. [名字统计](#7-名字统计)
8. [报告生成](#8-报告生成)
9. [黄历农历](#9-黄历农历)

---

## 1. 名字生成

### 生成名字

**POST** `/api/v1/names/generate`

根据宝宝的出生信息生成推荐名字列表

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| surname | string | 是 | 姓氏 |
| generation | string | 否 | 字辈 |
| generation_position | string | 否 | 字辈位置 |
| gender | string | 是 | 性别：male / female |
| birth_year | int | 是 | 出生年份 |
| birth_month | int | 是 | 出生月份 |
| birth_day | int | 是 | 出生日期 |
| birth_hour | int | 是 | 出生时辰（0-23） |
| birth_minute | int | 否 | 出生分钟（0-59） |
| birth_location | string | 否 | 出生地点 |
| birth_type | string | 否 | 出生类型 |
| name_type | string | 否 | 名字类型 |
| preferences | []string | 否 | 偏好设置 |
| name_length | int | 否 | 名字长度（1-4），默认2 |
| exclude_rare | bool | 否 | 是否排除生僻字 |
| wuxing_match | []string | 否 | 五行匹配 |
| source_classic | string | 否 | 经典来源 |
| min_strokes | int | 否 | 最小笔画数 |
| max_strokes | int | 否 | 最大笔画数 |
| include_poetry | bool | 否 | 是否包含诗词 |
| include_classic | bool | 否 | 是否包含经典 |
| meaning_keywords | []string | 否 | 寓意关键词 |
| pinyin_initial | string | 否 | 拼音首字母 |

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

**响应示例：**
```json
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
      "wuxing": {
        "jin": 2,
        "mu": 2,
        "shui": 2,
        "huo": 2,
        "tu": 2
      },
      "xiyongshen": ["水", "金"],
      "rishou": "戊",
      "rishou_wuxing": "土",
      "nayin": "海中金",
      "day_master": "土"
    },
    "nayin": "海中金",
    "zodiac": "兔",
    "hexagram": {
      "id": 1,
      "name": "乾",
      "number": 1,
      "symbol": "☰☰",
      "upper_trigram": 1,
      "lower_trigram": 1,
      "gua_ci": "元亨利贞",
      "xiang_ci": "天行健，君子以自强不息",
      "yao_ci": ["潜龙勿用", "见龙在田，利见大人", "君子终日乾乾"],
      "interpretation": "象征天，象征刚健有力..."
    },
    "hexagram_match": {
      "match": true,
      "score": 95,
      "analysis": "大吉"
    },
    "ziwei": {
      "tianzhu": "紫微",
      "diwei": "天府",
      "lu": "破军",
      "quan": "贪狼",
      "ke": "巨门",
      "ji": "廉贞",
      "zhushen": "紫微",
      "fuji": "左辅",
      "xingyao": "文昌、文曲、左辅、右弼",
      "gongwei": "命宫",
      "analysis": "命宫位于命宫，主星为紫微，辅星为左辅。四化情况：禄存于破军，权星于贪狼，科星于巨门，忌星于廉贞。紫微坐命，为人尊贵，有领导才能，适合从政或管理岗位。"
    },
    "names": [
      {
        "id": 1,
        "surname": "王",
        "given_name": "磊",
        "pinyin": "lěi",
        "meaning": "光明磊落",
        "wuxing": "土",
        "strokes": 15,
        "gender": "male",
        "score": 92.5
      }
    ]
  }
}
```

### 获取名字详情

**GET** `/api/v1/names/:id`

根据ID获取名字详情

**响应：**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "surname": "王",
    "given_name": "磊",
    "pinyin": "lěi",
    "meaning": "光明磊落",
    "wuxing": "土",
    "strokes": 15,
    "gender": "male",
    "score": 92.5
  }
}
```

---

## 2. 八字分析

### 分析八字

**POST** `/api/v1/bazi/analyze`

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| year | int | 是 | 出生年份 |
| month | int | 是 | 出生月份 |
| day | int | 是 | 出生日期 |
| hour | int | 是 | 出生时辰 |
| minute | int | 否 | 出生分钟 |

**响应：**
```json
{
  "success": true,
  "data": {
    "bazi": {
      "year": "癸卯",
      "month": "乙丑",
      "day": "戊子",
      "hour": "戊午"
    },
    "wuxing": {
      "jin": 2,
      "mu": 2,
      "shui": 2,
      "huo": 2,
      "tu": 2
    },
    "xiyongshen": ["水", "金"],
    "rishou": "戊",
    "rishou_wuxing": "土",
    "nayin": "海中金",
    "day_master": "土"
  }
}
```

---

## 3. 易经卦象

### 获取单个卦象

**GET** `/api/v1/yijing/hexagram/:id`

| 参数 | 类型 | 说明 |
|------|------|------|
| id | int | 卦象编号（1-64） |

**响应：**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "乾",
    "number": 1,
    "symbol": "☰☰",
    "upper_trigram": 1,
    "lower_trigram": 1,
    "gua_ci": "元亨利贞",
    "xiang_ci": "天行健，君子以自强不息",
    "yao_ci": ["潜龙勿用", "见龙在田，利见大人", "君子终日乾乾"],
    "interpretation": "象征天，象征刚健有力..."
  }
}
```

### 获取所有卦象

**GET** `/api/v1/yijing/hexagram`

**响应：** 返回64卦数组

---

## 4. 生肖属相

### 获取单个生肖

**GET** `/api/v1/zodiac/:animal`

| 参数 | 类型 | 说明 |
|------|------|------|
| animal | string | 生肖名称（鼠、牛、虎、兔...） |

**响应：**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "鼠",
    "wuxing": "水",
    "compatible": ["牛", "龙", "猴"],
    "conflicting": ["马", "兔", "羊"],
    "avoid_chars": "午、马、兔、羊",
    "lucky_numbers": [2, 3],
    "lucky_colors": "蓝色、金色、绿色",
    "lucky_direction": "东南、东北"
  }
}
```

### 获取所有生肖

**GET** `/api/v1/zodiac`

**响应：** 返回12生肖数组

---

## 5. 历史记录

### 获取历史记录

**GET** `/api/v1/history`

**响应：**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid-1",
      "surname": "王",
      "gender": "male",
      "birth_date": "2024-01-15",
      "birth_time": "12:00",
      "birth_location": "北京",
      "results": { ... },
      "created_at": "2024-01-15T12:00:00Z"
    }
  ]
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
| results | object | 是 | 生成结果 |

### 删除历史记录

**DELETE** `/api/v1/history/:id`

**响应：**
```json
{
  "success": true,
  "data": {
    "message": "删除成功"
  }
}
```

### 批量保存历史记录

**POST** `/api/v1/history/batch`

**请求参数：** 历史记录数组

**响应：**
```json
{
  "success": true,
  "data": {
    "ids": ["id1", "id2", "id3"]
  }
}
```

---

## 6. 名字收藏

### 获取收藏列表

**GET** `/api/v1/favorites`

获取所有已收藏的名字

**响应示例：**
```json
{
  "success": true,
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
  ]
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

**请求示例：**
```json
{
  "surname": "王",
  "given_name": "磊",
  "pinyin": "lěi",
  "gender": "male",
  "score": 92.5,
  "source": "诗词经典"
}
```

### 删除收藏

**DELETE** `/api/v1/favorites/:id`

根据ID删除收藏

### 检查是否已收藏

**GET** `/api/v1/favorites/check?surname=王&given_name=磊`

检查某个名字是否已收藏

**响应：**
```json
{
  "success": true,
  "data": {
    "is_favorite": true
  }
}
```

### 批量保存收藏

**POST** `/api/v1/favorites/batch`

**请求参数：** 收藏记录数组

**响应：**
```json
{
  "success": true,
  "data": {
    "ids": ["id1", "id2", "id3"]
  }
}
```

### 批量删除收藏

**DELETE** `/api/v1/favorites/batch`

**请求参数：**
```json
{
  "ids": ["id1", "id2", "id3"]
}
```

**响应：**
```json
{
  "success": true,
  "data": {
    "message": "批量删除成功"
  }
}
```

---

## 7. 名字统计

### 获取名字重名率

**GET** `/api/v1/namestat/:name`

获取指定名字的全国重名率统计

**响应示例：**
```json
{
  "success": true,
  "data": {
    "name": "伟",
    "count": 2456000,
    "rate": 1.7542857142857142,
    "rank": 1
  }
}
```

### 获取常用名字排行

**GET** `/api/v1/namestat?limit=20`

获取常用名字排行榜

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| limit | int | 返回数量，默认20，最大100 |

### 获取省份重名率

**GET** `/api/v1/namestat/:name/province?province=北京`

获取指定名字在特定省份的重名率

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| province | string | 省份名称，默认"北京" |

**响应：**
```json
{
  "success": true,
  "data": {
    "name": "伟",
    "count": 85000,
    "rate": 0.85,
    "province": "北京"
  }
}
```

---

## 8. 报告生成

### 生成PDF报告

**POST** `/api/v1/report/pdf`

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| data | object | 是 | 完整的分析数据 |

**响应：**
返回PDF文件下载

### 生成HTML报告

**POST** `/api/v1/report/html`

**响应：**
返回HTML文件下载

---

## 9. 黄历农历

### 获取黄历信息

**GET** `/api/v1/huangli`

根据日期获取黄历信息

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| year | int | 是 | 年份 |
| month | int | 是 | 月份 |
| day | int | 是 | 日期 |

**响应示例：**
```json
{
  "success": true,
  "data": {
    "year": 2024,
    "month": 1,
    "day": 15,
    "lunar_year": "癸卯",
    "lunar_month": "腊月",
    "lunar_day": "初五",
    "yi": "祭祀、祈福、求嗣",
    "ji": "嫁娶、开市、动土",
    "wuxing": "金",
    "chong": "兔"
  }
}
```

### 获取农历信息

**GET** `/api/v1/lunar`

根据日期获取农历信息

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| year | int | 是 | 年份 |
| month | int | 是 | 月份 |
| day | int | 是 | 日期 |
| hour | int | 否 | 小时，默认0 |
| minute | int | 否 | 分钟，默认0 |

**响应示例：**
```json
{
  "success": true,
  "data": {
    "year": 2024,
    "month": 1,
    "day": 15,
    "hour": 12,
    "minute": 0,
    "lunar_year": "癸卯",
    "lunar_month": "腊月",
    "lunar_day": "初五",
    "lunar_hour": "午时",
    "ganzhi": {
      "year": "癸卯",
      "month": "乙丑",
      "day": "戊子",
      "hour": "戊午"
    }
  }
}
```

---

## 错误响应

所有接口的错误响应格式：

```json
{
  "success": false,
  "error": "错误信息描述"
}
```

**HTTP状态码：**

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 速率限制

- 每个IP每分钟最多请求100次
- 超出限制将返回429状态码

---

## 版本控制

API版本通过URL路径控制，当前版本为v1，路径格式为`/api/v1/...`。

---

## 相关文档

- [开发文档](./03-开发文档.md)
- [部署使用手册](./05-部署使用手册.md)
- [API相对路径与动态端口适配](./08-API相对路径与动态端口适配.md)
