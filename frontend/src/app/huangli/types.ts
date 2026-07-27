// 黄历页面共享类型与常量

export interface HuangliHour {
  timeRange: string;
  jiXiong: string;
}

export interface LunarCalendarData {
  yearZhi?: string;
  dayNayin?: string;
  lunarYearName?: string;
  lunarMonthName?: string;
  lunarDayName?: string;
  dateStr?: string;
  lunarDate?: string;
  lunarMonth?: string;
  solarTerm?: string;
  [key: string]: unknown;
}

export interface HuangliData {
  chong?: string;
  sha?: string;
  liuYao?: string;
  zhiShen?: string;
  yi?: string[];
  ji?: string[];
  jiShen?: string[];
  jieShen?: string;
  hours?: HuangliHour[];
  xingXiu?: string;
  jianChu?: string;
  xiShen?: string;
  fuShen?: string;
  caiShen?: string;
  yangGui?: string;
  yinGui?: string;
  xiongSha?: string[];
  zhangSong?: string;
  taiShen?: string;
  pengZhu?: string;
  dayWuxing?: string;
}

export const zodiacIcons: Record<string, string> = {
  '鼠': '🐭', '牛': '🐮', '虎': '🐯', '兔': '🐰',
  '龙': '🐲', '蛇': '🐍', '马': '🐴', '羊': '🐑',
  '猴': '🐵', '鸡': '🐔', '狗': '🐶', '猪': '🐷'
};

export const getZodiacSign = (month: number, day: number): string => {
  const dates = [20, 19, 21, 20, 21, 22, 23, 23, 23, 24, 23, 22];
  const signs = ['水瓶座', '双鱼座', '白羊座', '金牛座', '双子座', '巨蟹座',
    '狮子座', '处女座', '天秤座', '天蝎座', '射手座', '摩羯座'];
  return day < dates[month - 1] ? signs[month - 1] : signs[month % 12];
};

export const tianGanDiZhi = ['甲子', '乙丑', '丙寅', '丁卯', '戊辰', '己巳', '庚午', '辛未', '壬申', '癸酉', '甲戌', '乙亥'];

export const shichenTime = ['23:00-01:00', '01:00-03:00', '03:00-05:00', '05:00-07:00', '07:00-09:00', '09:00-11:00',
  '11:00-13:00', '13:00-15:00', '15:00-17:00', '17:00-19:00', '19:00-21:00', '21:00-23:00'];
