import { getLunar } from 'chinese-lunar-calendar';

export interface LunarDate {
  year: number;
  month: number;
  day: number;
  isLeap: boolean;
}

export interface SolarDate {
  year: number;
  month: number;
  day: number;
}

const HEAVENLY_STEMS = ['甲', '乙', '丙', '丁', '戊', '己', '庚', '辛', '壬', '癸'];
const EARTHLY_BRANCHES = ['子', '丑', '寅', '卯', '辰', '巳', '午', '未', '申', '酉', '戌', '亥'];
const LUNAR_MONTHS = ['正', '二', '三', '四', '五', '六', '七', '八', '九', '十', '冬', '腊'];
const CHINESE_NUMBERS = ['一', '二', '三', '四', '五', '六', '七', '八', '九', '十'];

export function parseLunarYear(lunarYearStr: string): number {
  const stemIndex = HEAVENLY_STEMS.indexOf(lunarYearStr[0]);
  const branchIndex = EARTHLY_BRANCHES.indexOf(lunarYearStr[1]);
  
  if (stemIndex === -1 || branchIndex === -1) {
    throw new Error('Invalid lunar year string');
  }
  
  const year = 1900 + stemIndex;
  const branchOffset = (year - 4) % 12;
  const expectedBranchIndex = branchOffset >= 0 ? branchOffset : branchOffset + 12;
  
  const diff = branchIndex - expectedBranchIndex;
  return year + diff;
}

export function parseLunarMonth(dateStr: string): { month: number; isLeap: boolean } {
  const isLeap = dateStr.startsWith('闰');
  const monthStr = isLeap ? dateStr.substring(1, 2) : dateStr.substring(0, 1);
  const monthIndex = LUNAR_MONTHS.indexOf(monthStr);
  
  if (monthIndex === -1) {
    throw new Error('Invalid lunar month string');
  }
  
  return { month: monthIndex + 1, isLeap };
}

export function parseLunarDay(dateStr: string): number {
  const monthStr = dateStr.substring(0, 2);
  const dayStr = dateStr.substring(2);
  
  if (dayStr.startsWith('初')) {
    return CHINESE_NUMBERS.indexOf(dayStr[1]) + 1;
  } else if (dayStr.startsWith('十')) {
    if (dayStr.length === 2) {
      return 10;
    }
    return 10 + CHINESE_NUMBERS.indexOf(dayStr[1]);
  } else if (dayStr.startsWith('廿')) {
    if (dayStr === '廿十') {
      return 20;
    }
    return 20 + CHINESE_NUMBERS.indexOf(dayStr[1]);
  } else if (dayStr === '三十') {
    return 30;
  }
  
  throw new Error('Invalid lunar day string');
}

export function parseLunarDateStr(dateStr: string): LunarDate {
  const monthInfo = parseLunarMonth(dateStr);
  const day = parseLunarDay(dateStr);
  
  return {
    year: 0,
    month: monthInfo.month,
    day,
    isLeap: monthInfo.isLeap
  };
}

export function lunarToSolar(lunarYear: string, lunarMonthStr: string, lunarDayStr: string): SolarDate {
  const lunarYearNum = parseLunarYear(lunarYear);
  const lunarDate = parseLunarDateStr(lunarMonthStr + lunarDayStr);
  lunarDate.year = lunarYearNum;
  
  let year = lunarYearNum - 1;
  let found = false;
  
  for (let offset = 0; offset <= 2; offset++) {
    const testYear = lunarYearNum + offset;
    const testDate = new Date(testYear, 0, 1);
    
    for (let day = 0; day < 366; day++) {
      const currentDate = new Date(testDate.getTime() + day * 24 * 60 * 60 * 1000);
      const lunar = getLunar(currentDate.getFullYear(), currentDate.getMonth() + 1, currentDate.getDate());
      
      const parsedLunar = parseLunarDateStr(lunar.dateStr);
      const parsedYear = parseLunarYear(lunar.lunarYear);
      
      if (parsedYear === lunarYearNum && 
          parsedLunar.month === lunarDate.month && 
          parsedLunar.day === lunarDate.day &&
          parsedLunar.isLeap === lunarDate.isLeap) {
        return {
          year: currentDate.getFullYear(),
          month: currentDate.getMonth() + 1,
          day: currentDate.getDate()
        };
      }
    }
  }
  
  throw new Error('Cannot convert lunar date to solar date');
}

export function getChineseHour(hour: number): string {
  const chineseHours = ['子时', '丑时', '寅时', '卯时', '辰时', '巳时', '午时', '未时', '申时', '酉时', '戌时', '亥时'];
  const chineseHourIndex = Math.floor((hour + 1) / 2) % 12;
  return chineseHours[chineseHourIndex];
}

export function formatLunarDate(date: Date): string {
  const lunar = getLunar(date.getFullYear(), date.getMonth() + 1, date.getDate());
  const chineseHour = getChineseHour(date.getHours());
  return `农历:${lunar.lunarYear}${lunar.dateStr} ${chineseHour}${date.getMinutes()}分`;
}

export function formatSolarDate(date: Date): string {
  return `${date.getFullYear()}-${(date.getMonth() + 1).toString().padStart(2, '0')}-${date.getDate().toString().padStart(2, '0')} ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
}
