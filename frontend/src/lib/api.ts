import type { GenerateResponse, APIResponse, FavoriteData, GenerateRequest, StandardCharGroup, NameStat, AsyncSubmitResponse, TaskStatusResponse, ExploreResponse, EvaluateRequest, EvaluateResponse } from '@/types';
import type { HistoryResponse } from '@/types/api/history';
import type { FavoritesResponse } from '@/types/api/favorites';

// 动态获取API基础URL
// 策略优先级：
// 1. 环境变量：NEXT_PUBLIC_API_URL
// 2. 默认值：http://localhost:8080/api
const getApiBaseUrl = (): string => {
  // 优先使用环境变量，否则使用默认值
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';
};

const API_BASE_URL = getApiBaseUrl();

// 错误类型定义
export class ApiError extends Error {
  constructor(
    message: string,
    public statusCode?: number,
    public originalError?: unknown
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export class NetworkError extends ApiError {
  constructor(message: string = '网络连接失败，请检查网络设置') {
    super(message, 0);
    this.name = 'NetworkError';
  }
}

export class TimeoutError extends ApiError {
  constructor(message: string = '请求超时，请稍后重试') {
    super(message, 408);
    this.name = 'TimeoutError';
  }
}

export class ValidationError extends ApiError {
  constructor(message: string = '数据验证失败') {
    super(message, 400);
    this.name = 'ValidationError';
  }
}

export class ServerError extends ApiError {
  constructor(message: string = '服务器错误，请稍后重试') {
    super(message, 500);
    this.name = 'ServerError';
  }
}

export class AuthenticationError extends ApiError {
  constructor(message: string = '请先登录') {
    super(message, 401);
    this.name = 'AuthenticationError';
  }
}

export class ForbiddenError extends ApiError {
  constructor(message: string = '无权访问此资源') {
    super(message, 403);
    this.name = 'ForbiddenError';
  }
}

export class NotFoundError extends ApiError {
  constructor(message: string = '请求的资源不存在') {
    super(message, 404);
    this.name = 'NotFoundError';
  }
}

// 通用错误处理函数
async function handleResponse<T = unknown>(response: Response): Promise<T> {
  if (!response.ok) {
    let errorMessage = `HTTP error! status: ${response.status}`;
    
    try {
      const errorData = await response.json().catch(() => ({}));
      errorMessage = errorData.message || errorData.error || errorMessage;
      
      // 开发环境下打印详细错误信息
      if (process.env.NODE_ENV === 'development') {
        console.error('[API Error]', {
          status: response.status,
          url: response.url,
          error: errorData,
        });
      }
    } catch (e) {
      // 如果 JSON 解析失败，使用默认错误消息
    }
    
    // 根据状态码创建特定类型的错误
    if (response.status === 400) {
      throw new ValidationError(errorMessage);
    } else if (response.status === 401) {
      throw new AuthenticationError(errorMessage);
    } else if (response.status === 403) {
      throw new ForbiddenError(errorMessage);
    } else if (response.status === 404) {
      throw new NotFoundError(errorMessage);
    } else if (response.status === 408 || response.status === 504) {
      throw new TimeoutError(errorMessage);
    } else if (response.status >= 500) {
      throw new ServerError(errorMessage);
    } else if (response.status === 0) {
      throw new NetworkError(errorMessage);
    } else {
      throw new ApiError(errorMessage, response.status);
    }
  }
  
  try {
    return await response.json();
  } catch (error) {
    console.error('Failed to parse response JSON:', error);
    throw new ApiError('响应数据格式错误');
  }
}

// 带重试机制的fetch包装器
// 注意：maxRetries 和 retryDelay 仅对幂等请求（GET/HEAD）生效；
// 非幂等请求（POST/PUT/DELETE）不会重试，避免重复提交。
async function fetchWithRetry(
  url: string,
  options: RequestInit = {},
  maxRetries: number = 3,
  retryDelay: number = 1000
): Promise<Response> {
  let lastError: unknown;
  const method = (options.method || 'GET').toUpperCase();
  const isIdempotent = method === 'GET' || method === 'HEAD';
  const retries = isIdempotent ? maxRetries : 0;

  for (let attempt = 0; attempt <= retries; attempt++) {
    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 30000);

      const response = await fetch(url, {
        ...options,
        signal: controller.signal
      });

      clearTimeout(timeoutId);
      return response;
    } catch (error: unknown) {
      lastError = error;

      if (error instanceof Error && error.name === 'AbortError') {
        throw new TimeoutError();
      }

      if (attempt === retries) {
        throw new NetworkError();
      }

      // 指数退避 + 随机抖动，避免惊群效应
      const delay = Math.min(retryDelay * Math.pow(2, attempt) + Math.random() * 1000, 10000);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }

  throw lastError;
}

// 安全的数据转换函数
function safeParseInt(value: unknown, defaultValue: number = 0): number {
  if (typeof value === 'number') {
    return isNaN(value) ? defaultValue : value;
  }
  if (typeof value === 'string') {
    const parsed = parseInt(value, 10);
    return isNaN(parsed) ? defaultValue : parsed;
  }
  return defaultValue;
}

export async function generateNames(data: GenerateRequest): Promise<GenerateResponse> {
  const processedData = {
    ...data,
    birth_year: safeParseInt(data.birth_year),
    birth_month: safeParseInt(data.birth_month),
    birth_day: safeParseInt(data.birth_day),
    birth_hour: safeParseInt(data.birth_hour),
    birth_minute: safeParseInt(data.birth_minute),
  };

  if (!processedData.surname || processedData.surname.trim() === '') {
    throw new ValidationError('姓氏不能为空');
  }

  if (processedData.birth_year < 1900 || processedData.birth_year > 2100) {
    throw new ValidationError('出生年份必须在1900-2100之间');
  }

  const response = await fetchWithRetry(`${API_BASE_URL}/v1/names/generate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
    },
    body: JSON.stringify(processedData),
  });

  return await handleResponse(response);
}

// --- 异步生成任务（长任务体验） ---

/** 提交异步生成任务，立即返回 task_id（不阻塞等待引擎完成） */
export async function generateNamesAsync(data: GenerateRequest): Promise<string> {
  const processedData = {
    ...data,
    birth_year: safeParseInt(data.birth_year),
    birth_month: safeParseInt(data.birth_month),
    birth_day: safeParseInt(data.birth_day),
    birth_hour: safeParseInt(data.birth_hour),
    birth_minute: safeParseInt(data.birth_minute),
  };

  if (!processedData.surname || processedData.surname.trim() === '') {
    throw new ValidationError('姓氏不能为空');
  }
  if (processedData.birth_year < 1900 || processedData.birth_year > 2100) {
    throw new ValidationError('出生年份必须在1900-2100之间');
  }

  const response = await fetchWithRetry(`${API_BASE_URL}/v1/names/generate/async`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
    },
    body: JSON.stringify(processedData),
  });
  const result = await handleResponse<AsyncSubmitResponse>(response);
  if (!result.data?.task_id) {
    throw new ServerError('任务提交失败，请稍后重试');
  }
  return result.data.task_id;
}

/** 查询异步任务状态；成功时 data.result 为完整生成结果（取走即弃） */
export async function getTaskStatus(taskId: string): Promise<TaskStatusResponse['data']> {
  if (!taskId || taskId.trim() === '') {
    throw new ValidationError('任务ID不能为空');
  }
  const response = await fetchWithRetry(`${API_BASE_URL}/v1/names/task/${encodeURIComponent(taskId)}`);
  const result = await handleResponse<TaskStatusResponse>(response);
  return result.data;
}

/**
 * 提交异步生成并轮询直到完成，返回完整生成结果。
 * onProgress 每 500ms 回调一次（stage 阶段名 / percent 百分比）。
 */
export async function generateNamesWithProgress(
  data: GenerateRequest,
  onProgress?: (stage: string, percent: number) => void,
  pollIntervalMs: number = 500,
): Promise<GenerateResponse['data']> {
  const taskId = await generateNamesAsync(data);
  const deadline = Date.now() + 5 * 60 * 1000;

  for (;;) {
    const status = await getTaskStatus(taskId);
    if (!status) {
      throw new ServerError('任务不存在或已过期');
    }
    if (status.status === 'success' && status.result) {
      onProgress?.('完成', 100);
      return status.result;
    }
    if (status.status === 'failed') {
      throw new ServerError(status.error || '名字生成失败，请稍后重试');
    }
    onProgress?.(status.stage || '生成中', status.percent || 0);
    if (Date.now() > deadline) {
      throw new TimeoutError('生成耗时过长，请稍后重试');
    }
    await new Promise(resolve => setTimeout(resolve, pollIntervalMs));
  }
}

// --- 探索模式（换一批） ---

/** 从指定生成会话换一批：与上次结果零交集的随机候选 */
export async function exploreNames(generationId: string, count: number = 10): Promise<ExploreResponse['data']> {
  if (!generationId || generationId.trim() === '') {
    throw new ValidationError('生成会话ID不能为空');
  }
  const response = await fetchWithRetry(`${API_BASE_URL}/v1/names/generate/explore`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
    },
    body: JSON.stringify({ generation_id: generationId, count }),
  });
  const result = await handleResponse<ExploreResponse>(response);
  return result.data;
}

// --- 测名 + 风险体检 ---

/** 测名：输入姓名 + 生辰 → 完整评分报告 + 风险体检清单 */
export async function evaluateName(data: EvaluateRequest): Promise<EvaluateResponse['data']> {
  if (!data.surname || data.surname.trim() === '') {
    throw new ValidationError('姓氏不能为空');
  }
  if (!data.given_name || data.given_name.trim() === '') {
    throw new ValidationError('名字不能为空');
  }
  if (data.birth_year < 1900 || data.birth_year > 2100) {
    throw new ValidationError('出生年份必须在1900-2100之间');
  }

  const response = await fetchWithRetry(`${API_BASE_URL}/v1/names/evaluate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
    },
    body: JSON.stringify({
      ...data,
      birth_year: safeParseInt(data.birth_year),
      birth_month: safeParseInt(data.birth_month),
      birth_day: safeParseInt(data.birth_day),
      birth_hour: safeParseInt(data.birth_hour),
      birth_minute: safeParseInt(data.birth_minute ?? 0),
    }),
  });
  const result = await handleResponse<EvaluateResponse>(response);
  if (!result.data) {
    throw new ServerError('测名失败，请稍后重试');
  }
  return result.data;
}

