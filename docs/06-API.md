# 📡 API接口文档

> 宝宝起名大师 - 后端API接口定义

**Base URL:** `http://localhost:8080/api/v1`

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

---

## 1. 名字生成

### 生成名字

**POST** `/names/generate`

根据宝宝的出生信息生成推荐名字列表

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| surname | string | 是 | 姓氏 |
| gender | string | 是 | 性别：male / female |
| birth_year | int | 是 | 出生年份 |
| birth_month | int | 是 | 出生月份 |
| birth_day | int | 是 | 出生日期 |
| birth_hour | int | 是 | 出生时辰（0-23） |
| birth_minute | int | 否 | 出生分钟（0-59） |
| birth_location | string | 否 | 出生地点 |

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

**GET** `/names/:id`

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

**POST** `/bazi/analyze`

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| birth_year | int | 是 | 出生年份 |
| birth_month | int | 是 | 出生月份 |
| birth_day | int | 是 | 出生日期 |
| birth_hour | int | 是 | 出生时辰 |
| birth_minute | int | 否 | 出生分钟 |

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

**GET** `/yijing/hexagram/:id`

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

**GET** `/yijing/hexagram`

**响应：** 返回64卦数组

---

## 4. 生肖属相

### 获取单个生肖

**GET** `/zodiac/:animal`

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

**GET** `/zodiac`

**响应：** 返回12生肖数组

---

## 5. 历史记录

### 获取历史记录

**GET** `/history`

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

**POST** `/history`

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

**DELETE** `/history/:id`

**响应：**
```json
{
  "success": true,
  "message": "记录已删除"
}
```

---

## 6. 名字收藏

### 获取收藏列表

**GET** `/favorites`

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

**POST** `/favorites`

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

**DELETE** `/favorites/:id`

根据ID删除收藏

### 检查是否已收藏

**GET** `/favorites/check?surname=王&given_name=磊`

检查某个名字是否已收藏

**响应：**
```json
{
  "success": true,
  "data": {
    "exists": true,
    "favorite": { ... }
  }
}
```

---

## 7. 名字统计

### 获取名字重名率

**GET** `/namestat/:name`

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

**GET** `/namestat?limit=20`

获取常用名字排行榜

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| limit | int | 返回数量，默认20，最大100 |

### 获取省份重名率

**GET** `/namestat/:name/province?province=北京`

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

**POST** `/report/pdf`

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| data | object | 是 | 完整的分析数据 |

**响应：**
返回PDF文件下载

### 生成HTML报告

**POST** `/report/html`

**响应：**
返回HTML文件下载

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
