export namespace bazi {
	
	export class Bazi {
	    year: string;
	    month: string;
	    day: string;
	    hour: string;
	    year_ganzhi: string;
	    month_ganzhi: string;
	    day_ganzhi: string;
	    hour_ganzhi: string;
	    is_leap_month: boolean;
	    lunar_month: number;
	
	    static createFrom(source: any = {}) {
	        return new Bazi(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.month = source["month"];
	        this.day = source["day"];
	        this.hour = source["hour"];
	        this.year_ganzhi = source["year_ganzhi"];
	        this.month_ganzhi = source["month_ganzhi"];
	        this.day_ganzhi = source["day_ganzhi"];
	        this.hour_ganzhi = source["hour_ganzhi"];
	        this.is_leap_month = source["is_leap_month"];
	        this.lunar_month = source["lunar_month"];
	    }
	}
	export class StrengthInfo {
	    count: number;
	    strength: string;
	    score: number;
	    advice: string;
	
	    static createFrom(source: any = {}) {
	        return new StrengthInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.strength = source["strength"];
	        this.score = source["score"];
	        this.advice = source["advice"];
	    }
	}
	export class WuxingStrengthResult {
	    jin: StrengthInfo;
	    mu: StrengthInfo;
	    shui: StrengthInfo;
	    huo: StrengthInfo;
	    tu: StrengthInfo;
	
