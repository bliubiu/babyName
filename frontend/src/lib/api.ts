// 动态获取API基础URL
// 策略优先级：
// 1. 浏览器环境：使用相对路径 /api，让浏览器自动使用当前页面的主机和端口
// 2. 环境变量：NEXT_PUBLIC_API_URL
// 3. 默认值：http://localhost:8080/api
const getApiBaseUrl = (): string => {
  if (typeof window !== 'undefined') {
    // 浏览器环境：使用相对路径，自动适配当前页面的主机和端口
    // 这样无论后端运行在哪个端口，前端都能正确访问
    return '/api';
  }
  // 服务器端环境：使用环境变量或默认值
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';
};

const API_BASE_URL = getApiBaseUrl();

// 错误类型定义
export class ApiError extends Error {
  constructor(
    message: string,
    public statusCode?: number,
    public originalError?: any
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
async function handleResponse(response: Response): Promise<any> {
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
async function fetchWithRetry(
  url: string,
  options: RequestInit = {},
  maxRetries: number = 3,
  retryDelay: number = 1000
): Promise<Response> {
  let lastError: any;
  
  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 30000); // 30秒超时
      
      const response = await fetch(url, {
        ...options,
        signal: controller.signal
      });
      
      clearTimeout(timeoutId);
      return response;
    } catch (error: any) {
      lastError = error;
      
      // 如果是AbortError，说明超时
      if (error.name === 'AbortError') {
        throw new TimeoutError();
      }
      
      // 如果是最后一次尝试，直接抛出错误
      if (attempt === maxRetries) {
        throw new NetworkError();
      }
      
      // 等待一段时间后重试
      await new Promise(resolve => setTimeout(resolve, retryDelay * (attempt + 1)));
    }
  }
  
  throw lastError;
}

// 安全的数据转换函数
function safeParseInt(value: any, defaultValue: number = 0): number {
  if (typeof value === 'number') {
    return isNaN(value) ? defaultValue : value;
  }
  if (typeof value === 'string') {
    const parsed = parseInt(value, 10);
    return isNaN(parsed) ? defaultValue : parsed;
  }
  return defaultValue;
}

import type { GenerateResponse } from '@/types';

export async function generateNames(data: any): Promise<GenerateResponse> {
  try {
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
  } catch (error) {
    throw error;
  }
}

import type { HistoryResponse } from '@/types/api/history';

// ... existing code ...

export async function getHistory(): Promise<HistoryResponse> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/v1/history`);
    return await handleResponse(response);
  } catch (error) {
    console.error('Error getting history:', error);
    throw error;
  }
}

export async function saveHistory(data: any): Promise<HistoryResponse> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/v1/history`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json; charset=utf-8',
      },
      body: JSON.stringify(data),
    });
    return await handleResponse(response);
  } catch (error) {
    console.error('Error saving history:', error);
    throw error;
  }
}

export async function deleteHistory(id: string): Promise<HistoryResponse> {
  try {
    if (!id || id.trim() === '') {
      throw new ValidationError('历史记录ID不能为空');
    }
    
    const response = await fetchWithRetry(`${API_BASE_URL}/v1/history/${id}`, {
      method: 'DELETE',
    });
    return await handleResponse(response);
  } catch (error) {
    console.error('Error deleting history:', error);
    throw error;
  }
}

import type { Hexagram } from '@/types';

export async function getHexagrams(): Promise<Hexagram[]> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/v1/yijing/hexagram`);
    return await handleResponse(response);
  } catch (error) {
    console.error('Error getting hexagrams:', error);
    throw error;
  }
}

export async function getZodiacs(): Promise<any> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/v1/zodiac`);
    return await handleResponse(response);
  } catch (error) {
    console.error('Error getting zodiacs:', error);
    throw error;
  }
}

export type FavoriteData = {
  id?: string;
  surname: string;
  given_name: string;
  pinyin: string;
  gender: string;
  score: number;
  source?: string;
  notes?: string;
};

export async function getFavorites(): Promise<any> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/v1/favorites`);
    return await handleResponse(response);
  } catch (error) {
    console.error('Error getting favorites:', error);
    throw error;
  }
}

import type { FavoritesResponse } from '@/types/api/favorites';

// ... existing code ...

export async function saveFavorite(data: FavoriteData): Promise<FavoritesResponse> {
  try {
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
  } catch (error) {
    console.error('Error saving favorite:', error);
    throw error;
  }
}

export async function deleteFavorite(id: string): Promise<any> {
  try {
    if (!id || id.trim() === '') {
      throw new ValidationError('收藏ID不能为空');
    }
    
    const response = await fetchWithRetry(`${API_BASE_URL}/v1/favorites/${id}`, {
      method: 'DELETE',
    });
    return await handleResponse(response);
  } catch (error) {
    console.error('Error deleting favorite:', error);
    throw error;
  }
}

export async function checkFavorite(surname: string, givenName: string): Promise<any> {
  try {
    if (!surname || surname.trim() === '') {
      throw new ValidationError('姓氏不能为空');
    }
    if (!givenName || givenName.trim() === '') {
      throw new ValidationError('名字不能为空');
    }
    
    const response = await fetchWithRetry(`${API_BASE_URL}/v1/favorites/check?surname=${encodeURIComponent(surname)}&given_name=${encodeURIComponent(givenName)}`);
    return await handleResponse(response);
  } catch (error) {
    console.error('Error checking favorite:', error);
    throw error;
  }
}

export interface NameStat {
  name: string;
  count: number;
  rate: number;
  rank: number;
  province?: string;
  year_range?: string;
}

export async function getNameStats(name: string): Promise<any> {
  try {
    if (!name || name.trim() === '') {
      throw new ValidationError('名字不能为空');
    }
    if (name.length > 2) {
      throw new ValidationError('名字长度不能超过2个字符');
    }
    
    const response = await fetchWithRetry(`${API_BASE_URL}/v1/namestat/${encodeURIComponent(name)}`);
    return await handleResponse(response);
  } catch (error) {
    console.error('Error getting name stats:', error);
    throw error;
  }
}

export async function getProvinceStats(name: string, province: string = '北京'): Promise<any> {
  try {
    if (!name || name.trim() === '') {
      throw new ValidationError('名字不能为空');
    }
    if (name.length > 2) {
      throw new ValidationError('名字长度不能超过2个字符');
    }
    if (!province || province.trim() === '') {
      throw new ValidationError('省份不能为空');
    }
    
    const response = await fetchWithRetry(`${API_BASE_URL}/v1/namestat/${encodeURIComponent(name)}/province?province=${encodeURIComponent(province)}`);
    return await handleResponse(response);
  } catch (error) {
    console.error('Error getting province stats:', error);
    throw error;
  }
}

export async function getTopNames(limit: number = 20): Promise<any> {
  try {
    const safeLimit = safeParseInt(limit, 20);
    if (safeLimit < 1 || safeLimit > 100) {
      throw new ValidationError('查询数量必须在1-100之间');
    }
    
    const response = await fetchWithRetry(`${API_BASE_URL}/v1/namestat?limit=${safeLimit}`);
    return await handleResponse(response);
  } catch (error) {
    console.error('Error getting top names:', error);
    throw error;
  }
}

export async function getHuangli(year: number, month: number, day: number): Promise<any> {
  try {
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
  } catch (error) {
    console.error('Error getting huangli:', error);
    throw error;
  }
}

export async function getLunarCalendar(year: number, month: number, day: number, hour: number = 0, minute: number = 0): Promise<any> {
  try {
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
  } catch (error) {
    console.error('Error getting lunar calendar:', error);
    throw error;
  }
}
