# API相对路径与动态端口适配技术文档

## 概述

本文档详细说明了宝宝起名大师项目中API相对路径配置和动态端口适配的实现机制，帮助开发者理解前端如何自动适配后端服务的端口变化。

---

## 问题背景

### 传统硬编码端口的问题

在传统的Web应用开发中，前端通常需要硬编码后端API的完整URL，包括端口号：

```typescript
// 传统方式：硬编码端口
const API_BASE_URL = 'http://localhost:8080/api';

fetch(`${API_BASE_URL}/v1/names/generate`, { ... })
```

**存在的问题：**

1. **端口不匹配**：当后端使用 `--port` 参数指定非默认端口时，前端无法自动适配
2. **维护困难**：每次修改后端端口都需要更新前端代码
3. **部署复杂**：不同环境（开发、测试、生产）需要不同的配置
4. **扩展性差**：难以支持多实例、负载均衡等高级部署场景

### 实际案例

在项目开发过程中，用户遇到以下问题：

- 后端运行在端口8083：`namemaster.exe -mode all -port 8083`
- 前端尝试访问8080端口：`http://localhost:8080/api/v1/names/generate`
- 结果：连接被拒绝（`net::ERR_CONNECTION_REFUSED`）

---

## 解决方案：相对路径配置

### 核心原理

使用相对路径 `/api` 代替完整URL，让浏览器自动使用当前页面的协议、主机和端口：

```typescript
// 新方式：相对路径
const API_BASE_URL = '/api';

fetch(`${API_BASE_URL}/v1/names/generate`, { ... })
```

**工作原理：**

1. **浏览器自动解析**：浏览器将相对路径解析为当前页面的完整URL
2. **同源策略**：确保API请求与前端页面同源，避免CORS问题
3. **动态适配**：无论后端运行在哪个端口，前端都能正确访问

### 实现代码

#### 前端API配置

文件：`frontend/src/lib/api.ts`

```typescript
const getApiBaseUrl = (): string => {
  if (typeof window !== 'undefined') {
    // 浏览器环境：使用相对路径，自动适配当前页面的主机和端口
    return '/api';
  }
  // 服务器端环境：使用环境变量或默认值
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';
};

const API_BASE_URL = getApiBaseUrl();
```

**设计要点：**

1. **环境检测**：通过 `typeof window !== 'undefined'` 判断运行环境
2. **浏览器环境**：使用相对路径 `/api`，自动适配
3. **服务器端环境**：使用环境变量或默认值，支持SSR

#### Next.js Rewrite配置

文件：`frontend/next.config.js`

```javascript
async rewrites() {
  return [
    {
      source: '/api/:path*',
      destination: 'http://localhost:8080/api/:path*',
    },
  ];
}
```

**作用：**

- 开发环境（`npm run dev`）：将 `/api/*` 请求代理到后端服务
- 生产环境（`npm run build`）：不使用Rewrite，直接访问同源API

---

## 部署场景适配

### 场景一：开发环境

**架构：**

```
浏览器 → Next.js Dev Server (3000) → Rewrite → 后端服务 (8080)
```

**访问流程：**

1. 前端访问：`http://localhost:3000`
2. API请求：`/api/v1/names/generate`
3. Next.js Rewrite：代理到 `http://localhost:8080/api/v1/names/generate`
4. 后端处理并返回结果

**配置：**

```javascript
// frontend/next.config.js
async rewrites() {
  return [
    {
      source: '/api/:path*',
      destination: 'http://localhost:8080/api/:path*',
    },
  ];
}
```

### 场景二：生产环境（all模式）

**架构：**

```
浏览器 → 后端服务 (8080) → 静态文件 + API
```

**访问流程：**

1. 前端访问：`http://localhost:8080`
2. 静态资源：直接从后端服务的静态文件目录加载
3. API请求：`/api/v1/names/generate`，直接访问同源API

**启动命令：**

```bash
namemaster.exe -mode all -port 8080
```

### 场景三：生产环境（Nginx代理）

**架构：**

```
浏览器 → Nginx (80) → 前端静态文件
                    → 后端API代理 (8080)
```

