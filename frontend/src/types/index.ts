export interface GenerateRequest {
  surname: string;
  gender: 'male' | 'female';
  birth_year: number;
  birth_month: number;
  birth_day: number;
  birth_hour: number;
  birth_minute?: number;
  birth_location?: string;
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
  id: number;
  surname: string;
  given_name: string;
  pinyin: string;
  meaning: string;
  wuxing: string;
  strokes: number;
  gender: string;
  score: number;
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
  gender: string;
  birth_date: string;
  birth_time: string;
  birth_location: string;
  results: GenerateResponse['data'] | string;
  created_at: string;
}
