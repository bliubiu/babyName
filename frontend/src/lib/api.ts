// 桌面端 API 层：全部经 Wails TS 绑定直连 Go 后端（internal/desktop）。
//
// 迁移说明（docs/47 §4-S2）：
// - 原实现走 HTTP fetch（/api/v1 信封），现改为绑定调用；
// - 绑定方法返回的 error 是 Go 端整理好的中文文案（可行动），统一包成 ApiError；
// - 为最小化页面改动，本层继续返回与 HTTP 时代相同的信封形状（{success, message?, data}），
//   页面代码零改动；
// - AbortSignal 语义保留：取消在轮询边界/循环入口生效（绑定调用本身不可中断）。
import type {
  GenerateResponse, APIResponse, FavoriteData, GenerateRequest, StandardCharGroup,
  NameStat, TaskStatusResponse, ExploreResponse,
  EvaluateRequest, EvaluateResponse,
} from '@/types';
import type { HistoryResponse } from '@/types/api/history';
import type { FavoritesResponse } from '@/types/api/favorites';
import {
  EvaluateName, ExploreNames, SubmitGenerateTask, GetGenerateTask,
  GetHistory, SaveHistory, DeleteHistory, GetFavorites, SaveFavorite, DeleteFavorite,
  GenerateReportPDF, CreateShare, GetShare, GetHuangli, GetLunarCalendar,
  GetLunarMonthCalendar, GetCharGroups, GetNameStats, GetTopNames,
  GetMaterialStates, ImportMaterials,
} from '../../bindings/name/internal/desktop/app';
import type {
  MaterialStatesResponse, MaterialImportResponse,
} from '@/types/api/materials';

// 错误类型定义（保持原有导出，供调用方 instanceof 判断）
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