**访问流程：**

1. 前端访问：`http://your-domain.com`
2. 静态资源：Nginx直接返回前端静态文件
3. API请求：`/api/v1/names/generate`，Nginx代理到后端服务

**Nginx配置：**

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 前端静态文件
    location / {
        root /path/to/frontend/out;
        try_files $uri $uri/ /index.html;
    }

    # 后端API代理
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 场景四：自定义端口

**架构：**

```
浏览器 → 后端服务 (PORT) → 静态文件 + API
```

**访问流程：**

1. 前端访问：`http://localhost:PORT`
2. API请求：`/api/v1/names/generate`，自动适配PORT端口

**启动命令：**

```bash
namemaster.exe -mode all -port 8083
# 访问 http://localhost:8083
```

**优势：**

- 前端无需修改任何代码
- 自动适配任何端口
- 支持多实例部署

---

## 环境变量配置

### 前端环境变量

文件：`frontend/.env.local`

```env
# 可选：指定服务器端渲染时的API地址
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

**使用场景：**

- 服务器端渲染（SSR）时的API访问
- 特殊部署场景下的API地址配置

**优先级：**

1. 浏览器环境：相对路径 `/api`
2. 服务器端环境：环境变量 `NEXT_PUBLIC_API_URL`
3. 默认值：`http://localhost:8080/api`

### 后端环境变量

**静态文件路径：**

```bash
# Windows
set FRONTEND_STATIC_PATH=D:\19-Training\learngo\name\frontend\out

# Linux/Mac
export FRONTEND_STATIC_PATH=/path/to/frontend/out
```

**端口配置：**

```bash
# 通过命令行参数
namemaster.exe -port 8083
```

---

## 技术细节

### 相对路径 vs 绝对路径

| 特性 | 相对路径 | 绝对路径 |
|------|---------|---------|
| 示例 | `/api` | `http://localhost:8080/api` |
| 端口适配 | 自动适配 | 需要手动配置 |
| 部署灵活性 | 高 | 低 |
| 维护成本 | 低 | 高 |
| 适用场景 | 同源部署 | 跨域部署 |

### 浏览器URL解析规则

**相对路径解析示例：**

| 当前页面URL | 相对路径 | 解析结果 |
|------------|---------|---------|
| `http://localhost:8080/` | `/api` | `http://localhost:8080/api` |
| `http://localhost:8083/` | `/api` | `http://localhost:8083/api` |
| `http://example.com/` | `/api` | `http://example.com/api` |

**关键点：**

- 以 `/` 开头的相对路径是相对于域名根目录
- 浏览器自动使用当前页面的协议、主机和端口

### 同源策略

**同源定义：**

- 协议相同（http/https）
- 域名相同
- 端口相同

**优势：**

- 避免CORS（跨域资源共享）问题
- 简化安全配置
- 提高性能（无需预检请求）

---

## 最佳实践

### 1. 使用相对路径

```typescript
// 推荐
const API_BASE_URL = '/api';

// 不推荐
const API_BASE_URL = 'http://localhost:8080/api';
```

### 2. 环境检测

```typescript
const getApiBaseUrl = (): string => {
  if (typeof window !== 'undefined') {
    return '/api';
  }
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';
};
```

### 3. 生产环境使用Nginx代理

```nginx
location /api/ {
    proxy_pass http://localhost:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

### 4. 避免硬编码端口

```typescript
// 不推荐
const API_BASE_URL = `http://localhost:${PORT}/api`;

// 推荐
const API_BASE_URL = '/api';
```

### 5. 使用环境变量管理配置

```env
# frontend/.env.local
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

---

## 故障排查

### 问题1：API连接失败

**症状：**

```
net::ERR_CONNECTION_REFUSED
```

**排查步骤：**

1. 检查后端服务是否运行
2. 检查端口是否正确
3. 检查防火墙设置
4. 查看浏览器控制台网络请求

**解决方案：**

```bash
# 检查后端服务
netstat -ano | findstr :8080

# 测试API访问
curl http://localhost:8080/api/v1/health
```

