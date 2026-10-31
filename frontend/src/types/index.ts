export interface GenerateRequest {
  surname: string;
  gender: 'male' | 'female';
  birth_year: number;
  birth_month: number;
  birth_day: number;
  birth_hour: number;
  birth_minute?: number;
  birth_location?: string;
  /** 显式出生地经度（东经为正）；>0 时优先于地点查表 */
  birth_longitude?: number;
  generation?: string;
  generation_position?: 'middle' | 'end';
  name_type?: 'double' | 'single';
  birth_type?: 'solar' | 'lunar';
  preferences?: string[];
  name_length?: number;
  source_classic?: string;
  // 筛选条件
  exclude_rare?: boolean;
  wuxing_match?: string[];
  min_strokes?: number;
  max_strokes?: number;
  include_poetry?: boolean;
  include_classic?: boolean;
  meaning_keywords?: string[];
  // 前端「按偏旁选字」点选的用字（结构化通道，不与关键词混用）
  selected_chars?: string[];
  pinyin_initial?: string;
  // 避讳长辈姓名列表（父系/母系直系长辈，建议往上两代）
  avoid_elder_names?: string[];
  // 人名频率过滤（来自 Chinese-Names-Corpus 语料统计）
  min_frequency_tier?: number; // 最小频率档位（1-5，0=不限）
  max_frequency_tier?: number; // 最大频率档位（1-5，0=不限）
  // 开启粤语评分：粤语音律 + 粤语谐音独立成「粤语分」展示，
  // 并以固定权重轻量并入综合分影响排序；关闭时综合分完全由普通话八维决定
  enable_cantonese?: boolean;
}

export interface Bazi {
  year: string;
  month: string;
  day: string;
  hour: string;
  year_ganzhi: string;
  month_ganzhi: string;
  day_ganzhi: string;
  hour_ganzhi: string;
}

export interface WuxingResult {
  jin: number;
  mu: number;
  shui: number;
  huo: number;
  tu: number;
}

export interface NayinInfo {
  ganzhi: string;
  nayin: string;
  nayin_wuxing: string;
}

export interface FourPillarNayin {
  year: NayinInfo;
  month: NayinInfo;
  day: NayinInfo;
  hour: NayinInfo;
}

export interface BaziAnalysis {
  bazi: Bazi;
  wuxing: WuxingResult;
  xiyongshen: string[];
  rishou: string;
  rishou_wuxing: string;
  nayin: string;
  /** 四柱纳音（年/月/日/时），兼容旧响应可能缺失 */
  four_nayin?: FourPillarNayin;
  day_master: string;
}

export interface Hexagram {
  id: number;
  name: string;
  number: number;
  symbol: string;
  upper_trigram: number;
  lower_trigram: number;
  gua_ci: string;
  xiang_ci: string;
  yao_ci: string[];
  interpretation: string;
}

export interface NameBase {
  surname: string;
  given_name: string;
  pinyin: string;
  gender: 'male' | 'female';
  score: number;
}

export interface Name extends NameBase {
  id?: number;
  generation?: string;
  full_name?: string;
  meaning: string;
  wuxing: string;
  nayin?: string;
  strokes: number;
  bazi_score?: number;
  huangli?: string;
  xiang?: string;
  reasons?: string[];
  wuxing_analysis?: string;
  bazi_score_detail?: string;
  yinyun?: string;
  poetry_source?: string;
  poetry_chapter?: string;
  poetry_sentence?: string;
  poetry_author?: string;       // 诗词作者
  poetry_dynasty?: string;      // 诗词朝代
  poetry_full_text?: string;    // 完整诗篇（竖线分隔各句）
  poetry_matched_chars?: string[]; // 实际命中的名字用字（前端据此高亮）
  // 统一评分体系：8 维评分
  total_score?: number;      // 综合总分（0-100）
  wuxing_score?: number;     // 五行匹配分
  yinyun_score?: number;     // 音韵律动分
  meaning_score?: number;    // 字义内涵分
  meaning_detail?: string;   // 字义内涵分析
  sancai_score?: number;     // 三才五格分
  zodiac_score?: number;     // 生肖适配分
  nayin_score?: number;      // 纳音评分
  novelty_score?: number;    // 新颖度评分
  bigram_score?: number;     // 诗词共现评分
  frequency_score?: number;  // 人名频率评分（来自 Chinese-Names-Corpus 语料统计）
  sancai_analysis?: string;  // 三才分析描述
  // 粤拼标注（姓 + 名各字粤拼，空格组合），与粤语评分开关无关，始终透传
  jyutping?: string;
  // 独立「粤语分」（0-100）；仅在开启粤语评分时非零
  cantonese_score?: number;
  // 粤语分维度明细（粤语音律/粤语谐音），仅在开启粤语评分时非空
  cantonese_score_detail?: Array<{
    name: string;
    score: number;
    detail: string;
  }>;
  // 确定性评分：各维度分数 + 依据文字（后端 ScoreDetail 透传）
  score_detail?: Array<{
    name: string;
    score: number;
    detail: string;
  }>;
}