export async function getHistory(): Promise<HistoryResponse> {
  const response = await fetchWithRetry(`${API_BASE_URL}/v1/history`);
  return await handleResponse(response);
}

export async function saveHistory(data: Record<string, unknown>): Promise<HistoryResponse> {
  const response = await fetchWithRetry(`${API_BASE_URL}/v1/history`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
    },
    body: JSON.stringify(data),
  });
  return await handleResponse(response);
}

export async function deleteHistory(id: string): Promise<HistoryResponse> {
  if (!id || id.trim() === '') {
    throw new ValidationError('历史记录ID不能为空');
  }

  const response = await fetchWithRetry(`${API_BASE_URL}/v1/history/${id}`, {
    method: 'DELETE',
  });
  return await handleResponse(response);
}

export async function getFavorites(): Promise<FavoritesResponse> {
  const response = await fetchWithRetry(`${API_BASE_URL}/v1/favorites`);
  return await handleResponse<FavoritesResponse>(response);
}

export async function saveFavorite(data: FavoriteData): Promise<FavoritesResponse> {
  // 验证必要字段
  if (!data.surname || data.surname.trim() === '') {
    throw new ValidationError('姓氏不能为空');
  }
  if (!data.given_name || data.given_name.trim() === '') {
    throw new ValidationError('名字不能为空');
  }
  if (data.score === undefined || data.score === null) {
    throw new ValidationError('评分不能为空');
  }

  const response = await fetchWithRetry(`${API_BASE_URL}/v1/favorites`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
    },
    body: JSON.stringify(data),
  });
  return await handleResponse(response);
}