/** 绑定层抛出的 Error（中文文案）统一包成 ApiError，保留原 message */
function wrapBindingError(err: unknown, fallback: string): ApiError {
  if (err instanceof ApiError) return err;
  const message = err instanceof Error && err.message ? err.message : fallback;
  return new ApiError(message);
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

// --- 异步生成任务（长任务体验） ---

/** 提交异步生成任务，立即返回 task_id（不阻塞等待引擎完成） */
export async function generateNamesAsync(data: GenerateRequest, signal?: AbortSignal): Promise<string> {
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

  // 调用方已取消：不再发起提交
  if (signal?.aborted) throw createAbortError();

  const result = await SubmitGenerateTask(processedData as never)
    .catch((err: unknown) => { throw wrapBindingError(err, '任务提交失败，请稍后重试'); });
  if (!result?.task_id) {
    throw new ServerError('任务提交失败，请稍后重试');
  }
  return result.task_id;
}

/** 查询异步任务状态；成功时 result 为完整生成结果（取走即弃） */
export async function getTaskStatus(taskId: string, signal?: AbortSignal): Promise<TaskStatusResponse['data']> {
  if (!taskId || taskId.trim() === '') {
    throw new ValidationError('任务ID不能为空');
  }
  if (signal?.aborted) throw createAbortError();
  const status = await GetGenerateTask(taskId)
    .catch((err: unknown) => { throw wrapBindingError(err, '任务查询失败，请稍后重试'); });
  if (!status) {
    throw new ServerError('任务不存在或已过期');
  }
  return {
    task_id: status.task_id,
    status: status.status as 'pending' | 'running' | 'success' | 'failed',
    stage: status.stage,
    percent: status.percent,
    error: status.error,
    result: status.result as unknown as TaskStatusResponse['data'] extends infer T ? T : never,
  } as TaskStatusResponse['data'];
}

/**
 * 提交异步生成并轮询直到完成，返回完整生成结果。
 * onProgress 每 500ms 回调一次（stage 阶段名 / percent 百分比）。
 *
 * 传入 signal 可主动取消整条链（提交 + 轮询 + 退避等待），docs/29 B17：
 * 取消时抛出 name === 'AbortError' 的错误，调用方应静默处理、不弹错误提示。
 * 注意：绑定调用本身不可中断，取消在轮询边界生效（最多延迟一个 pollIntervalMs）。
 */
export async function generateNamesWithProgress(
  data: GenerateRequest,
  onProgress?: (stage: string, percent: number) => void,
  pollIntervalMs: number = 500,
  signal?: AbortSignal
): Promise<GenerateResponse['data']> {
  const taskId = await generateNamesAsync(data, signal);
  const deadline = Date.now() + 5 * 60 * 1000;

  for (;;) {
    // 循环入口先看取消：避免取消后再多打一次请求
    if (signal?.aborted) throw createAbortError();

    const status = await getTaskStatus(taskId, signal);
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
    await sleep(pollIntervalMs, signal);
  }
}

// --- 探索模式（换一批） ---

/** 从指定生成会话换一批：与上次结果零交集的随机候选 */
export async function exploreNames(generationId: string, count: number = 10): Promise<ExploreResponse['data']> {
  if (!generationId || generationId.trim() === '') {
    throw new ValidationError('生成会话ID不能为空');
  }
  const result = await ExploreNames({ generation_id: generationId, count })
    .catch((err: unknown) => { throw wrapBindingError(err, '换一批失败，请稍后重试'); });
  if (!result) {
    throw new ServerError('换一批失败，请稍后重试');
  }
  return result as unknown as ExploreResponse['data'];
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

  const result = await EvaluateName({
    ...data,
    birth_year: safeParseInt(data.birth_year),
    birth_month: safeParseInt(data.birth_month),
    birth_day: safeParseInt(data.birth_day),
    birth_hour: safeParseInt(data.birth_hour),
    birth_minute: safeParseInt(data.birth_minute ?? 0),
  } as never)
    .catch((err: unknown) => { throw wrapBindingError(err, '测名失败，请稍后重试'); });
  if (!result) {
    throw new ServerError('测名失败，请稍后重试');
  }
  return result as unknown as EvaluateResponse['data'];
}

export async function getHistory(page: number = 1, limit: number = 100): Promise<HistoryResponse> {
  const result = await GetHistory(page, limit)
    .catch((err: unknown) => { throw wrapBindingError(err, '历史记录读取失败，请稍后重试'); });
  // 桌面绑定直接返回分页结构 {records, total, page, limit}，归一化为旧信封数组
  return { success: true, data: (result?.records ?? []) as unknown as HistoryResponse['data'] };
}

export async function saveHistory(data: Record<string, unknown>): Promise<HistoryResponse> {
  await SaveHistory(data as never)
    .catch((err: unknown) => { throw wrapBindingError(err, '历史记录保存失败，请稍后重试'); });
  return { success: true, data: [] };
}

export async function deleteHistory(id: string): Promise<HistoryResponse> {
  if (!id || id.trim() === '') {
    throw new ValidationError('历史记录ID不能为空');
  }
  await DeleteHistory(id)
    .catch((err: unknown) => { throw wrapBindingError(err, '历史记录删除失败，请稍后重试'); });
  return { success: true, data: [] };
}

export async function getFavorites(): Promise<FavoritesResponse> {
  // 桌面单用户固定会话，一次拉满（上限 100）后归一化为数组
  const result = await GetFavorites(1, 100)
    .catch((err: unknown) => { throw wrapBindingError(err, '收藏读取失败，请稍后重试'); });
  return { success: true, data: (result?.records ?? []) as unknown as FavoriteData[] };
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

  await SaveFavorite(data as never)
    .catch((err: unknown) => { throw wrapBindingError(err, '收藏保存失败，请稍后重试'); });
  return { success: true, data: [] };
}

export async function deleteFavorite(id: string): Promise<APIResponse> {
  if (!id || id.trim() === '') {
    throw new ValidationError('收藏ID不能为空');
  }
  await DeleteFavorite(id)
    .catch((err: unknown) => { throw wrapBindingError(err, '收藏删除失败，请稍后重试'); });
  return { success: true };
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

  const huangli = await GetHuangli(safeYear, safeMonth, safeDay)
    .catch((err: unknown) => { throw wrapBindingError(err, '黄历查询失败，请稍后重试'); });
  return { success: true, data: huangli };
}

export interface LunarMonthDay {
  day: number;
  solarTerm: string;
  lunarFestival: string;
}

export interface LunarMonthCalendar {
  year: number;
  month: number;
  days: LunarMonthDay[];
}

/**
 * 获取某公历年月的逐日节气与农历节日标注（用于万年历网格展示）。
 * 返回 data.days：[{ day, solarTerm, lunarFestival }]，非节气/非节日字段为空串。
 */
export async function getLunarMonthCalendar(year: number, month: number): Promise<APIResponse<LunarMonthCalendar>> {
  const safeYear = safeParseInt(year);
  const safeMonth = safeParseInt(month);

  if (safeYear < 1900 || safeYear > 2100) {
    throw new ValidationError('年份必须在1900-2100之间');
  }
  if (safeMonth < 1 || safeMonth > 12) {
    throw new ValidationError('月份必须在1-12之间');
  }

  const calendar = await GetLunarMonthCalendar(safeYear, safeMonth)
    .catch((err: unknown) => { throw wrapBindingError(err, '农历月历查询失败，请稍后重试'); });
  return { success: true, data: calendar as unknown as LunarMonthCalendar };
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

  const lunar = await GetLunarCalendar(safeYear, safeMonth, safeDay, safeHour, safeMinute)
    .catch((err: unknown) => { throw wrapBindingError(err, '农历查询失败，请稍后重试'); });
  return { success: true, data: lunar };
}

// --- 偏旁选字 API ---

export async function getCharGroups(): Promise<StandardCharGroup[]> {
  const groups = await GetCharGroups()
    .catch((err: unknown) => { throw wrapBindingError(err, '偏旁分组读取失败，请稍后重试'); });
  return (groups ?? []) as unknown as StandardCharGroup[];
}

// --- 名字统计 API ---

export async function getNameStats(name: string): Promise<NameStat | null> {
  if (!name || name.trim() === '') {
    throw new ValidationError('名字不能为空');
  }
  const stat = await GetNameStats(name)
    .catch((err: unknown) => { throw wrapBindingError(err, '名字统计读取失败，请稍后重试'); });
  return (stat ?? null) as unknown as NameStat | null;
}

export async function getTopNames(limit: number = 20): Promise<NameStat[]> {
  const safeLimit = safeParseInt(limit, 20);
  if (safeLimit < 1 || safeLimit > 100) {
    throw new ValidationError('查询数量必须在1-100之间');
  }
  // kind 传 all（与旧 HTTP 全量榜单口径一致）
  const list = await GetTopNames(safeLimit, 'all')
    .catch((err: unknown) => { throw wrapBindingError(err, '名字榜读取失败，请稍后重试'); });
  return (list ?? []) as unknown as NameStat[];
}

// --- 结果页动作区：导出 PDF / 分享快照 ---

/** /report/pdf 请求体：字段名与后端 reportModel.parseReportModel 对齐 */
export interface ReportPayload {
  surname?: string;
  gender?: string;
  birth_date?: string;
  birth_time?: string;
  bazi?: string;
  wuxing?: string;
  xiyongshen?: string;
  /** 日主强弱 */
  rishou?: string;
  /** 纳音 */
  nayin?: string;
  /** 生肖 */
  zodiac?: string;
  /** 命理详批文本（前端 generateDetailAnalysis 生成） */
  analysis?: string[];
  /** 五行分布数值，用于条形图；键为单字 金木水火土 */
  wuxing_dist?: Record<string, number>;
  /** 卦象摘要：本卦/上下卦/卦辞/象辞/彖传/卦德/解读 */
  hexagram?: {
    number?: number;
    name?: string;
    symbol?: string;
    upper?: string;
    lower?: string;
    gua_ci?: string;
    xiang_ci?: string;
    tuan_ci?: string;
    gua_de?: string;
    interpretation?: string;
  };
  /** 紫微斗数摘要：命宫/身宫/五行局/四柱/四化/十二宫主星/排盘分析 */
  ziwei?: {
    ming_gong?: string;
    shen_gong?: string;
    five_elements?: string;
    soul?: string;
    body?: string;
    nian_zhu?: string;
    yue_zhu?: string;
    ri_zhu?: string;
    shi_zhu?: string;
    si_hua?: Record<string, string>;
    zhu_xing?: Record<string, string[]>;
    brightness?: Record<string, string>;
    analysis?: string;
  };
  /** 导出详略：full=含命理详批与分布图，summary=仅基本信息+名字表 */
  options?: { detail?: 'full' | 'summary' };
  names?: Array<{
    full_name?: string;
    pinyin?: string;
    wuxing?: string;
    score?: number | string;
    meaning?: string;
    /** 各维评分（如 五行/音律/字义…），PDF 详批模式展示 */
    scores?: Array<{ name: string; score: number }>;
  }>;
}

/**
 * 导出 PDF：返回二进制 Blob，由调用方触发浏览器下载。
 *
 * 绑定契约两处坑（HTTP → Wails 迁移遗留）：
 * - 参数直接透传，不再由 handler 解包 {data:...} 信封，必须传扁平对象；
 * - Go 方法返回 []byte，运行时经 json.Marshal 编码为 base64 字符串，
 *   而 typescriptify 生成的声明写成 Array<number>。`new Uint8Array(字符串)`
 *   会静默得到空数组并报「PDF 内容为空」，故这里兼容两种返回形态。
 */
export async function exportReportPDF(data: ReportPayload): Promise<Blob> {
  const raw: unknown = await GenerateReportPDF(data as never)
    .catch((err: unknown) => { throw wrapBindingError(err, '导出 PDF 失败，请稍后重试'); });
  let bytes: Uint8Array;
  if (typeof raw === 'string') {
    const binary = atob(raw);
    bytes = Uint8Array.from(binary, (ch) => ch.charCodeAt(0));
  } else {
    bytes = new Uint8Array(Array.isArray(raw) ? (raw as number[]) : []);
  }
  // TS 5.7+ 的 Uint8Array<ArrayBufferLike> 与 DOM BlobPart 不直接兼容，
  // 此处字节一定来自 ArrayBuffer，显式收窄后交给 Blob。
  const blob = new Blob([bytes as unknown as BlobPart], { type: 'application/pdf' });
  if (!blob.size) {
    throw new ServerError('PDF 内容为空，请稍后重试');
  }
  return blob;
}

/** 分享快照：只含展示结论，不含出生地/长辈姓名等输入信息（后端仍有字段白名单）。 */
export interface ShareSnapshot {
  title?: string;
  surname?: string;
  gender?: string;
  rishou?: string;
  xiyongshen?: string[];
  zodiac?: string;
  hexagram_name?: string;
  five_elements?: string;
  total?: number;
  generated_at?: string;
  names: Array<{
    full_name: string;
    pinyin?: string;
    wuxing?: string;
    score?: number;
    meaning?: string;
  }>;
}

export interface ShareCreateResult {
  id: string;
  url: string;
}

export async function createShareSnapshot(snapshot: ShareSnapshot): Promise<ShareCreateResult> {
  if (!Array.isArray(snapshot.names) || snapshot.names.length === 0) {
    throw new ValidationError('没有可分享的名字');
  }
  // 绑定层直接透传快照对象：若套 {data:...} 信封，后端字段白名单会把
  // 顶层唯一的 data 键剔除，返回「分享内容不合法: 没有可分享的内容」。
  const result = await CreateShare(snapshot as never)
    .catch((err: unknown) => { throw wrapBindingError(err, '创建分享失败，请稍后重试'); });
  if (!result?.id) {
    throw new ServerError('创建分享失败，请稍后重试');
  }
  return { id: result.id, url: result.url };
}

export async function getShareSnapshot(id: string): Promise<ShareSnapshot> {
  if (!/^[a-f0-9]{32}$/.test(id)) {
    throw new NotFoundError('分享不存在或已失效');
  }
  const raw = await GetShare(id)
    .catch((err: unknown) => { throw wrapBindingError(err, '分享不存在或已失效'); });
  if (!raw) {
    throw new ServerError('分享内容格式错误');
  }
  // GetShare 返回 json.RawMessage（字符串），需反序列化为快照对象
  let data: ShareSnapshot;
  try {
    data = typeof raw === 'string' ? JSON.parse(raw) : (raw as unknown as ShareSnapshot);
  } catch {
    throw new ServerError('分享内容格式错误');
  }
  if (!data || !Array.isArray(data.names)) {
    throw new ServerError('分享内容格式错误');
  }
  return data;
}

// --- 物料管理（桌面端专属：状态查看 + 手动重导） ---

export async function getMaterialStates(): Promise<MaterialStatesResponse> {
  const raw = await GetMaterialStates()
    .catch((err: unknown) => { throw wrapBindingError(err, '物料状态读取失败，请稍后重试'); });
  const list = Array.isArray(raw) ? raw : [];
  return { success: true, data: list };
}

export async function importMaterials(): Promise<MaterialImportResponse> {
  const report = await ImportMaterials()
    .catch((err: unknown) => { throw wrapBindingError(err, '物料导入失败，请稍后重试'); });
  if (!report || !Array.isArray(report.items)) {
    throw new ServerError('物料导入报告格式错误');
  }
  return { success: true, data: report };
}

// --- 取消语义（原 fetch 版保留） ---

/** 构造可识别的取消错误，供调用方与真实失败区分 */
function createAbortError(): Error {
  if (typeof DOMException !== 'undefined') {
    return new DOMException('请求已取消', 'AbortError');
  }
  const err = new Error('请求已取消');
  err.name = 'AbortError';
  return err;
}

/** 可被外部信号打断的 sleep；取消时抛出 AbortError */
function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(createAbortError());
      return;
    }
    const timer = setTimeout(() => {
      signal?.removeEventListener('abort', onAbort);
      resolve();
    }, ms);
    const onAbort = () => {
      clearTimeout(timer);
      reject(createAbortError());
    };
    signal?.addEventListener('abort', onAbort, { once: true });
  });
}