export interface GenerateResponse {
  success: boolean;
  message?: string;
  data: {
    bazi: BaziAnalysis;
    nayin: string;
    zodiac: string;
    hexagram: Hexagram;
    hexagram_match?: Record<string, unknown>;
    ziwei?: ZiweiChart;
    names: Name[];
    suggestions?: string[];
    /** 生成会话标识：凭它调 POST /names/generate/explore 换一批（探索模式） */
    generation_id?: string;
  };
}

// 紫微斗数完整盘面（与后端 ZiweiChart JSON 一致）。
// 此前后端将盘面压缩为 9 字段摘要导致「算而不显」，现透传完整结构由前端渲染。
export interface ZiweiChart {
  year: number;
  month: number;
  day: number;
  hour: number;
  gender: string;
  nian_zhu: string;
  yue_zhu: string;
  ri_zhu: string;
  shi_zhu: string;
  ming_gong: string; // 命宫（干支）
  shen_gong: string; // 身宫（干支）
  five_elements: string; // 五行局，如「火六局」
  soul: string; // 命主
  body: string; // 身主
  // 大限：宫名 → { 起运年龄范围, 天干, 地支 }
  da_xian: Record<string, { range: [number, number]; heavenly_stem: string; earthly_branch: string }>;
  // 宫名 → 主星列表（同宫双星全部保留）
  zhu_xing: Record<string, string[]>;
  // 宫名 → 辅星列表
  fu_xing: Record<string, string[]>;
  // 宫名 → 杂曜列表
  sui_xing: Record<string, string[]>;
  // 生年四化：禄/权/科/忌 → 星名
  si_hua: Record<string, string>;
  // 主星亮度（庙/旺/得/利/平/不/陷），键为主星名
  brightness: Record<string, string>;
  analysis: string; // 排盘分析文本
}

// --- 异步生成任务（长任务体验） ---

export interface AsyncSubmitResponse {
  success: boolean;
  message?: string;
  data?: { task_id: string };
}

export interface TaskStatusResponse {
  success: boolean;
  message?: string;
  data?: {
    task_id: string;
    status: 'pending' | 'running' | 'success' | 'failed';
    stage: string;
    percent: number;
    error?: string;
    result?: GenerateResponse['data'];
  };
}

// --- 探索模式（换一批） ---

export interface ExploreResponse {
  success: boolean;
  message?: string;
  data?: {
    generation_id: string;
    names: Name[];
    remaining: number;
  };
}

// --- 测名 + 风险体检 ---

export interface EvaluateRequest {
  surname: string;
  given_name: string;
  gender: 'male' | 'female';
  birth_year: number;
  birth_month: number;
  birth_day: number;
  birth_hour: number;
  birth_minute?: number;
  birth_location?: string;
  birth_longitude?: number;
}

/** 风险体检单项（谐音/生僻字/多音字/负面联想/门禁字/撞名热度/户籍友好度/敏感词） */
export interface RiskItem {
  category: string;
  level: 'pass' | 'warn' | 'fail';
  detail: string;
}

export interface EvaluateResponse {
  success: boolean;
  message?: string;
  data?: {
    full_name: string;
    /** 与 /generate 的 names[] 元素同构，可直接复用 Name 渲染组件 */
    name: Name;
    bazi: BaziAnalysis;
    zodiac: string;
    risks: RiskItem[];
    risk_level: 'pass' | 'warn' | 'fail';
  };
}

export interface HistoryRecord {
  id: string;
  surname: string;
  gender: 'male' | 'female';
  birth_date: string;
  birth_time: string;
  birth_location: string;
  generation?: string;
  results: GenerateResponse['data'] | string;
  created_at: string;
}

export interface FavoriteData extends NameBase {
  id?: string;
  created_at?: string;
  source?: string;
  notes?: string;
}

export interface FormData {
  surname: string;
  gender: 'male' | 'female';
  birthYear: number;
  birthMonth: number;
  birthDay: number;
  birthHour: number;
  birthMinute: number;
  birthLocation: string;
  generation: string;
  generationPosition: 'middle' | 'end';
  nameType: 'double' | 'single';
  birthType: 'solar' | 'lunar';
  preferences: string[];
  nameLength: number;
  sourceClassic: string;
  // 避讳长辈姓名（逗号分隔的字符串，提交时转为数组）
  avoidElderNames: string;
  // 人名频率过滤
  minFrequencyTier: number;
  maxFrequencyTier: number;
  // 粤语评分开关：开启后结果页展示粤拼标注与独立粤语分（粤语音律 + 粤语谐音）
  enableCantonese: boolean;
}

export interface APIResponse<T = unknown> {
  success: boolean;
  message?: string;
  data?: T;
}

// 偏旁选字类型
export interface StandardCharGroup {
  radical: string;
  name: string;
  meaning: string;
  chars: string[];
}

export interface CuratedName {
  name: string;
  pinyin: string;
  gender: string;
  source: string;
  meaning: string;
  wuxing: string;
  yinyun_score: number;
  styles?: string[];
  tags?: string[];
}

export interface NameStat {
  name: string;
  count: number;
  rate: number;
  rank: number;
}