export async function deleteFavorite(id: string): Promise<APIResponse> {
  if (!id || id.trim() === '') {
    throw new ValidationError('收藏ID不能为空');
  }

  const response = await fetchWithRetry(`${API_BASE_URL}/v1/favorites/${id}`, {
    method: 'DELETE',
  });
  return await handleResponse(response);
}

export async function getHuangli(year: number, month: number, day: number): Promise<APIResponse> {
  const safeYear = safeParseInt(year);
  const safeMonth = safeParseInt(month);
  const safeDay = safeParseInt(day);

  if (safeYear < 1900 || safeYear > 2100) {
    throw new ValidationError('年份必须在1900-2100之间');
  }
  if (safeMonth < 1 || safeMonth > 12) {
    throw new ValidationError('月份必须在1-12之间');
  }
  if (safeDay < 1 || safeDay > 31) {
    throw new ValidationError('日期必须在1-31之间');
  }

  const response = await fetchWithRetry(`${API_BASE_URL}/v1/huangli?year=${safeYear}&month=${safeMonth}&day=${safeDay}`);
  return await handleResponse(response);
}

export async function getLunarCalendar(year: number, month: number, day: number, hour: number = 0, minute: number = 0): Promise<APIResponse> {
  const safeYear = safeParseInt(year);
  const safeMonth = safeParseInt(month);
  const safeDay = safeParseInt(day);
  const safeHour = safeParseInt(hour);
  const safeMinute = safeParseInt(minute);

  if (safeYear < 1900 || safeYear > 2100) {
    throw new ValidationError('年份必须在1900-2100之间');
  }
  if (safeMonth < 1 || safeMonth > 12) {
    throw new ValidationError('月份必须在1-12之间');
  }
  if (safeDay < 1 || safeDay > 31) {
    throw new ValidationError('日期必须在1-31之间');
  }
  if (safeHour < 0 || safeHour > 23) {
    throw new ValidationError('小时必须在0-23之间');
  }
  if (safeMinute < 0 || safeMinute > 59) {
    throw new ValidationError('分钟必须在0-59之间');
  }

  const response = await fetchWithRetry(`${API_BASE_URL}/v1/lunar?year=${safeYear}&month=${safeMonth}&day=${safeDay}&hour=${safeHour}&minute=${safeMinute}`);
  return await handleResponse(response);
}

// --- 偏旁选字 API ---

export async function getCharGroups(): Promise<StandardCharGroup[]> {
  const response = await fetchWithRetry(`${API_BASE_URL}/v1/characters/groups`);
  const result = await handleResponse<APIResponse<StandardCharGroup[]>>(response);
  return result.data || [];
}

// --- 名字统计 API ---

export async function getNameStats(name: string): Promise<NameStat | null> {
  if (!name || name.trim() === '') {
    throw new ValidationError('名字不能为空');
  }
  const response = await fetchWithRetry(`${API_BASE_URL}/v1/namestat/${encodeURIComponent(name)}`);
  const result = await handleResponse<APIResponse<NameStat>>(response);
  return result.data || null;
}

export async function getTopNames(limit: number = 20): Promise<NameStat[]> {
  const safeLimit = safeParseInt(limit, 20);
  if (safeLimit < 1 || safeLimit > 100) {
    throw new ValidationError('查询数量必须在1-100之间');
  }
  const response = await fetchWithRetry(`${API_BASE_URL}/v1/namestat?limit=${safeLimit}`);
  const result = await handleResponse<APIResponse<NameStat[]>>(response);
  return result.data || [];
}