### 问题2：404错误

**症状：**

```
404 Not Found
```

**排查步骤：**

1. 检查API路径是否正确
2. 检查后端路由配置
3. 检查Nginx配置（生产环境）

**解决方案：**

```bash
# 检查API路径
curl http://localhost:8080/api/v1/names/generate

# 检查Nginx配置
nginx -t
```

### 问题3：CORS错误

**症状：**

```
Access to fetch at 'http://localhost:8080/api' from origin 'http://localhost:3000' has been blocked by CORS policy
```

**排查步骤：**

1. 检查是否使用相对路径
2. 检查后端CORS配置
3. 检查是否同源

**解决方案：**

```typescript
// 使用相对路径
const API_BASE_URL = '/api';

// 或配置后端CORS
router.Use(cors.Default())
```

---

## 性能优化

### 1. 减少DNS查询

使用相对路径避免DNS查询：

```typescript
// 相对路径：无DNS查询
fetch('/api/v1/names/generate')

// 绝对路径：需要DNS查询
fetch('http://localhost:8080/api/v1/names/generate')
```

### 2. 连接复用

同源请求可以复用TCP连接：

```typescript
// 同源请求：复用连接
fetch('/api/v1/names/generate')
fetch('/api/v1/history')

// 跨源请求：新建连接
fetch('http://localhost:8080/api/v1/names/generate')
fetch('http://localhost:8080/api/v1/history')
```

### 3. 缓存策略

配置适当的缓存策略：

```nginx
location /api/ {
    proxy_pass http://localhost:8080;
    
    # 缓存配置
    proxy_cache_bypass $http_upgrade;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
}
```

---

## 总结

### 核心优势

1. **自动适配**：前端自动适配后端端口，无需手动配置
2. **简化部署**：支持多种部署场景，配置简单
3. **提高灵活性**：支持动态端口、多实例、负载均衡
4. **降低维护成本**：减少硬编码，提高代码可维护性

### 适用场景

- 同源部署（推荐）
- 动态端口配置
- 多实例部署
- 负载均衡

### 不适用场景

- 跨域部署（需要CORS配置）
- 前后端完全分离部署

### 相关文档

- [API接口文档](./06-API.md)
- [部署使用手册](./07-部署使用手册.md)
- [前端开发文档](./03-开发文档.md)

---

## 附录

### A. 完整配置示例

#### 前端配置

```typescript
// frontend/src/lib/api.ts
const getApiBaseUrl = (): string => {
  if (typeof window !== 'undefined') {
    return '/api';
  }
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';
};

const API_BASE_URL = getApiBaseUrl();

export async function generateNames(data: any): Promise<any> {
  const response = await fetch(`${API_BASE_URL}/v1/names/generate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
    },
    body: JSON.stringify(data),
  });
  return await handleResponse(response);
}
```

#### Next.js配置

```javascript
// frontend/next.config.js
const withPWA = require('next-pwa')({
  dest: 'public',
  register: true,
  skipWaiting: true,
  disable: process.env.NODE_ENV === 'development'
});

const nextConfig = {
  reactStrictMode: true,
  distDir: 'out',
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:8080/api/:path*',
      },
    ];
  },
};

module.exports = withPWA(nextConfig);
```

#### Nginx配置

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        root /path/to/frontend/out;
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### B. 测试命令

```bash
# 测试后端服务
curl http://localhost:8080/api/v1/health

# 测试前端页面
curl http://localhost:8080/

# 测试API请求
curl -X POST http://localhost:8080/api/v1/names/generate \
  -H "Content-Type: application/json" \
  -d '{"surname":"张","gender":"male","birth_year":2024,"birth_month":1,"birth_day":1,"birth_hour":12}'
```

### C. 常用端口

| 服务 | 默认端口 | 说明 |
|------|---------|------|
| Next.js Dev Server | 3000 | 开发环境前端 |
| 后端服务 | 8080 | 默认后端端口 |
| Nginx | 80 | 生产环境HTTP |
| PostgreSQL | 5432 | 数据库 |