	    static createFrom(source: any = {}) {
	        return new WuxingStrengthResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jin = this.convertValues(source["jin"], StrengthInfo);
	        this.mu = this.convertValues(source["mu"], StrengthInfo);
	        this.shui = this.convertValues(source["shui"], StrengthInfo);
	        this.huo = this.convertValues(source["huo"], StrengthInfo);
	        this.tu = this.convertValues(source["tu"], StrengthInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NayinInfo {
	    ganzhi: string;
	    nayin: string;
	    nayin_wuxing: string;
	
	    static createFrom(source: any = {}) {
	        return new NayinInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ganzhi = source["ganzhi"];
	        this.nayin = source["nayin"];
	        this.nayin_wuxing = source["nayin_wuxing"];
	    }
	}
	export class FourPillarNayin {
	    year: NayinInfo;
	    month: NayinInfo;
	    day: NayinInfo;
	    hour: NayinInfo;
	
	    static createFrom(source: any = {}) {
	        return new FourPillarNayin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = this.convertValues(source["year"], NayinInfo);
	        this.month = this.convertValues(source["month"], NayinInfo);
	        this.day = this.convertValues(source["day"], NayinInfo);
	        this.hour = this.convertValues(source["hour"], NayinInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WuxingResult {
	    jin: number;
	    mu: number;
	    shui: number;
	    huo: number;
	    tu: number;
	
	    static createFrom(source: any = {}) {
	        return new WuxingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jin = source["jin"];
	        this.mu = source["mu"];
	        this.shui = source["shui"];
	        this.huo = source["huo"];
	        this.tu = source["tu"];
	    }
	}
	export class BaziAnalysis {
	    bazi: Bazi;
	    wuxing: WuxingResult;
	    xiyongshen: string[];
	    yiyongshen: string[];
	    tiaohou_shen: string[];
	    rishou: string;
	    rishou_wuxing: string;
	    nayin: string;
	    four_nayin?: FourPillarNayin;
	    day_master: string;
	    day_master_strength: string;
	    solar_term: string;
	    season: string;
	    wuxing_strength?: WuxingStrengthResult;
	    best_hexagram: string;
	    hexagram_wuxing: string;
	    bazi_pattern: string;
	    yongshen_score: number;
	    dayan_hexagram: string;
	    dayan_interpretation: string;
	    is_leap_month: boolean;
	    leap_month_advice?: string;
	
	    static createFrom(source: any = {}) {
	        return new BaziAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bazi = this.convertValues(source["bazi"], Bazi);
	        this.wuxing = this.convertValues(source["wuxing"], WuxingResult);
	        this.xiyongshen = source["xiyongshen"];
	        this.yiyongshen = source["yiyongshen"];
	        this.tiaohou_shen = source["tiaohou_shen"];
	        this.rishou = source["rishou"];
	        this.rishou_wuxing = source["rishou_wuxing"];
	        this.nayin = source["nayin"];
	        this.four_nayin = this.convertValues(source["four_nayin"], FourPillarNayin);
	        this.day_master = source["day_master"];
	        this.day_master_strength = source["day_master_strength"];
	        this.solar_term = source["solar_term"];
	        this.season = source["season"];
	        this.wuxing_strength = this.convertValues(source["wuxing_strength"], WuxingStrengthResult);
	        this.best_hexagram = source["best_hexagram"];
	        this.hexagram_wuxing = source["hexagram_wuxing"];
	        this.bazi_pattern = source["bazi_pattern"];
	        this.yongshen_score = source["yongshen_score"];
	        this.dayan_hexagram = source["dayan_hexagram"];
	        this.dayan_interpretation = source["dayan_interpretation"];
	        this.is_leap_month = source["is_leap_month"];
	        this.leap_month_advice = source["leap_month_advice"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class HourYiJi {
	    timeRange: string;
	    tianShen: string;
	    heiHuangDao: string;
	    jiXiong: string;
	    yi: string[];
	    ji: string[];
	
	    static createFrom(source: any = {}) {
	        return new HourYiJi(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timeRange = source["timeRange"];
	        this.tianShen = source["tianShen"];
	        this.heiHuangDao = source["heiHuangDao"];
	        this.jiXiong = source["jiXiong"];
	        this.yi = source["yi"];
	        this.ji = source["ji"];
	    }
	}
	export class Huangli {
	    date: string;
	    year: string;
	    month: string;
	    day: string;
	    yi: string[];
	    ji: string[];
	    jiShen: string[];
	    xiongSha: string[];
	    baiCang: string;
	    pengZhu: string;
	    fu: string;
	    jieShen: string;
	    chong: string;
	    sha: string;
	    zhangSong: string;
	    yearWuxing: string;
	    monthWuxing: string;
	    dayWuxing: string;
	    liuYao: string;
	    zhiShen: string;
	    taiShen: string;
	    jianChu: string;
	    xingXiu: string;
	    xiShen: string;
	    fuShen: string;
	    caiShen: string;
	    yangGui: string;
	    yinGui: string;
	    hours: HourYiJi[];
	
	    static createFrom(source: any = {}) {
	        return new Huangli(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.year = source["year"];
	        this.month = source["month"];
	        this.day = source["day"];
	        this.yi = source["yi"];
	        this.ji = source["ji"];
	        this.jiShen = source["jiShen"];
	        this.xiongSha = source["xiongSha"];
	        this.baiCang = source["baiCang"];
	        this.pengZhu = source["pengZhu"];
	        this.fu = source["fu"];
	        this.jieShen = source["jieShen"];
	        this.chong = source["chong"];
	        this.sha = source["sha"];
	        this.zhangSong = source["zhangSong"];
	        this.yearWuxing = source["yearWuxing"];
	        this.monthWuxing = source["monthWuxing"];
	        this.dayWuxing = source["dayWuxing"];
	        this.liuYao = source["liuYao"];
	        this.zhiShen = source["zhiShen"];
	        this.taiShen = source["taiShen"];
	        this.jianChu = source["jianChu"];
	        this.xingXiu = source["xingXiu"];
	        this.xiShen = source["xiShen"];
	        this.fuShen = source["fuShen"];
	        this.caiShen = source["caiShen"];
	        this.yangGui = source["yangGui"];
	        this.yinGui = source["yinGui"];
	        this.hours = this.convertValues(source["hours"], HourYiJi);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LunarCalendar {
	    year: number;
	    month: number;
	    day: number;
	    isLeap: boolean;
	    lunarMonthNumber: number;
	    lunarDayNumber: number;
	    yearGan: string;
	    yearZhi: string;
	    monthGan: string;
	    monthZhi: string;
	    dayGan: string;
	    dayZhi: string;
	    hourGan: string;
	    hourZhi: string;
	    nayin: string;
	    yearNayin: string;
	    monthNayin: string;
	    dayNayin: string;
	    lunarYearName: string;
	    lunarMonthName: string;
	    lunarDayName: string;
	    dateStr: string;
	    lunarDate: string;
	    lunarMonth: string;
	    solarTerm: string;
	
	    static createFrom(source: any = {}) {
	        return new LunarCalendar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.month = source["month"];
	        this.day = source["day"];
	        this.isLeap = source["isLeap"];
	        this.lunarMonthNumber = source["lunarMonthNumber"];
	        this.lunarDayNumber = source["lunarDayNumber"];
	        this.yearGan = source["yearGan"];
	        this.yearZhi = source["yearZhi"];
	        this.monthGan = source["monthGan"];
	        this.monthZhi = source["monthZhi"];
	        this.dayGan = source["dayGan"];
	        this.dayZhi = source["dayZhi"];
	        this.hourGan = source["hourGan"];
	        this.hourZhi = source["hourZhi"];
	        this.nayin = source["nayin"];
	        this.yearNayin = source["yearNayin"];
	        this.monthNayin = source["monthNayin"];
	        this.dayNayin = source["dayNayin"];
	        this.lunarYearName = source["lunarYearName"];
	        this.lunarMonthName = source["lunarMonthName"];
	        this.lunarDayName = source["lunarDayName"];
	        this.dateStr = source["dateStr"];
	        this.lunarDate = source["lunarDate"];
	        this.lunarMonth = source["lunarMonth"];
	        this.solarTerm = source["solarTerm"];
	    }
	}
	export class LunarMonthDay {
	    day: number;
	    solarTerm: string;
	    lunarFestival: string;
	
	    static createFrom(source: any = {}) {
	        return new LunarMonthDay(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.day = source["day"];
	        this.solarTerm = source["solarTerm"];
	        this.lunarFestival = source["lunarFestival"];
	    }
	}
	
	
	

}

export namespace desktop {
	
	export class CharGroup {
	    radical: string;
	    name: string;
	    meaning: string;
	    chars: string[];
	
	    static createFrom(source: any = {}) {
	        return new CharGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.radical = source["radical"];
	        this.name = source["name"];
	        this.meaning = source["meaning"];
	        this.chars = source["chars"];
	    }
	}
	export class FavoritePage {
	    records: services.FavoriteRecord[];
	    total: number;
	    page: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new FavoritePage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.records = this.convertValues(source["records"], services.FavoriteRecord);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.limit = source["limit"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HistoryPage {
	    records: services.HistoryRecord[];
	    total: number;
	    page: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new HistoryPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.records = this.convertValues(source["records"], services.HistoryRecord);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.limit = source["limit"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LunarMonthCalendar {
	    year: number;
	    month: number;
	    days: bazi.LunarMonthDay[];
	
	    static createFrom(source: any = {}) {
	        return new LunarMonthCalendar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.month = source["month"];
	        this.days = this.convertValues(source["days"], bazi.LunarMonthDay);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Message {
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	    }
	}
	export class RecordID {
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new RecordID(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	    }
	}
	export class ShareResult {
	    id: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new ShareResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	    }
	}
	export class SubmitTaskResult {
	    task_id: string;
	
	    static createFrom(source: any = {}) {
	        return new SubmitTaskResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.task_id = source["task_id"];
	    }
	}
	export class TaskResult {
	    task_id: string;
	    status: string;
	    stage: string;
	    percent: number;
	    error?: string;
	    result?: services.GenerateResponse;
	
	    static createFrom(source: any = {}) {
	        return new TaskResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.task_id = source["task_id"];
	        this.status = source["status"];
	        this.stage = source["stage"];
	        this.percent = source["percent"];
	        this.error = source["error"];
	        this.result = this.convertValues(source["result"], services.GenerateResponse);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace fate {
	
	export class RiskItem {
	    category: string;
	    level: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new RiskItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.level = source["level"];
	        this.detail = source["detail"];
	    }
	}

}

export namespace name {
	
	export class ScoreDetailItem {
	    name: string;
	    score: number;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new ScoreDetailItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.score = source["score"];
	        this.detail = source["detail"];
	    }
	}
	export class Name {
	    id: number;
	    surname: string;
	    generation: string;
	    given_name: string;
	    full_name: string;
	    pinyin: string;
	    meaning: string;
	    wuxing: string;
	    nayin: string;
	    strokes: number;
	    gender: string;
	    bazi_score: number;
	    huangli: string;
	    xiang: string;
	    poetry_source: string;
	    poetry_chapter: string;
	    poetry_sentence: string;
	    poetry_author?: string;
	    poetry_dynasty?: string;
	    poetry_full_text?: string;
	    poetry_matched_chars?: string[];
	    wuxing_analysis: string;
	    meaning_detail: string;
	    bazi_score_detail: string;
	    yinyun: string;
	    reasons: string[];
	    total_score: number;
	    wuxing_score: number;
	    yinyun_score: number;
	    meaning_score: number;
	    sancai_score: number;
	    zodiac_score: number;
	    nayin_score: number;
	    novelty_score: number;
	    bigram_score: number;
	    frequency_score: number;
	    sancai_analysis: string;
	    hexagram: string;
	    hexagram_meaning: string;
	    hexagram_detail?: yijing.PlumBlossomResult;
	    tiyong?: yijing.TiYongAnalysis;
	    score_detail?: ScoreDetailItem[];
	    jyutping?: string;
	    cantonese_score?: number;
	    cantonese_score_detail?: ScoreDetailItem[];
	    risks?: fate.RiskItem[];
	
	    static createFrom(source: any = {}) {
	        return new Name(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.surname = source["surname"];
	        this.generation = source["generation"];
	        this.given_name = source["given_name"];
	        this.full_name = source["full_name"];
	        this.pinyin = source["pinyin"];
	        this.meaning = source["meaning"];
	        this.wuxing = source["wuxing"];
	        this.nayin = source["nayin"];
	        this.strokes = source["strokes"];
	        this.gender = source["gender"];
	        this.bazi_score = source["bazi_score"];
	        this.huangli = source["huangli"];
	        this.xiang = source["xiang"];
	        this.poetry_source = source["poetry_source"];
	        this.poetry_chapter = source["poetry_chapter"];
	        this.poetry_sentence = source["poetry_sentence"];
	        this.poetry_author = source["poetry_author"];
	        this.poetry_dynasty = source["poetry_dynasty"];
	        this.poetry_full_text = source["poetry_full_text"];
	        this.poetry_matched_chars = source["poetry_matched_chars"];
	        this.wuxing_analysis = source["wuxing_analysis"];
	        this.meaning_detail = source["meaning_detail"];
	        this.bazi_score_detail = source["bazi_score_detail"];
	        this.yinyun = source["yinyun"];
	        this.reasons = source["reasons"];
	        this.total_score = source["total_score"];
	        this.wuxing_score = source["wuxing_score"];
	        this.yinyun_score = source["yinyun_score"];
	        this.meaning_score = source["meaning_score"];
	        this.sancai_score = source["sancai_score"];
	        this.zodiac_score = source["zodiac_score"];
	        this.nayin_score = source["nayin_score"];
	        this.novelty_score = source["novelty_score"];
	        this.bigram_score = source["bigram_score"];
	        this.frequency_score = source["frequency_score"];
	        this.sancai_analysis = source["sancai_analysis"];
	        this.hexagram = source["hexagram"];
	        this.hexagram_meaning = source["hexagram_meaning"];
	        this.hexagram_detail = this.convertValues(source["hexagram_detail"], yijing.PlumBlossomResult);
	        this.tiyong = this.convertValues(source["tiyong"], yijing.TiYongAnalysis);
	        this.score_detail = this.convertValues(source["score_detail"], ScoreDetailItem);
	        this.jyutping = source["jyutping"];
	        this.cantonese_score = source["cantonese_score"];
	        this.cantonese_score_detail = this.convertValues(source["cantonese_score_detail"], ScoreDetailItem);
	        this.risks = this.convertValues(source["risks"], fate.RiskItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace namestat {
	
	export class NameStat {
	    name: string;
	    count: number;
	    rate: number;
	    rank: number;
	
	    static createFrom(source: any = {}) {
	        return new NameStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.count = source["count"];
	        this.rate = source["rate"];
	        this.rank = source["rank"];
	    }
	}

}

export namespace services {
	
	export class EvaluateRequest {
	    surname: string;
	    given_name: string;
	    gender: string;
	    birth_year: number;
	    birth_month: number;
	    birth_day: number;
	    birth_hour: number;
	    birth_minute: number;
	    birth_location: string;
	    birth_longitude: number;
	
	    static createFrom(source: any = {}) {
	        return new EvaluateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.surname = source["surname"];
	        this.given_name = source["given_name"];
	        this.gender = source["gender"];
	        this.birth_year = source["birth_year"];
	        this.birth_month = source["birth_month"];
	        this.birth_day = source["birth_day"];
	        this.birth_hour = source["birth_hour"];
	        this.birth_minute = source["birth_minute"];
	        this.birth_location = source["birth_location"];
	        this.birth_longitude = source["birth_longitude"];
	    }
	}
	export class EvaluateResponse {
	    full_name: string;
	    name: name.Name;
	    bazi: bazi.BaziAnalysis;
	    zodiac: string;
	    risks: fate.RiskItem[];
	    risk_level: string;
	    hexagram?: yijing.Hexagram;
	    hexagram_interpretation?: yijing.HexagramInterpretation;
	    tiyong?: yijing.TiYongAnalysis;
	
	    static createFrom(source: any = {}) {
	        return new EvaluateResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.full_name = source["full_name"];
	        this.name = this.convertValues(source["name"], name.Name);
	        this.bazi = this.convertValues(source["bazi"], bazi.BaziAnalysis);
	        this.zodiac = source["zodiac"];
	        this.risks = this.convertValues(source["risks"], fate.RiskItem);
	        this.risk_level = source["risk_level"];
	        this.hexagram = this.convertValues(source["hexagram"], yijing.Hexagram);
	        this.hexagram_interpretation = this.convertValues(source["hexagram_interpretation"], yijing.HexagramInterpretation);
	        this.tiyong = this.convertValues(source["tiyong"], yijing.TiYongAnalysis);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExploreRequest {
	    generation_id: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ExploreRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generation_id = source["generation_id"];
	        this.count = source["count"];
	    }
	}
	export class ExploreResponse {
	    generation_id: string;
	    names: name.Name[];
	    remaining: number;
	
	    static createFrom(source: any = {}) {
	        return new ExploreResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generation_id = source["generation_id"];
	        this.names = this.convertValues(source["names"], name.Name);
	        this.remaining = source["remaining"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FavoriteRecord {
	    id: string;
	    surname: string;
	    given_name: string;
	    pinyin: string;
	    gender: string;
	    score: number;
	    source: string;
	    notes: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.surname = source["surname"];
	        this.given_name = source["given_name"];
	        this.pinyin = source["pinyin"];
	        this.gender = source["gender"];
	        this.score = source["score"];
	        this.source = source["source"];
	        this.notes = source["notes"];
	        this.created_at = this.convertValues(source["created_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GenerateRequest {
	    surname: string;
	    generation: string;
	    generation_position: string;
	    gender: string;
	    birth_year: number;
	    birth_month: number;
	    birth_day: number;
	    birth_hour: number;
	    birth_minute: number;
	    birth_location: string;
	    birth_longitude: number;
	    birth_type: string;
	    name_type: string;
	    preferences: string[];
	    name_length: number;
	    exclude_rare: boolean;
	    wuxing_match: string[];
	    source_classic: string;
	    min_strokes: number;
	    max_strokes: number;
	    include_poetry: boolean;
	    include_classic: boolean;
	    meaning_keywords: string[];
	    selected_chars: string[];
	    pinyin_initial: string;
	    enable_cantonese: boolean;
	    avoid_elder_names: string[];
	    min_frequency_tier: number;
	    max_frequency_tier: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.surname = source["surname"];
	        this.generation = source["generation"];
	        this.generation_position = source["generation_position"];
	        this.gender = source["gender"];
	        this.birth_year = source["birth_year"];
	        this.birth_month = source["birth_month"];
	        this.birth_day = source["birth_day"];
	        this.birth_hour = source["birth_hour"];
	        this.birth_minute = source["birth_minute"];
	        this.birth_location = source["birth_location"];
	        this.birth_longitude = source["birth_longitude"];
	        this.birth_type = source["birth_type"];
	        this.name_type = source["name_type"];
	        this.preferences = source["preferences"];
	        this.name_length = source["name_length"];
	        this.exclude_rare = source["exclude_rare"];
	        this.wuxing_match = source["wuxing_match"];
	        this.source_classic = source["source_classic"];
	        this.min_strokes = source["min_strokes"];
	        this.max_strokes = source["max_strokes"];
	        this.include_poetry = source["include_poetry"];
	        this.include_classic = source["include_classic"];
	        this.meaning_keywords = source["meaning_keywords"];
	        this.selected_chars = source["selected_chars"];
	        this.pinyin_initial = source["pinyin_initial"];
	        this.enable_cantonese = source["enable_cantonese"];
	        this.avoid_elder_names = source["avoid_elder_names"];
	        this.min_frequency_tier = source["min_frequency_tier"];
	        this.max_frequency_tier = source["max_frequency_tier"];
	    }
	}
	export class GenerateResponse {
	    bazi: bazi.BaziAnalysis;
	    nayin: string;
	    zodiac: string;
	    hexagram?: yijing.Hexagram;
	    hexagram_match?: yijing.HexagramMatch;
	    hexagram_detail?: yijing.PlumBlossomResult;
	    hexagram_interpretation?: yijing.HexagramInterpretation;
	    tiyong?: yijing.TiYongAnalysis;
	    ziwei?: ziwei.ZiweiChart;
	    names: name.Name[];
	    generation_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new GenerateResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bazi = this.convertValues(source["bazi"], bazi.BaziAnalysis);
	        this.nayin = source["nayin"];
	        this.zodiac = source["zodiac"];
	        this.hexagram = this.convertValues(source["hexagram"], yijing.Hexagram);
	        this.hexagram_match = this.convertValues(source["hexagram_match"], yijing.HexagramMatch);
	        this.hexagram_detail = this.convertValues(source["hexagram_detail"], yijing.PlumBlossomResult);
	        this.hexagram_interpretation = this.convertValues(source["hexagram_interpretation"], yijing.HexagramInterpretation);
	        this.tiyong = this.convertValues(source["tiyong"], yijing.TiYongAnalysis);
	        this.ziwei = this.convertValues(source["ziwei"], ziwei.ZiweiChart);
	        this.names = this.convertValues(source["names"], name.Name);
	        this.generation_id = source["generation_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HistoryRecord {
	    id: string;
	    surname: string;
	    gender: string;
	    birth_date: string;
	    birth_time: string;
	    birth_location: string;
	    results: number[];
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new HistoryRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.surname = source["surname"];
	        this.gender = source["gender"];
	        this.birth_date = source["birth_date"];
	        this.birth_time = source["birth_time"];
	        this.birth_location = source["birth_location"];
	        this.results = source["results"];
	        this.created_at = this.convertValues(source["created_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace yijing {
	
	export class Hexagram {
	    id: number;
	    name: string;
	    number: number;
	    symbol: string;
	    upper_trigram: number;
	    lower_trigram: number;
	    gua_ci: string;
	    xiang_ci: string;
	    yao_ci: string[];
	    tuan_ci: string;
	    xiao_xiang: string[];
	    interpretation: string;
	    gua_de?: string;
	
	    static createFrom(source: any = {}) {
	        return new Hexagram(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.number = source["number"];
	        this.symbol = source["symbol"];
	        this.upper_trigram = source["upper_trigram"];
	        this.lower_trigram = source["lower_trigram"];
	        this.gua_ci = source["gua_ci"];
	        this.xiang_ci = source["xiang_ci"];
	        this.yao_ci = source["yao_ci"];
	        this.tuan_ci = source["tuan_ci"];
	        this.xiao_xiang = source["xiao_xiang"];
	        this.interpretation = source["interpretation"];
	        this.gua_de = source["gua_de"];
	    }
	}
	export class HexagramInterpretation {
	    overall: string;
	    career: string;
	    love: string;
	    health: string;
	    fortune: string;
	    auspicious: string;
	    inauspicious: string;
	
	    static createFrom(source: any = {}) {
	        return new HexagramInterpretation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.overall = source["overall"];
	        this.career = source["career"];
	        this.love = source["love"];
	        this.health = source["health"];
	        this.fortune = source["fortune"];
	        this.auspicious = source["auspicious"];
	        this.inauspicious = source["inauspicious"];
	    }
	}
	export class HexagramMatch {
	    Hexagram?: Hexagram;
	    score: number;
	    reason: string;
	    wuxing_compat: string;
	
	    static createFrom(source: any = {}) {
	        return new HexagramMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Hexagram = this.convertValues(source["Hexagram"], Hexagram);
	        this.score = source["score"];
	        this.reason = source["reason"];
	        this.wuxing_compat = source["wuxing_compat"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PlumBlossomResult {
	    original_hexagram?: Hexagram;
	    changed_hexagram?: Hexagram;
	    inter_hexagram?: Hexagram;
	    upper_trigram_id: number;
	    lower_trigram_id: number;
	    upper_trigram_name: string;
	    lower_trigram_name: string;
	    moving_yao: number;
	    original_yao_lines: number[];
	    changed_yao_lines: number[];
	    inter_upper: number;
	    inter_lower: number;
	    interpretation: string;
	
	    static createFrom(source: any = {}) {
	        return new PlumBlossomResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.original_hexagram = this.convertValues(source["original_hexagram"], Hexagram);
	        this.changed_hexagram = this.convertValues(source["changed_hexagram"], Hexagram);
	        this.inter_hexagram = this.convertValues(source["inter_hexagram"], Hexagram);
	        this.upper_trigram_id = source["upper_trigram_id"];
	        this.lower_trigram_id = source["lower_trigram_id"];
	        this.upper_trigram_name = source["upper_trigram_name"];
	        this.lower_trigram_name = source["lower_trigram_name"];
	        this.moving_yao = source["moving_yao"];
	        this.original_yao_lines = source["original_yao_lines"];
	        this.changed_yao_lines = source["changed_yao_lines"];
	        this.inter_upper = source["inter_upper"];
	        this.inter_lower = source["inter_lower"];
	        this.interpretation = source["interpretation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TiYongAnalysis {
	    ti_gua: string;
	    yong_gua: string;
	    ti_wuxing: string;
	    yong_wuxing: string;
	    relation: string;
	    verdict: string;
	    wang_shuai: string;
	    xiyong_note: string;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new TiYongAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ti_gua = source["ti_gua"];
	        this.yong_gua = source["yong_gua"];
	        this.ti_wuxing = source["ti_wuxing"];
	        this.yong_wuxing = source["yong_wuxing"];
	        this.relation = source["relation"];
	        this.verdict = source["verdict"];
	        this.wang_shuai = source["wang_shuai"];
	        this.xiyong_note = source["xiyong_note"];
	        this.text = source["text"];
	    }
	}

}

export namespace ziwei {
	
	export class DaXianEntry {
	    range: number[];
	    heavenly_stem: string;
	    earthly_branch: string;
	
	    static createFrom(source: any = {}) {
	        return new DaXianEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.range = source["range"];
	        this.heavenly_stem = source["heavenly_stem"];
	        this.earthly_branch = source["earthly_branch"];
	    }
	}
	export class ZiweiChart {
	    year: number;
	    month: number;
	    day: number;
	    hour: number;
	    gender: string;
	    nian_zhu: string;
	    yue_zhu: string;
	    ri_zhu: string;
	    shi_zhu: string;
	    ming_gong: string;
	    shen_gong: string;
	    five_elements: string;
	    soul: string;
	    body: string;
	    da_xian: Record<string, DaXianEntry>;
	    zhu_xing: Record<string, Array<string>>;
	    fu_xing: Record<string, Array<string>>;
	    sui_xing: Record<string, Array<string>>;
	    si_hua: Record<string, string>;
	    brightness: Record<string, string>;
	    analysis: string;
	
	    static createFrom(source: any = {}) {
	        return new ZiweiChart(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.month = source["month"];
	        this.day = source["day"];
	        this.hour = source["hour"];
	        this.gender = source["gender"];
	        this.nian_zhu = source["nian_zhu"];
	        this.yue_zhu = source["yue_zhu"];
	        this.ri_zhu = source["ri_zhu"];
	        this.shi_zhu = source["shi_zhu"];
	        this.ming_gong = source["ming_gong"];
	        this.shen_gong = source["shen_gong"];
	        this.five_elements = source["five_elements"];
	        this.soul = source["soul"];
	        this.body = source["body"];
	        this.da_xian = this.convertValues(source["da_xian"], DaXianEntry, true);
	        this.zhu_xing = source["zhu_xing"];
	        this.fu_xing = source["fu_xing"];
	        this.sui_xing = source["sui_xing"];
	        this.si_hua = source["si_hua"];
	        this.brightness = source["brightness"];
	        this.analysis = source["analysis"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

