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

export interface Name {
  id?: number;
  surname: string;
  generation?: string;
  given_name: string;
  full_name?: string;
  pinyin: string;
  meaning: string;
  wuxing: string;
  nayin?: string;
  strokes: number;
  gender: 'male' | 'female';
  score: number;
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
}

export interface GenerateResponse {
  success: boolean;
  message?: string;
  data: {
    bazi: BaziAnalysis;
    nayin: string;
    zodiac: string;
    hexagram: Hexagram;
    names: Name[];
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

export interface FavoriteData {
  id?: string;
  surname: string;
  given_name: string;
  pinyin: string;
  gender: 'male' | 'female';
  score: number;
  created_at?: string;
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
}

export interface APIResponse<T = any> {
  success: boolean;
  message?: string;
  data?: T;
}

export interface NameStat {
  count: number;
  rate: number;
  province?: string;
}
