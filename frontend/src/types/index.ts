export interface GenerateRequest {
  surname: string;
  gender: 'male' | 'female';
  birth_year: number;
  birth_month: number;
  birth_day: number;
  birth_hour: number;
  birth_minute?: number;
  birth_location?: string;
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
  pinyin_initial?: string;
  // 避讳长辈姓名列表（父系/母系直系长辈，建议往上两代）
  avoid_elder_names?: string[];
  // 人名频率过滤（来自 Chinese-Names-Corpus 语料统计）
  min_frequency_tier?: number; // 最小频率档位（1-5，0=不限）
  max_frequency_tier?: number; // 最大频率档位（1-5，0=不限）
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

export interface BaziAnalysis {
  bazi: Bazi;
  wuxing: WuxingResult;
  xiyongshen: string[];
  rishou: string;
  rishou_wuxing: string;
  nayin: string;
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
  // 统一评分体系：8 维评分
  total_score?: number;      // 综合总分（0-100）
  wuxing_score?: number;     // 五行匹配分
  yinyun_score?: number;     // 音韵律动分
  meaning_score?: number;    // 字义内涵分
  sancai_score?: number;     // 三才五格分
  zodiac_score?: number;     // 生肖适配分
  nayin_score?: number;      // 纳音评分
  novelty_score?: number;    // 新颖度评分
  bigram_score?: number;     // 诗词共现评分
  frequency_score?: number;  // 人名频率评分（来自 Chinese-Names-Corpus 语料统计）
  sancai_analysis?: string;  // 三才分析描述
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
    ziwei?: Record<string, unknown>;
    names: Name[];
    suggestions?: string[];
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
  count: number;
  rate: number;
  province?: string;
}
